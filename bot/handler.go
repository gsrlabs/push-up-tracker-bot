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
	perDayLimit inputType = iota
	inputTypeMaxReps
	inputTypeCustomNorm
)
const (
	oneTimeEntryLimit = 1000
	maxRepsLimit = 500
	castomDailyNormLimit = 500
)
type pendingInput struct {
	inputType   inputType
	messageID   int
	cancelMsgID int // ID сообщения с кнопкой отмены
}

type BotHandler struct {
	bot           *tgbotapi.BotAPI
	service       *service.PushupService
	pendingInputs sync.Map
	adminIDs      map[int64]bool
}

func NewBotHandler(bot *tgbotapi.BotAPI, service *service.PushupService) *BotHandler {
	return &BotHandler{bot: bot, service: service, adminIDs: map[int64]bool{
		1036193976: true, // user_id
	}}
}

func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Data == "cancel_input" {
			chatID := update.CallbackQuery.Message.Chat.ID
			h.clearPendingInput(chatID)

			// Ответ на callback
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "Ввод отменен")
			h.bot.Request(callback)

			// Убираем inline-кнопку из сообщения
			editMsg := tgbotapi.NewEditMessageText(chatID, update.CallbackQuery.Message.MessageID, "Ввод отменен")
			h.bot.Send(editMsg)

			// Отправляем новое сообщение с главной клавиатурой
			notificationsEnabled, _ := h.service.GetNotificationsStatus(context.Background(), update.CallbackQuery.From.ID)
			msg := tgbotapi.NewMessage(chatID, "Ввод отменен")
			msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
			h.bot.Send(msg)

			return
		}
	}

	if update.Message == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	username := update.Message.From.UserName
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Проверяем, есть ли ожидаемый ввод
	if input, ok := h.getPendingInput(chatID); ok {
		// Пытаемся распарсить число
		if count, err := strconv.Atoi(text); err == nil {
			// Успех — очищаем ожидание и обрабатываем
			h.clearPendingInput(chatID)
			notificationsEnabled, err := h.service.GetNotificationsStatus(ctx, userID)
			if err != nil {
				log.Printf("Ошибка получения статуса уведомлений: %v", err)
				notificationsEnabled = true

			}

			switch input.inputType {
			case perDayLimit:
				h.handleAddPushups(ctx, userID, username, chatID, count, notificationsEnabled)
			case inputTypeMaxReps:
				h.handleSetMaxReps(ctx, userID, username, chatID, count, notificationsEnabled)
			case inputTypeCustomNorm:
				h.handleSetCustomNorm(ctx, userID, chatID, count, notificationsEnabled)
			}
			return

		}
		h.clearPendingInput(chatID)
		// Если пришло не число — просим повторить и сохраняем цепочку reply
		replyMsg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите число:")
		replyMsg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "Введите число",
			Selective:             true,
		}
		// reply к исходному сообщению запроса (мы сохранили его id в pendingInput)

		msg, err := h.bot.Send(replyMsg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

		h.sendCancelButton(chatID, input.inputType, msg.MessageID)
		return
	}

	// Если ожидания ввода нет — обычная обработка кнопок/команд
	notificationsEnabled, err := h.service.GetNotificationsStatus(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения статуса уведомлений: %v", err)
		notificationsEnabled = true
	}

	// Сброс дневной нормы и maxReps
	if text == "/reset_norm" {
		if err := h.service.ResetMaxReps(ctx, userID); err != nil {
			log.Printf("Ошибка сброса max_reps: %v", err)
			h.bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка при сбросе. Попробуйте позже."))
			return
		}
		msg := tgbotapi.NewMessage(chatID, "✅ Дневная норма сброшена до значения по умолчанию (40)")
		msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
		h.bot.Send(msg)
		return
	}

	if text == "/debug_cache" {
		if !h.adminIDs[userID] {
			h.bot.Send(tgbotapi.NewMessage(chatID, "⛔ У тебя нет прав для этой команды"))
			return
		}

		userCount := h.service.DebugCache().Size()
		dump := h.service.DebugCache().Dump()
		debugMassage := fmt.Sprintf("Общее число пользователей: %d \n%s", userCount, dump)
		msg := tgbotapi.NewMessage(chatID, debugMassage)
		h.bot.Send(msg)
		return
	}

	switch text {
	case "/start":
		h.handleStart(ctx, chatID, userID, username, notificationsEnabled)
	case "➕ Добавить отжимания":
		h.requestPushupCount(chatID, perDayLimit)
	case "🎯 Определить норму":
		h.requestMaxReps(chatID)
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
		// Если это неизвестная команда (начинается с '/')
		if strings.HasPrefix(text, "/") {
			msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте меню.")
			msg.ReplyMarkup = ui.MainKeyboard(notificationsEnabled)
			h.bot.Send(msg)
		}
		// Обычный текст — игнорируем
	}
}

