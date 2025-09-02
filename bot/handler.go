package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
	ui "trackerbot/keyboard"
	"trackerbot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type inputType int

const (
	inputTypeDaily inputType = iota
	inputTypeMaxReps
	inputTypeCustomNorm
)

type pendingInput struct {
	expiry    time.Time
	inputType inputType
	messageID int
}

type BotHandler struct {
	bot           *tgbotapi.BotAPI
	service       *service.PushupService
	pendingInputs sync.Map
}

const inputTimeout = 2 * time.Minute

func NewBotHandler(bot *tgbotapi.BotAPI, service *service.PushupService) *BotHandler {
	return &BotHandler{bot: bot, service: service}
}

func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	// Игнорируем все обновления, кроме сообщений
	if update.Message == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	username := update.Message.From.UserName
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	// Сначала проверяем, есть ли ожидаемый ввод для этого чата
	if input, ok := h.getPendingInput(chatID); ok {
		if time.Now().After(input.expiry) {
			h.clearPendingInput(chatID)
			msg := tgbotapi.NewMessage(chatID, "Ввод отменен по таймауту.")
			notificationsEnabled, _ := h.service.GetNotificationsStatus(ctx, userID)
			msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
			h.bot.Send(msg)
			return
		}

		// Обрабатываем ввод числа в режиме ожидания
		if count, err := strconv.Atoi(update.Message.Text); err == nil {
			h.clearPendingInput(chatID)
			
			// Получаем статус уведомлений для клавиатуры
			notificationsEnabled, err := h.service.GetNotificationsStatus(ctx, userID)
			if err != nil {
				log.Printf("Ошибка получения статуса уведомлений: %v", err)
				notificationsEnabled = true
			}
			
			switch input.inputType {
			case inputTypeDaily:
				h.handleAddPushups(ctx, userID, username, chatID, count, inputTypeDaily, notificationsEnabled)
			case inputTypeMaxReps:
				h.handleAddPushups(ctx, userID, username, chatID, count, inputTypeMaxReps, notificationsEnabled)
			case inputTypeCustomNorm:
				h.handleSetCustomNorm(ctx, userID, chatID, count, notificationsEnabled)
			}
			return
		}

		// Если введено не число, просим повторить ввод (в reply на предыдущее сообщение)
		replyMsg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите число:")
		replyMsg.ReplyToMessageID = input.messageID
		h.bot.Send(replyMsg)
		return
	}

	// Если нет ожидаемого ввода, обрабатываем только команды/кнопки
	notificationsEnabled, err := h.service.GetNotificationsStatus(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения статуса уведомлений: %v", err)
		notificationsEnabled = true
	}

	// Обработка команды /reset
	if update.Message.Text == "🔄 Сброс" {
		if err := h.service.ResetMaxReps(ctx, userID); err != nil {
			log.Printf("Ошибка сброса max_reps: %v", err)
			msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при сбросе. Попробуйте позже.")
			h.bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(chatID, "✅ Дневная норма сброшена до значения по умолчанию (40)")
		msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
		h.bot.Send(msg)
		return
	}

	// Обработка команд и кнопок
	switch update.Message.Text {
	case "/start":
		h.handleStart(ctx, chatID, userID, notificationsEnabled)
	case "Добавить отжимания":
		h.requestPushupCount(chatID, inputTypeDaily)
	case "🎯 Определить норму":
		h.requestPushupCount(chatID, inputTypeMaxReps)
	case "📝 Установить норму":
		h.requestCustomNorm(chatID)
	case "📊 Статистика":
		h.handleTodayStat(ctx, userID, chatID, notificationsEnabled)
		h.handleTotalStat(ctx, userID, chatID, notificationsEnabled)
		h.handleTodayLeaderboard(ctx, chatID)
	case "🔕 Отключить напоминания":
		h.handleToggleNotifications(ctx, userID, chatID, false)
	case "🔔 Включить напоминания":
		h.handleToggleNotifications(ctx, userID, chatID, true)
	case "🛠️ Настройки":
		msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
		msg.ReplyMarkup = ui.SettingsKeyboard(notificationsEnabled)
		h.bot.Send(msg)
	case "⬅️ Назад":
		msg := tgbotapi.NewMessage(chatID, "Главное меню:")
		msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
		h.bot.Send(msg)
	default:
		// Игнорируем обычные текстовые сообщения, которые не являются командами
		if strings.HasPrefix(update.Message.Text, "/") {
			msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте меню.")
			msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
			h.bot.Send(msg)
		}
		// Для обычных текстовых сообщений без префикса "/" ничего не делаем
	}
}


