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

	// Обработка команды /reset
	if update.Message.Text == "Сброс" {

		if err := h.service.ResetMaxReps(ctx, userID); err != nil {
			log.Printf("Ошибка сброса max_reps: %v", err)
			msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при сбросе. Попробуйте позже.")
			h.bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(chatID, "✅ Дневная норма сброшена до значения по умолчанию (40)")
		msg.ReplyMarkup = mainKeyboard()
		h.bot.Send(msg)
		return
	}

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
			switch input.inputType {
			case inputTypeDaily:
				h.handleAddPushups(ctx, userID, username, chatID, count, inputTypeDaily)
			case inputTypeMaxReps:
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
		h.handleStart(ctx, userID, chatID)
	case "+ за день":
		h.requestPushupCount(chatID, inputTypeDaily)
	case "+ за один подход":
		h.requestPushupCount(chatID, inputTypeMaxReps)
	case "Статистика":
		h.handleTodayStat(ctx, userID, chatID)
		h.handleTotalStat(ctx, userID, chatID)
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

	isMaxReps := (inputType == inputTypeMaxReps)
	result, err := h.service.AddPushups(ctx, userID, username, count, isMaxReps)
	if err != nil {
		log.Printf("Ошибка при добавлении отжиманий: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	log.Printf("Username%s UserID %d added %d pushups", username, userID, count)

	maxReps, err := h.service.GetUserMaxReps(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения данных: %v", err)
		return
	}

	var response string
	if inputType == inputTypeMaxReps && count >= maxReps {
		response = fmt.Sprintf("🔔Твоя дневная норма составляет: %d\n", result.DailyNorm)
	}

	response += fmt.Sprintf("✅Добавлено: %d отжиманий!\n📈Товой прогресс: %d/%d", count, result.TotalToday, result.DailyNorm)

	if result.TotalToday >= result.DailyNorm {
		response += "\n🎯Ты выполнил дневную норму!"
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
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📊Сегодня ты отжался %d/%d %s.\n%s\n", total, dailyNorm, formatTimesWord(total), generateProgressBar(total, dailyNorm, 10)))
	msg.ReplyMarkup = mainKeyboard()
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

func (h *BotHandler) handleTotalStat(ctx context.Context, userID int64, chatID int64) {
	total, err := h.service.GetTotalStat(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("💪За все время ты отжался: %d %s", total, formatTimesWord(total)))
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

func (h *BotHandler) handleStart(ctx context.Context, chatID int64, userID int64) {

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