func (h *BotHandler) handleAddPushups(ctx context.Context, userID int64, username string, chatID int64, count int, notEnable bool) {
	if count <= 0 {
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число:")
		msg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "Введите число",
			Selective:             true,
		}
		sentMsg, err := h.bot.Send(msg)

		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			return
		}

		h.sendCancelButton(chatID, perDayLimit, sentMsg.MessageID)
		return
	}

	if count > oneTimeEntryLimit {
        msg := tgbotapi.NewMessage(chatID, "❌ Превышен лимит разового ввода (1000 отжиманий)")
		msg.ReplyMarkup = ui.MainKeyboard(notEnable)
        h.bot.Send(msg)
        return
    }

	result, err := h.service.AddPushups(ctx, userID, username, count)
	if err != nil {
		log.Printf("Ошибка при добавлении отжиманий: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже или введите /start.")
		h.bot.Send(msg)
		return
	}

	log.Printf("Username%s UserID %d added %d pushups", username, userID, count)

	response := fmt.Sprintf("✅Добавлено: %d отжиманий!\n📈Твой прогресс: %d/%d\n", count, result.TotalToday, result.DailyNorm)

	// Проверка выполнения нормы
	hasCompleted, firstCompleter := h.service.CheckNormCompletion(result.DailyNorm)
	
	if result.TotalToday >= result.DailyNorm {
		response += "\n🎯 Ты выполнил дневную норму!\n"
	} else {
		if !hasCompleted {
			response += "\n❌ Никто еще не выполнил норму сегодня.\nМожет, ты будешь первым? 💪\n"
		} else {
			response += fmt.Sprintf("\n🎯 %s уже выполнил норму!\nА ты не отставай, присоединяйся! 🚀\n", firstCompleter)
		}
	}

	msg := tgbotapi.NewMessage(chatID, response)
	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
	h.bot.Send(msg)
}

func (h *BotHandler) handleSetMaxReps(ctx context.Context, userID int64, username string, chatID int64, count int, notEnable bool) {
	if count <= 0 {
		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число:")
		msg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "Введите число",
			Selective:             true,
		}
		sentMsg, err := h.bot.Send(msg)

		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			return
		}

		h.sendCancelButton(chatID, inputTypeMaxReps, sentMsg.MessageID)
		return
	}

	if count > maxRepsLimit {
        msg := tgbotapi.NewMessage(chatID, "❌ Превышен лимит для одного подхода (500 отжиманий)")
		msg.ReplyMarkup = ui.MainKeyboard(notEnable)
        h.bot.Send(msg)
        return
    }

	err := h.service.SetMaxReps(ctx, userID, username, count) 
	if err != nil {
		log.Printf("Ошибка при при записи max_reps: %v", err)
	}

	delyNorm := service.CalculateDailyNorm(count)

	err = h.service.SetDailyNorm(ctx, userID, delyNorm)
	if err != nil {
		log.Printf("Ошибка при определении нормы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже или введите /start.")
		h.bot.Send(msg)
		return
	}

	response := fmt.Sprintf("🔔Твоя дневная норма установлена: %d\n", delyNorm)

	log.Printf("Username%s UserID %d set %d dely_norm", username, userID, delyNorm)

	msg := tgbotapi.NewMessage(chatID, response)
	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
	h.bot.Send(msg)
}

func (h *BotHandler) requestPushupCount(chatID int64, inputType inputType) {

	// Отправляем ForceReply
	msg := tgbotapi.NewMessage(chatID, "Введите количество отжиманий:")
	msg.ReplyMarkup = tgbotapi.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "Введите число",
		Selective:             true,
	}
	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}

	h.sendCancelButton(chatID, inputType, sentMsg.MessageID)
}

func (h *BotHandler) requestMaxReps(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Введите максимальное количество отжиманий за один подход:")
	msg.ReplyMarkup = tgbotapi.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "Введите число",
		Selective:             true,
	}
	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}

	h.sendCancelButton(chatID, inputTypeMaxReps, sentMsg.MessageID)
}

func (h *BotHandler) requestCustomNorm(chatID int64) {
	// Сначала отправляем сообщение
	msg := tgbotapi.NewMessage(chatID, "Введите дневную норму отжиманий:")

	msg.ReplyMarkup = tgbotapi.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: "Введите число",
		Selective:             true,
	}
	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
		return
	}

	h.sendCancelButton(chatID, inputTypeCustomNorm, sentMsg.MessageID)
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

	daylyStatText := fmt.Sprintf("📊Сегодня ты отжался %d/%d %s.\n%s\n", total, dailyNorm, formatTimesWord(total), generateProgressBar(total, dailyNorm, 10))

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

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty) // или  ░ ▒ ▓ █ 🪫 🔋
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