func (h *BotHandler) handleAddPushups(ctx context.Context, userID int64, username string, chatID int64, count int, inputType inputType, notEnable bool) {
	if count <= 0 {
		// Получаем текущий input для получения messageID
		if input, ok := h.getPendingInput(chatID); ok {
			msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число:")
			// Используем ForceReply для продолжения режима Reply
			msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
			msg.ReplyToMessageID = input.messageID
			sentMsg, err := h.bot.Send(msg)
			if err != nil {
				log.Printf("Ошибка отправки сообщения: %v", err)
				return
			}
			// Обновляем messageID для продолжения цепочки reply
			h.setPendingInput(chatID, input.inputType, time.Now().Add(inputTimeout), sentMsg.MessageID)
		}
		return
	}

	isMaxReps := inputType == inputTypeMaxReps
	result, err := h.service.AddPushups(ctx, userID, username, count, isMaxReps)
	if err != nil {
		log.Printf("Ошибка при добавлении отжиманий: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	log.Printf("Username%s UserID %d added %d pushups", username, userID, count)

	var response string
	if inputType == inputTypeMaxReps {
		response = fmt.Sprintf("🔔Твоя дневная норма установлена: %d\n", result.DailyNorm)
	}

	response += fmt.Sprintf("✅Добавлено: %d отжиманий!\n📈Твой прогресс: %d/%d\n", count, result.TotalToday, result.DailyNorm)

	// Проверяем выполнение нормы через кеш
	hasCompleted, firstCompleter := h.service.CheckNormCompletion(result.DailyNorm)

	var responseFirstCompleter string
	
	if !hasCompleted {
		responseFirstCompleter = "❌ Никто еще не выполнил норму сегодня.\nМожет, ты будешь первым? 💪\n\n"
	} else {
		responseFirstCompleter = fmt.Sprintf("🎯 %s уже выполнил норму!\nА ты не отставай, присоединяйся! 🚀\n\n", firstCompleter)
	}

	if result.TotalToday >= result.DailyNorm {
		response += "\n🎯Ты выполнил дневную норму!\n"
		responseFirstCompleter = ""
	}

	msg := tgbotapi.NewMessage(chatID, response+responseFirstCompleter)
	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
	h.bot.Send(msg)
}

func (h *BotHandler) handleTodayStat(ctx context.Context, userID int64, chatID int64, notEnable bool) {
	total, err := h.service.GetTodayStat(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	dailyNorm, err := h.service.GetDailyNorm(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении дневной нормы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	daylyStatText :=  fmt.Sprintf("📊Сегодня ты отжался %d/%d %s.\n%s\n", total, dailyNorm, formatTimesWord(total), generateProgressBar(total, dailyNorm, 10))

	msg := tgbotapi.NewMessage(chatID, daylyStatText)
	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
	h.bot.Send(msg)
}

func generateProgressBar(current, total, barWidth int) string {
	if total <= 0 || barWidth <= 0 {
		return "Прогресс: [не определён]"
	}

	percentage := float64(current) / float64(total)
	clamped := percentage
	if clamped > 1 {
		clamped = 1
	}

	filled := int(clamped * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}
	empty := barWidth - filled

	bar := strings.Repeat("🔋", filled) + strings.Repeat("🪫", empty) // или  ░ ▒ ▓ █
	percentText := int(percentage * 100)

	// Добавим бонусную метку если перевыполнил
	suffix := ""
	if percentage > 1 {
		suffix = " 🏆"
	}

	return fmt.Sprintf("Прогресс: [%s] %d%%%s", bar, percentText, suffix)
}

func (h *BotHandler) handleTotalStat(ctx context.Context, userID int64, chatID int64, notEnable bool) {
	total, err := h.service.GetTotalStat(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	statText := fmt.Sprintf("💪За все время ты отжался: %d %s\n", total, formatTimesWord(total))
	var FirstWorkoutDateText string

	firstWorkoutDate, err := h.service.GetFirstWorkoutDate(ctx, userID)
	if err != nil || firstWorkoutDate == "" {
		FirstWorkoutDateText = "Пользователь еще не начинал тренироваться"
	} 
	FirstWorkoutDateText = fmt.Sprintf("Первая тренировка: %s", firstWorkoutDate)
	
	msg := tgbotapi.NewMessage(chatID, statText+FirstWorkoutDateText)
	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
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

func (h *BotHandler) handleStart(ctx context.Context, chatID int64, userID int64, notEnable bool) {

	maxReps, err := h.service.GetUserMaxReps(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения данных: %v", err)
		return
	}

	if maxReps == 0 {
		msg := tgbotapi.NewMessage(chatID, "Необходимо определить твою дневную норму!")
		h.bot.Send(msg)
		h.requestPushupCount(chatID, inputTypeMaxReps)
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Выберите действие:")

	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
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
	var messageText string
	if inputType == inputTypeDaily {
		messageText = "Введите количество отжиманий:"
	} else {
		messageText = "Введите максимальное количество отжиманий за один подход:"
	}

	// Сначала отправляем сообщение
	msg := tgbotapi.NewMessage(chatID, messageText)
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}

	// Сохраняем ID отправленного сообщения для reply
	h.setPendingInput(chatID, inputType, time.Now().Add(inputTimeout), sentMsg.MessageID)
}

// Добавим новую функцию для обработки установки дневной нормы
func (h *BotHandler) handleSetCustomNorm(ctx context.Context, userID int64, chatID int64, dailyNorm int, notEnable bool) {
	if dailyNorm <= 0 {
		
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число:")
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
		sentMsg, err := h.bot.Send(msg)
	
		if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}
		h.setPendingInput(chatID, inputTypeCustomNorm, time.Now().Add(inputTimeout), sentMsg.MessageID)
		h.bot.Send(msg)
		return
	}
	err := h.service.SetDailyNorm(ctx, userID, dailyNorm)
	if err != nil {
		log.Printf("Ошибка при установке дневной нормы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Дневная норма установлена: %d", dailyNorm))
	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
	h.bot.Send(msg)
}

func (h *BotHandler) requestCustomNorm(chatID int64) {
	// Сначала отправляем сообщение
	msg := tgbotapi.NewMessage(chatID, "Введите дневную норму отжиманий:")
	msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: true}
	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}

	// Сохраняем ID отправленного сообщения для reply
	h.setPendingInput(chatID, inputTypeCustomNorm, time.Now().Add(inputTimeout), sentMsg.MessageID)
}

func (h *BotHandler) setPendingInput(chatID int64, inputType inputType, expiry time.Time, messageID int) {
	h.pendingInputs.Store(chatID, pendingInput{
		expiry:    expiry,
		inputType: inputType,
		messageID: messageID,
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

				ctx := context.Background()
				userID := chatID
				notificationsEnabled, err := h.service.GetNotificationsStatus(ctx, userID)
				if err != nil {
					log.Printf("Ошибка получения статуса уведомленийctx: %v", err)
					notificationsEnabled = true
				}

				msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
				if _, err := h.bot.Send(msg); err != nil {
					log.Printf("Ошибка отправки сообщения об отмене: %v", err)
				}
				log.Printf("Отменен ввод для чата %d по таймауту", chatID)
			}
			return true
		})
	}
}

// Добавляем новый метод для переключения напоминаний
func (h *BotHandler) handleToggleNotifications(ctx context.Context, userID int64, chatID int64, enable bool) {
	var err error
	var message string

	if enable {
		err = h.service.EnableNotifications(ctx, userID)
		message = "🔔 Напоминания включены! Буду напоминать о тренировках."
	} else {
		err = h.service.DisableNotifications(ctx, userID)
		message = "🔕 Напоминания отключены. Не забывай тренироваться самостоятельно! 💪"
	}

	if err != nil {
		log.Printf("Ошибка переключения напоминаний: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Ошибка изменения настроек уведомлений")
		msg.ReplyMarkup = ui.MainKeyboard(enable)
		h.bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ReplyMarkup = ui.MainKeyboard(enable)
	h.bot.Send(msg)
}
