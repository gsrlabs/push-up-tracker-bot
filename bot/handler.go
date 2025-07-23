// Пакет bot содержит логику обработки Telegram-команд и взаимодействия с пользователем
package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"trackerbot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BotHandler обрабатывает входящие сообщения Telegram и управляет взаимодействием с пользователем
type BotHandler struct {
	bot     *tgbotapi.BotAPI       // Клиент Telegram Bot API
	service *service.PushupService // Сервис для работы с отжиманиями
	pendingInputs sync.Map
}

const inputTimeout = 10 * time.Second

// NewBotHandler создает новый экземпляр обработчика бота
// Принимает:
// - bot: клиент Telegram API
// - service: сервис для работы с данными отжиманий
// Возвращает:
// - указатель на созданный BotHandler
func NewBotHandler(bot *tgbotapi.BotAPI, service *service.PushupService) *BotHandler {
	return &BotHandler{bot: bot, service: service}
}

// HandleUpdate обрабатывает входящее обновление от Telegram
// Определяет тип команды и делегирует обработку соответствующему методу
func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	// Создаем контекст с таймаутом 2 секунды для обработки запроса
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // Гарантированное освобождение ресурсов

	// Игнорируем обновления, не содержащие сообщение
	if update.Message == nil {
		return
	}

	// Извлекаем идентификаторы пользователя и чата
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	// Проверяем не истекло ли время ожидания ввода
    if expiry, ok := h.getPendingInput(chatID); ok {
        if time.Now().After(expiry) {
            // Время истекло, но cleanup еще не сработал
            h.clearPendingInput(chatID)
            msg := tgbotapi.NewMessage(chatID, "Ввод отменен по таймауту.")
            msg.ReplyMarkup = mainKeyboard()
            h.bot.Send(msg)
            return
        }

        // Пытаемся обработать как число
        if count, err := strconv.Atoi(update.Message.Text); err == nil {
            h.clearPendingInput(chatID)
            h.handleAddPushups(ctx, userID, chatID, count)
            return
        }
        
        // Не число - сообщаем об ошибке
        msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите число:")
        h.bot.Send(msg)
        return
    }

	// Маршрутизация команд
	switch update.Message.Text {
	case "/start":
		h.handleStart(chatID) // Обработка команды /start
	case "Добавить отжимания":
		h.requestPushupCount(chatID) // Запрос количества отжиманий
	case "Статистика за сегодня":
		h.handleTodayStat(ctx, userID, chatID) // Статистика за сегодня
	case "Статистика за всё время":
		h.handleTotalStat(ctx, userID, chatID) // Общая статистика
	default:
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте меню.")
        msg.ReplyMarkup = mainKeyboard()
        h.bot.Send(msg)
	}
}

// handleAddPushups обрабатывает добавление новых отжиманий
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
// - chatID: идентификатор чата
// - count: количество отжиманий для добавления
func (h *BotHandler) handleAddPushups(ctx context.Context, userID int64, chatID int64, count int) {

	// Валидацию ввода

	if count <= 0 {
		h.setPendingInput(chatID, time.Now().Add(inputTimeout))
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число:")
		h.bot.Send(msg)
		return
	}

	// Вызываем сервис для добавления отжиманий
	response, err := h.service.AddPushups(ctx, userID, count)
	if err != nil {
		log.Printf("Ошибка при добавлении отжиманий: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	// Логирование действий пользователя:
	log.Printf("User %d added %d pushups", userID, count)

	// Отправляем пользователю результат операции
	msg := tgbotapi.NewMessage(chatID, response)
	msg.ReplyMarkup = mainKeyboard() // Восстанавливаем основную клавиатуру
	h.bot.Send(msg)

}

// handleTodayStat обрабатывает запрос статистики за сегодня
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
// - chatID: идентификатор чата
func (h *BotHandler) handleTodayStat(ctx context.Context, userID int64, chatID int64) {
	// Получаем статистику через сервис
	total, err := h.service.GetTodayStat(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	// Формируем и отправляем ответ
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Сегодня вы отжались %d раз.", total))
	msg.ReplyMarkup = mainKeyboard()
	h.bot.Send(msg)
}

// handleStart обрабатывает команду /start - приветствие и основное меню
// Параметры:
// - chatID: идентификатор чата
func (h *BotHandler) handleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
	msg.ReplyMarkup = mainKeyboard() // Показываем основную клавиатуру
	h.bot.Send(msg)
}

// handleTotalStat обрабатывает запрос общей статистики
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
// - chatID: идентификатор чата
func (h *BotHandler) handleTotalStat(ctx context.Context, userID int64, chatID int64) {
	// Заглушка для реализации
	msg := tgbotapi.NewMessage(chatID, "Вывод общей статистики")
	h.bot.Send(msg)
}

// requestPushupCount запрашивает у пользователя количество отжиманий
// Параметры:
// - chatID: идентификатор чата
func (h *BotHandler) requestPushupCount(chatID int64) {
	// Устанавливаем ожидание ввода на 2 минуты
    h.setPendingInput(chatID, time.Now().Add(inputTimeout))
    
    msg := tgbotapi.NewMessage(chatID, "Введите количество отжиманий:")
    msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
    h.bot.Send(msg)
}

// setPendingInput сохраняет время ожидания ввода для указанного чата
func (h *BotHandler) setPendingInput(chatID int64, expiry time.Time) {
    h.pendingInputs.Store(chatID, expiry)
}

// getPendingInput возвращает время истечения ожидания ввода для чата
func (h *BotHandler) getPendingInput(chatID int64) (time.Time, bool) {
    value, ok := h.pendingInputs.Load(chatID)
    if !ok {
        return time.Time{}, false
    }
    return value.(time.Time), true
}

// clearPendingInput удаляет состояние ожидания для указанного чата
func (h *BotHandler) clearPendingInput(chatID int64) {
    h.pendingInputs.Delete(chatID)
}

// cleanupExpiredInputs периодически очищает устаревшие ожидания
func (h *BotHandler) CleanupExpiredInputs() {
    for {
        time.Sleep(1 * time.Second) // Проверяем чаще - каждую секунду
        
        now := time.Now()
        h.pendingInputs.Range(func(key, value interface{}) bool {
            expiry := value.(time.Time)
            if now.After(expiry) {
                chatID := key.(int64)
                h.pendingInputs.Delete(key)
                
                // Отправляем сообщение об отмене
                msg := tgbotapi.NewMessage(chatID, "⌛ Ввод отменен по таймауту (10 секунд).")
                msg.ReplyMarkup = mainKeyboard()
                if _, err := h.bot.Send(msg); err != nil {
                    log.Printf("Ошибка отправки сообщения об отмене: %v", err)
                }
                log.Printf("Отменен ввод для чата %d по таймауту", chatID)
            }
            return true
        })
    }
}