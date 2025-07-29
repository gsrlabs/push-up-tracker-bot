package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
	"trackerbot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type inputType int

const (
	inputTypeDaily inputType = iota
	inputTypeMaxReps
)

type pendingInput struct {
	expiry    time.Time
	inputType inputType
}

type BotHandler struct {
	bot           *tgbotapi.BotAPI
	service       *service.PushupService
	pendingInputs sync.Map
}

const inputTimeout = 10 * time.Second

func NewBotHandler(bot *tgbotapi.BotAPI, service *service.PushupService) *BotHandler {
	return &BotHandler{bot: bot, service: service}
}

func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if update.Message == nil {
		return
	}

	username := update.Message.From.UserName
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	if input, ok := h.getPendingInput(chatID); ok {
		if time.Now().After(input.expiry) {
			h.clearPendingInput(chatID)
			msg := tgbotapi.NewMessage(chatID, "Ввод отменен по таймауту.")
			msg.ReplyMarkup = mainKeyboard()
			h.bot.Send(msg)
			return
		}

		if count, err := strconv.Atoi(update.Message.Text); err == nil {
			h.clearPendingInput(chatID)
			if input.inputType == inputTypeDaily {
				h.handleAddPushups(ctx, userID, username, chatID, count, inputTypeDaily)
			} else if input.inputType == inputTypeMaxReps {
				h.handleAddPushups(ctx, userID, username, chatID, count, inputTypeMaxReps)
			}
			return
		}

		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите число:")
		h.bot.Send(msg)
		return
	}

	switch update.Message.Text {
	case "/start":
		h.handleStart(chatID)
	case "Добавить отжимания за день":
		h.requestPushupCount(chatID, inputTypeDaily)
	case "Добавить отжимания за раз":
		h.requestPushupCount(chatID, inputTypeMaxReps)
	case "Статистика за сегодня":
		h.handleTodayStat(ctx, userID, chatID)
		h.handleTodayLeaderboard(ctx, chatID)
	default:
		msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте меню.")
		msg.ReplyMarkup = mainKeyboard()
		h.bot.Send(msg)
	}
}

func (h *BotHandler) handleAddPushups(ctx context.Context, userID int64, username string, chatID int64, count int, inputType inputType) {
	if count <= 0 {
		h.setPendingInput(chatID, inputType, time.Now().Add(inputTimeout))
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число:")
		h.bot.Send(msg)
		return
	}

	result, err := h.service.AddPushups(ctx, userID, username, count)
	if err != nil {
		log.Printf("Ошибка при добавлении отжиманий: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	log.Printf("Username%s UserID %d added %d pushups", username, userID, count)

	var response string
	if inputType == inputTypeMaxReps {
		response = fmt.Sprintf("🔔Ваша дневная норма составляет: %d\n", result.DailyNorm)
	}

	response += fmt.Sprintf("✅Добавлено: %d отжиманий\n📊Ваш прогресс: %d/%d", count, result.TotalToday, result.DailyNorm)

	if result.TotalToday >= result.DailyNorm {
		response += "\n🎯Вы выполнили дневную норму!"
	}

	msg := tgbotapi.NewMessage(chatID, response)
	msg.ReplyMarkup = mainKeyboard()
	h.bot.Send(msg)
}

func (h *BotHandler) handleTodayStat(ctx context.Context, userID int64, chatID int64) {
	total, err := h.service.GetTodayStat(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	maxReps, err := h.service.GetUserMaxReps(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении max_reps: %v", err)
	}

	dailyNorm := service.CalculateDailyNorm(maxReps)
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Сегодня вы отжались %d %s.\nДневная норма: %d/%d", total,formatTimesWord(total), total, dailyNorm))
	msg.ReplyMarkup = mainKeyboard()
	h.bot.Send(msg)
}

func formatTimesWord(n int) string {
	n = n % 100 // учитываем "11–14"
	if n >= 11 && n <= 14 {
		return "раз"
	}

	switch n % 10 {
	case 1:
		return "раз"
	case 2, 3, 4:
		return "раза"
	default:
		return "раз"
	}
}


func (h *BotHandler) handleStart(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
	msg.ReplyMarkup = mainKeyboard()
	h.bot.Send(msg)
}

func (h *BotHandler) handleTodayLeaderboard(ctx context.Context, chatID int64) {
	leaderboard, err := h.service.GetTodayLeaderboard(ctx)
	if err != nil {
		log.Printf("Ошибка получения рейтинга: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка загрузки рейтинга")
		h.bot.Send(msg)
		return
	}

	var response strings.Builder
	response.WriteString("🏆 Статистика за сегодня:\n\n")
	for _, item := range leaderboard {
		response.WriteString(fmt.Sprintf("%d. %s: %d\n", item.Rank, item.Username, item.Count))
	}

	msg := tgbotapi.NewMessage(chatID, response.String())
	h.bot.Send(msg)
}

func (h *BotHandler) requestPushupCount(chatID int64, inputType inputType) {
	h.setPendingInput(chatID, inputType, time.Now().Add(inputTimeout))

	var messageText string
	if inputType == inputTypeDaily {
		messageText = "Введите количество отжиманий:"
	} else {
		messageText = "Введите максимальное количество отжиманий за один подход:"
	}

	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	h.bot.Send(msg)
}

func (h *BotHandler) setPendingInput(chatID int64, inputType inputType, expiry time.Time) {
	h.pendingInputs.Store(chatID, pendingInput{
		expiry:    expiry,
		inputType: inputType,
	})
}

func (h *BotHandler) getPendingInput(chatID int64) (pendingInput, bool) {
	value, ok := h.pendingInputs.Load(chatID)
	if !ok {
		return pendingInput{}, false
	}
	return value.(pendingInput), true
}

func (h *BotHandler) clearPendingInput(chatID int64) {
	h.pendingInputs.Delete(chatID)
}

func (h *BotHandler) CleanupExpiredInputs() {
	for {
		time.Sleep(1 * time.Second)

		now := time.Now()
		h.pendingInputs.Range(func(key, value interface{}) bool {
			input := value.(pendingInput)
			if now.After(input.expiry) {
				chatID := key.(int64)
				h.pendingInputs.Delete(key)

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