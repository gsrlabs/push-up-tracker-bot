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
	oneTimeEntryLimit    = 1000
	maxRepsLimit         = 500
	castomDailyNormLimit = 500
)

type pendingInput struct {
	inputType   inputType
	messageID   int
	cancelMsgID int // ID сообщения с кнопкой отмены
}

type BotHandler struct {
	bot           *tgbotapi.BotAPI
	service       service.PushupService
	pendingInputs sync.Map
	adminIDs      map[int64]bool
}

func NewBotHandler(bot *tgbotapi.BotAPI, service service.PushupService) *BotHandler {
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

			msg := tgbotapi.NewMessage(chatID, "Ввод отменен")
			msg.ReplyMarkup = ui.MainKeyboard()
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

			switch input.inputType {
			case perDayLimit:
				h.handleAddPushups(ctx, userID, username, chatID, count)
			case inputTypeMaxReps:
				h.handleSetMaxReps(ctx, userID, username, chatID, count)
			case inputTypeCustomNorm:
				h.handleSetCustomNorm(ctx, userID, chatID, count)
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

	// Сброс дневной нормы и maxReps
	if text == "/reset_norm" {
		if err := h.service.ResetDailyNorm(ctx, userID); err != nil {
			log.Printf("Ошибка сброса daily_norm: %v", err)
			h.bot.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка при сбросе. Попробуйте позже."))
			return
		}
		msg := tgbotapi.NewMessage(chatID, "✅ Дневная норма сброшена до значения по умолчанию (40)")
		msg.ReplyMarkup = ui.MainKeyboard()
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
		h.handleStart(ctx, chatID, userID, username)
	case "➕ Добавить отжимания":
		h.requestPushupCount(chatID, perDayLimit)
	case "🎯 Тест максимальных отжиманий":
		h.requestMaxReps(chatID)
	case "📝 Установить норму":
		h.requestCustomNorm(chatID)
	case "📊 Статистика":
		h.handleTodayStat(ctx, userID, chatID)
		h.handleTotalStat(ctx, userID, chatID)
		h.handleTodayLeaderboard(ctx, chatID)
	case "⚙️ Дополнительно":
		msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
		msg.ReplyMarkup = ui.SettingsKeyboard()
		h.bot.Send(msg)
	case "/info", "/help":
		h.handleInfo(chatID)
	case "📈 Мой прогресс":
		h.handleProgressHistory(ctx, userID, chatID)
	case "⬅️ Назад":
		msg := tgbotapi.NewMessage(chatID, "Главное меню:")
		msg.ReplyMarkup = ui.MainKeyboard()
		h.bot.Send(msg)
	default:
		// Если это неизвестная команда (начинается с '/')
		if strings.HasPrefix(text, "/") {
			msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте меню.")
			msg.ReplyMarkup = ui.MainKeyboard()
			h.bot.Send(msg)
		}
		// Обычный текст — игнорируем
	}
}

func (h *BotHandler) handleAddPushups(ctx context.Context, userID int64, username string, chatID int64, count int) {
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
		msg.ReplyMarkup = ui.MainKeyboard()
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
	hasCompleted, firstCompleter := h.service.CheckNormCompletion(ctx)

	if result.TotalToday >= result.DailyNorm {
		response += "\n🎯 Ты выполнил дневную норму!\n"
		if err := h.service.SetDateCompletionOfDailyNorm(ctx, userID); err != nil {
			log.Printf("Ошибка обновления даты выполнения нормы: %v", err)
		}
	} else {
		if !hasCompleted {
			response += "\n❌ Никто еще не выполнил норму сегодня.\nМожет, ты будешь первым? 💪\n"
		} else {
			response += fmt.Sprintf("\n🎯 %s уже выполнил норму!\nА ты не отставай, присоединяйся! 🚀\n", firstCompleter)
		}
	}

	msg := tgbotapi.NewMessage(chatID, response)
	//msg.ParseMode = "Markdown"
	msg.ReplyMarkup = ui.MainKeyboard()
	h.bot.Send(msg)
}

func (h *BotHandler) handleSetMaxReps(ctx context.Context, userID int64, username string, chatID int64, count int) {
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
		msg.ReplyMarkup = ui.MainKeyboard()
		h.bot.Send(msg)
		return
	}

	// Сохраняем текущее максимальное количество отжиманий
	err := h.service.SetMaxReps(ctx, userID, username, count)
	if err != nil {
		log.Printf("Ошибка при записи max_reps: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		msg.ReplyMarkup = ui.MainKeyboard()
		//msg.ParseMode = "Markdown"
		h.bot.Send(msg)
		return
	}

	// Рассчитываем и устанавливаем дневную норму
	dailyNorm := service.CalculateDailyNorm(count)
	err = h.service.SetDailyNorm(ctx, userID, dailyNorm)
	if err != nil {
		log.Printf("Ошибка при определении нормы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		msg.ReplyMarkup = ui.MainKeyboard()
		h.bot.Send(msg)
		return
	}

	// Получаем историю максимальных отжиманий
	history, err := h.service.GetMaxRepsHistory(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения истории max_reps: %v", err)
		// Продолжаем выполнение, даже если история недоступна
	}

	record, err := h.service.GetMaxRepsRecord(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения рекорда max_reps: %v", err)
		// Продолжаем выполнение, даже если история недоступна
	}

	// Формируем ответ с историей
	response := fmt.Sprintf("✅ Твой результат: %d отжиманий за подход!\n\n", count)
	response += fmt.Sprintf("🔔 Дневная норма установлена: %d\n\n", dailyNorm)
	response += fmt.Sprintf("🎖️ Твой текущий ранг: %s!\n\n", service.GetUserRank(count))

	repsToNextRank := service.GetRepsToNextRank(count)
	if repsToNextRank > 0 {
		response += fmt.Sprintf("🎯 До следующего ранга тебе осталось: +%d\n\n", repsToNextRank)
	}

	if record.MaxReps != 0 {
		response += fmt.Sprintf("💪 Твой рекорд: %s → %d отжиманий!\n\n",
			record.Date.Format("02.01.2006"),
			record.MaxReps)
	}

	if len(history) >= 2 {
		response += "📝 Твой предыдущий результат:\n"

		item := history[1]
		response += fmt.Sprintf("• %s → %d\n",
			item.Date.Format("02.01.2006"),
			item.MaxReps)

		// Анализ прогресса
		if len(history) > 1 {
			latest := history[0].MaxReps
			previous := history[1].MaxReps
			if latest > previous {
				progress := latest - previous
				response += fmt.Sprintf("\n🎉 Прогресс: +%d отжиманий! 💪", progress)
			} else if latest == previous {
				response += "\n📊 Стабильный результат! 🎯"
			}
		}
	} else {
		response += "\n🎯 Это твой первый рекорд! Начнем отслеживать прогресс!"
	}

	log.Printf("Username %s UserID %d set max_reps: %d, daily_norm: %d", username, userID, count, dailyNorm)

	msg := tgbotapi.NewMessage(chatID, response)
	msg.ReplyMarkup = ui.MainKeyboard()
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

func (h *BotHandler) handleTodayStat(ctx context.Context, userID int64, chatID int64) {
	total, err := h.service.GetTodayStat(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	if total == 0 {
		return
	}

	dailyNorm, err := h.service.GetDailyNorm(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении дневной нормы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	daylyStatText := fmt.Sprintf("📊Сегодня ты отжался %s \nТвоя дневная норма: %d \n%s\n", service.FormatTimesWord(total), dailyNorm, service.GenerateProgressBar(total, dailyNorm, 10))

	msg := tgbotapi.NewMessage(chatID, daylyStatText)
	msg.ReplyMarkup = ui.MainKeyboard()
	h.bot.Send(msg)
}

func (h *BotHandler) handleTotalStat(ctx context.Context, userID int64, chatID int64) {
	total, err := h.service.GetTotalStat(ctx, userID)
	if err != nil {
		log.Printf("Ошибка при получении статистики: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		h.bot.Send(msg)
		return
	}

	var statText string
	var FirstWorkoutDateText string

	firstWorkoutDate, err := h.service.GetFirstWorkoutDate(ctx, userID)
	if err != nil || firstWorkoutDate == "01.01.0001" {
		FirstWorkoutDateText = "Ты ещё не начинал тренироваться"
	} else {
		statText = fmt.Sprintf("💪За все время ты отжался: %s\n", service.FormatTimesWord(total))
		FirstWorkoutDateText = fmt.Sprintf("Первая тренировка: %s", firstWorkoutDate)
	}

	msg := tgbotapi.NewMessage(chatID, statText+FirstWorkoutDateText)
	msg.ReplyMarkup = ui.MainKeyboard()
	h.bot.Send(msg)
}

func (h *BotHandler) handleStart(ctx context.Context, chatID int64, userID int64, username string) {
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

	welcomeMsg := `*Добро пожаловать в бот для учёта отжиманий!*

Я помогу вам:
• 📊 Следить за ежедневным прогрессом
• 🎯 Определить вашу персональную норму
• 📈 Отслеживать рост силы over time

🚀 *Чтобы начать:*
1. Сделайте *«🎯 Тест максимальных отжиманий»* для установки нормы
2. Ежедневно добавляйте отжимания кнопкой *«➕ Добавить отжимания»*
3. Следите за прогрессом в *«📈 Мой прогресс»*

📖 Подробная инструкция — /info`

	if maxReps == 0 {
		msg := tgbotapi.NewMessage(chatID, welcomeMsg)
		msg.ParseMode = "Markdown"
		h.bot.Send(msg)
		h.requestMaxReps(chatID)
		return
	}

	msg := tgbotapi.NewMessage(chatID, welcomeMsg+"\n\nВыберите действие:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = ui.MainKeyboard()
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
func (h *BotHandler) handleSetCustomNorm(ctx context.Context, userID int64, chatID int64, dailyNorm int) {
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
		msg.ReplyMarkup = ui.MainKeyboard()
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
	msg.ReplyMarkup = ui.MainKeyboard()
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

// handleProgressHistory метод для обработки истории прогресса
func (h *BotHandler) handleProgressHistory(ctx context.Context, userID int64, chatID int64) {
	history, err := h.service.GetMaxRepsHistory(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения истории прогресса: %v", err)
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка загрузки истории прогресса")
		//msg.ReplyMarkup = ui.MainKeyboard(notEnable)
		h.bot.Send(msg)
		return
	}

	if len(history) == 0 {
		msg := tgbotapi.NewMessage(chatID, "📊 История прогресса пуста.\nИспользуй \"🎯 Обновить прогресс\" что бы начать историю прогресса!")
		msg.ReplyMarkup = ui.MainKeyboard()
		h.bot.Send(msg)
		return
	}

	var response strings.Builder
	response.WriteString("📈 Твоя история прогресса максимальных отжиманий:\n\n")

	for i := 0; i < len(history); i++ {
		response.WriteString(fmt.Sprintf("%d. %s → %d отжиманий\n",
			i+1,
			history[len(history)-1-i].Date.Format("02.01.2006"),
			history[len(history)-1-i].MaxReps))
	}

	// Анализ общего прогресса
	if len(history) > 1 {
		first := history[len(history)-1].MaxReps
		last := history[0].MaxReps
		progress := last - first

		response.WriteString("\n📊 Общий прогресс: ")
		if progress > 0 {
			response.WriteString(fmt.Sprintf("+%d отжиманий! 🚀", progress))
		} else if progress < 0 {
			response.WriteString(fmt.Sprintf("%d отжиманий 📉", progress))
		} else {
			response.WriteString("стабильно! 🎯")
		}
	}

	msg := tgbotapi.NewMessage(chatID, response.String())
	msg.ReplyMarkup = ui.MainKeyboard()
	h.bot.Send(msg)

	err = service.SendSchedule(h.bot, chatID, history)
	if err != nil {
		log.Println(err)
	}
}

// handleInfo отправляет инструкцию по использованию бота
func (h *BotHandler) handleInfo(chatID int64) {
	instruction := `🤖 *Инструкция по использованию PushUpper*

🎯 *Основные функции*

*➕ Добавить отжимания*
Записывайте ежедневные отжимания в общую статистику
Показывает текущий прогресс выполнения дневной нормы
Участвуйте в соревновании - кто первый выполнит норму сегодня!

*⚙️ Дополнительное меню* -> настройки, статистика и прогресс

*🎯 Тест максимальных отжиманий*
Определите ваш рекорд в одном подходе
На основе результата устанавливается персональная дневная норма
Получите свой ранг силы и увидите прогресс до следующего уровня
_Рекомендуется обновлять каждые 1-2 недели_
(Не влияет на выполнение дневной нормы и статистику)

*📊 Статистика* 
└─ *Сегодня*: ваш прогресс и процент выполнения нормы
└─ *Общая*: сумма всех отжиманий за всё время
└─ *Рейтинг*: таблица лидеров среди всех пользователей

*📈 Мой прогресс*
График и список всех ваших рекордов за подход
Отслеживайте динамику роста силы

*📝 Установить норму*
Ручная установка индивидуальной дневной нормы
Полезно если хотите тренироваться по собственному плану

💡 *Советы по использованию*

1. *Начните с теста* - определите свой текущий уровень
2. *Регулярно добавляйте отжимания* - даже небольшие подходы
3. *Обновляйте рекорд* раз в неделю
4. *Следите за прогрессом* через историю и графики

🚀 *Начните сейчас с кнопки «🎯 Тест максимальных отжиманий»!*`

	msg := tgbotapi.NewMessage(chatID, instruction)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = ui.MainKeyboard()
	h.bot.Send(msg)
}