func (h *BotHandler) handleStart(ctx context.Context, chatID int64, userID int64, username string, notEnable bool) {

	err := h.service.EnsureUser(ctx, userID, username)
	if err != nil {
		log.Printf("Ошибка при попытке создать или обновить пользователя: %v", err)
		return
	}

	maxReps, err := h.service.GetUserMaxReps(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения данных: %v", err)
		return
	}

	if maxReps == 0 {
		msg := tgbotapi.NewMessage(chatID, "Необходимо определить твою дневную норму!")
		h.bot.Send(msg)
		h.requestMaxReps(chatID)
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

// Добавим новую функцию для обработки установки дневной нормы
func (h *BotHandler) handleSetCustomNorm(ctx context.Context, userID int64, chatID int64, dailyNorm int, notEnable bool) {
	if dailyNorm <= 0 {

		msg := tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число:")
		msg.ReplyMarkup = tgbotapi.ForceReply{
			ForceReply:            true,
			InputFieldPlaceholder: "Введите число",
			Selective:             true,
		}
		sentMsg, err := h.bot.Send(msg)

		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
			return
		}

		h.sendCancelButton(chatID, inputTypeCustomNorm, sentMsg.MessageID)
		return
	}

	 if dailyNorm > castomDailyNormLimit {
        msg := tgbotapi.NewMessage(chatID, "❌ Максимальная дневная норма - 500 отжиманий")
		msg.ReplyMarkup = ui.MainKeyboard(notEnable)
        h.bot.Send(msg)
        return
    }

	err := h.service.SetDailyNorm(ctx, userID, dailyNorm)
	if err != nil {
		log.Printf("Ошибка при установке дневной нормы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже или введите /start.")

		h.bot.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Дневная норма установлена: %d", dailyNorm))
	msg.ReplyMarkup = ui.MainKeyboard(notEnable)
	h.bot.Send(msg)
}

// sendCancelButton показывает inline-кнопку "Отменить" и сохраняет её ID в pendingInput.
// Перед отправкой новой кнопки удаляет старую (если она была).
func (h *BotHandler) sendCancelButton(chatID int64, inputType inputType, replyMsgID int) {
	// 1) Если уже есть pendingInput — удаляем старое сообщение с кнопкой (чтобы не копилось)
	if old, ok := h.getPendingInput(chatID); ok {
		if old.cancelMsgID != 0 {
			delOld := tgbotapi.NewDeleteMessage(chatID, old.cancelMsgID)
			if _, err := h.bot.Send(delOld); err != nil {
				// Логируем, но продолжаем — возможно сообщение уже удалено/истекло
				log.Printf("sendCancelButton: не удалось удалить старую кнопку отмены (chat=%d, msg=%d): %v", chatID, old.cancelMsgID, err)
			}
		}
	}

	// 2) Отправляем новое сообщение с inline-кнопкой "Отменить"
	cancelMsg := tgbotapi.NewMessage(chatID, "Если передумал — нажми Отменить:")
	cancelMsg.ReplyMarkup = ui.CancelInlineKeyboard()
	sentCancelMsg, err := h.bot.Send(cancelMsg)
	if err != nil {
		log.Printf("sendCancelButton: ошибка отправки кнопки отмены: %v", err)
		return
	}

	// 3) Сохраняем новое состояние (перезаписываем pendingInput для этого чата)
	h.pendingInputs.Store(chatID, pendingInput{
		inputType:   inputType,
		messageID:   replyMsgID,
		cancelMsgID: sentCancelMsg.MessageID,
	})
}

func (h *BotHandler) getPendingInput(chatID int64) (pendingInput, bool) {
	value, ok := h.pendingInputs.Load(chatID)
	if !ok {
		return pendingInput{}, false
	}
	return value.(pendingInput), true
}

// clearPendingInput удаляет pendingInput и сообщение с кнопкой отмены (если есть)
func (h *BotHandler) clearPendingInput(chatID int64) {
	if input, ok := h.getPendingInput(chatID); ok {
		if input.cancelMsgID != 0 {
			del := tgbotapi.NewDeleteMessage(chatID, input.cancelMsgID)
			if _, err := h.bot.Send(del); err != nil {
				log.Printf("clearPendingInput: не удалось удалить сообщение с кнопкой отмены (chat=%d, msg=%d): %v", chatID, input.cancelMsgID, err)
			}
		}
	}
	h.pendingInputs.Delete(chatID)
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
