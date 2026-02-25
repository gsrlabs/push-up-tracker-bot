package hendler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"time"
	ui "trackerbot/keyboard"
	"trackerbot/presenter"
	"trackerbot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBot interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

type numericConfig struct {
	prompt      string
	placeholder string
	min         int
	max         int
	handler     func(ctx context.Context, userID int64, username string, chatID int64, value int)
}

type inputType int

const (
	inputDayLimit inputType = iota
	inputTypeMaxReps
	inputTypeCustomNorm
)
const (
	oneTimeEntryLimit    = 1000
	maxRepsLimit         = 500
	castomDailyNormLimit = 500
)

type BotHandler struct {
	bot          TelegramBot
	service      service.PushupService
	inputManager *InputManager

	adminIDs       map[int64]bool
	numericConfigs map[inputType]numericConfig
}

func NewBotHandler(bot TelegramBot, service service.PushupService) *BotHandler {
	h := &BotHandler{
		bot:          bot,
		service:      service,
		inputManager: NewInputManager(),
		adminIDs: map[int64]bool{
			1036193976: true,
		},
	}

	h.numericConfigs = map[inputType]numericConfig{
		inputDayLimit: {
			prompt:      "Введите количество отжиманий:",
			placeholder: "Введите число",
			min:         1,
			max:         oneTimeEntryLimit,
			handler:     h.handleAddPushups,
		},
		inputTypeMaxReps: {
			prompt:      "Введите максимальное количество отжиманий за один подход:",
			placeholder: "Введите число",
			min:         1,
			max:         maxRepsLimit,
			handler:     h.handleSetMaxReps,
		},
		inputTypeCustomNorm: {
			prompt:      "Введите дневную норму отжиманий:",
			placeholder: "Введите число",
			min:         1,
			max:         castomDailyNormLimit,
			handler: func(ctx context.Context, userID int64, username string, chatID int64, value int) {
				h.handleSetCustomNorm(ctx, userID, chatID, value)
			},
		},
	}

	return h
}

func (h *BotHandler) HandleUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(update)
		return
	}

	if update.Message != nil {
		h.handleMessage(update)
	}

}

func (h *BotHandler) handleMessage(update tgbotapi.Update) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	username := update.Message.From.UserName
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Проверяем, есть ли ожидаемый ввод
	if input, ok := h.getPendingInput(chatID); ok {
		h.handlePendingInput(ctx, input, userID, username, chatID, text)
		return
	}

	// Команды
	switch text {

	case "/start":
		h.handleStart(ctx, chatID, userID, username, inputTypeMaxReps)

	case "➕ Добавить отжимания":
		h.requestNumber(chatID, inputDayLimit)

	case "🎯 Тест максимальных отжиманий":
		h.requestNumber(chatID, inputTypeMaxReps)

	case "📝 Установить норму":
		h.requestNumber(chatID, inputTypeCustomNorm)

	case "📊 Статистика":
		h.handleFullStat(ctx, userID, chatID)
		return
	case "⚙️ Дополнительно":
		msg := tgbotapi.NewMessage(chatID, "Выберите действие:")
		msg.ReplyMarkup = ui.SettingsKeyboard()
		_, err := h.bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		}

	case "/info", "/help", "📖 Инфо":
		h.handleInfo(chatID)

	case "📈 Мой прогресс":
		h.handleProgressHistory(ctx, userID, chatID)

	case "⬅️ Назад":
		msg := tgbotapi.NewMessage(chatID, "Главное меню:")
		msg.ReplyMarkup = ui.MainKeyboard()

		_, err := h.bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения ⬅️ Назад: %v", err)
		}

	default:
		if strings.HasPrefix(text, "/") {
			msg := tgbotapi.NewMessage(chatID, "Неизвестная команда. Используйте меню.")
			msg.ReplyMarkup = ui.MainKeyboard()
			_, err := h.bot.Send(msg)

			if err != nil {
				log.Printf("Ошибка отправки сообщения default: %v", err)
			}
		}
	}
}

func (h *BotHandler) handlePendingInput(
	ctx context.Context,
	input PendingInput,
	userID int64,
	username string,
	chatID int64,
	text string,
) {
	cfg := h.numericConfigs[input.InputType]

	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		// ❌ Некорректный ввод (буквы, символы и т.д.)
		h.sendValidationError(chatID, input.InputType, "Пожалуйста, введите число")
		return
	}

	// ❌ Меньше минимума (0 или отрицательное)
	if value < cfg.min {
		h.sendValidationError(chatID, input.InputType, "Пожалуйста, введите положительное число")
		return
	}

	// ❌ Больше лимита
	if value > cfg.max {
		h.sendValidationError(chatID, input.InputType,
			fmt.Sprintf("❌ Превышен лимит (%d)", cfg.max))
		return
	}

	// ✅ УСПЕХ — теперь очищаем
	h.clearPendingInput(chatID)

	cfg.handler(ctx, userID, username, chatID, value)
}

func (h *BotHandler) getPendingInput(chatID int64) (PendingInput, bool) {
	value, ok := h.inputManager.Get(chatID)
	if !ok {
		return PendingInput{}, false
	}
	return value, true
}

// clearPendingInput удаляет pendingInput и сообщение с кнопкой отмены (если есть)
func (h *BotHandler) clearPendingInput(chatID int64) {
	if input, ok := h.inputManager.Get(chatID); ok {
		if input.CancelMsgID != 0 {
			del := tgbotapi.NewDeleteMessage(chatID, input.CancelMsgID)
			_, _ = h.bot.Send(del)
		}
	}

	h.inputManager.Delete(chatID)
}

func (h *BotHandler) requestNumber(chatID int64, t inputType) {
	cfg := h.numericConfigs[t]

	msg := tgbotapi.NewMessage(chatID, cfg.prompt)
	msg.ReplyMarkup = tgbotapi.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: cfg.placeholder,
		Selective:             true,
	}

	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения requestNumber: %v", err)
		return
	}

	h.sendCancelButton(chatID, t, sentMsg.MessageID)
}

// sendCancelButton показывает inline-кнопку "Отменить" и сохраняет её ID в pendingInput.
// Перед отправкой новой кнопки удаляет старую (если она была).
func (h *BotHandler) sendCancelButton(chatID int64, inputType inputType, replyMsgID int) {
	// 1) Если уже есть pendingInput — удаляем старое сообщение с кнопкой (чтобы не копилось)
	if old, ok := h.inputManager.Get(chatID); ok {
		if old.CancelMsgID != 0 {
			delOld := tgbotapi.NewDeleteMessage(chatID, old.CancelMsgID)
			_, _ = h.bot.Send(delOld)

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
	h.inputManager.Set(chatID, PendingInput{
		InputType:   inputType,
		MessageID:   replyMsgID,
		CancelMsgID: sentCancelMsg.MessageID,
	})
}

func (h *BotHandler) handleAddPushups(
	ctx context.Context,
	userID int64,
	username string,
	chatID int64,
	count int,
) {

	vm, err := h.service.AddPushups(ctx, userID, count)
	if err != nil {
		h.sendError(chatID)
		return
	}

	response := presenter.FormatAddPushups(vm)

	h.sendMessage(chatID, response, ui.MainKeyboard())
}

func (h *BotHandler) handleSetMaxReps(
	ctx context.Context,
	userID int64,
	username string,
	chatID int64,
	count int,
) {
	vm, err := h.service.UpdateMaxReps(ctx, userID, count)
	if err != nil {
		h.sendError(chatID)
		return
	}

	response := presenter.FormatMaxReps(vm)

	h.sendMessage(chatID, response, ui.MainKeyboard())
}

func (h *BotHandler) handleStart(ctx context.Context, chatID int64, userID int64, username string, perDayLimit inputType) {
	// Проверяем или создаем пользователя
	if err := h.service.EnsureUser(ctx, userID, username); err != nil {
		log.Printf("Ошибка при создании или обновлении пользователя: %v", err)
		return
	}

	maxReps, err := h.service.GetUserMaxReps(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения данных: %v", err)
		return
	}

	// Формируем текст через presenter
	welcomeMsg := presenter.FormatWelcomeMessage(maxReps)

	msg := tgbotapi.NewMessage(chatID, welcomeMsg)
	msg.ParseMode = tgbotapi.ModeHTML

	if maxReps == 0 {
		_, err := h.bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения handleStart: %v", err)
			return
		}
		h.requestNumber(chatID, perDayLimit)
		return
	}

	h.sendMarkdownMessage(chatID, welcomeMsg, ui.MainKeyboard())
}

// Добавим новую функцию для обработки установки дневной нормы
func (h *BotHandler) handleSetCustomNorm(ctx context.Context, userID int64, chatID int64, dailyNorm int) {

	err := h.service.SetDailyNorm(ctx, userID, dailyNorm)
	if err != nil {
		log.Printf("Ошибка при установке дневной нормы: %v", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте позже или введите /start.")

		_, err := h.bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения handleSetCustomNorm(SetDailyNorm): %v", err)
			return
		}
		return
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Дневная норма установлена: %d", dailyNorm))
	msg.ReplyMarkup = ui.MainKeyboard()
	_, err = h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения handleSetCustomNorm: %v", err)
		return
	}

}

// handleProgressHistory метод для обработки истории прогресса
func (h *BotHandler) handleProgressHistory(ctx context.Context, userID int64, chatID int64) {
	history, err := h.service.GetMaxRepsHistory(ctx, userID)
	if err != nil {
		log.Printf("Ошибка получения истории прогресса: %v", err)
		h.sendError(chatID)
		return
	}

	response := presenter.FormatProgressHistory(history)

	h.sendMessage(chatID, response, ui.MainKeyboard())

	if len(history) > 0 {

		image, err := h.service.BuildSchedule(ctx, userID, history)
		if err != nil {
			log.Println("Ошибка при отправке графика прогресса:", err)
		}

		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
			Name:  "schedule.png",
			Bytes: image.Bytes(),
		})

		_, err = h.bot.Send(photo)
		if err != nil {
			log.Printf("Ошибка отправки сообщения handleProgressHistory: %v", err)
			return
		}

	}
}

// handleInfo отправляет инструкцию по использованию бота
func (h *BotHandler) handleInfo(chatID int64) {
	instruction := presenter.FormatInfoMessage()
	h.sendMarkdownMessage(chatID, instruction, ui.MainKeyboard())
}

func (h *BotHandler) handleCallback(update tgbotapi.Update) {
	callback := update.CallbackQuery

	if callback.Data == "cancel_input" {
		chatID := callback.Message.Chat.ID

		h.clearPendingInput(chatID)

		// Ответ на callback
		cb := tgbotapi.NewCallback(callback.ID, "Ввод отменен")
		_, err := h.bot.Request(cb)
		if err != nil {
			log.Printf("Ошибка отправки сообщения handleInfo(NewCallback): %v", err)
			return
		}

		// Обновляем сообщение с кнопкой
		editMsg := tgbotapi.NewEditMessageText(
			chatID,
			callback.Message.MessageID,
			"Ввод отменен",
		)
		_, err = h.bot.Send(editMsg)
		if err != nil {
			log.Printf("Ввод отменен")
		}

		// Отправляем главное меню
		msg := tgbotapi.NewMessage(chatID, "Ввод отменен")
		msg.ReplyMarkup = ui.MainKeyboard()
		_, err = h.bot.Send(msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения handleCallback(Ввод отменен): %v", err)
		}

	}
}

func (h *BotHandler) handleFullStat(
	ctx context.Context,
	userID int64,
	chatID int64,
) {

	vm, err := h.service.GetFullStat(ctx, userID)
	if err != nil {
		log.Printf("GetFullStat error: %v", err)
		h.sendError(chatID)
		return
	}

	response := presenter.FormatFullStat(vm)

	msg := tgbotapi.NewMessage(chatID, response)
	msg.ReplyMarkup = ui.MainKeyboard()

	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("telegram send error: %v", err)
	}
}

func (h *BotHandler) sendMessage(chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	if markup != nil {
		switch v := markup.(type) {
		case tgbotapi.ReplyKeyboardMarkup:
			msg.ReplyMarkup = v
		case tgbotapi.InlineKeyboardMarkup:
			msg.ReplyMarkup = v
		case tgbotapi.ForceReply:
			msg.ReplyMarkup = v
		}
	}
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения sendMessage: %v", err)
		return
	}
}

func (h *BotHandler) sendMarkdownMessage(chatID int64, text string, markup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	if markup != nil {
		switch v := markup.(type) {
		case tgbotapi.ReplyKeyboardMarkup:
			msg.ReplyMarkup = v
		case tgbotapi.InlineKeyboardMarkup:
			msg.ReplyMarkup = v
		case tgbotapi.ForceReply:
			msg.ReplyMarkup = v
		}
	}
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения sendMarkdownMessage: %v", err)
		return
	}
}

func (h *BotHandler) sendError(chatID int64) {
	h.sendMessage(chatID, "Произошла ошибка. Попробуйте позже или наждмите /start", ui.MainKeyboard())
}

func (h *BotHandler) sendValidationError(chatID int64, t inputType, message string) {
	cfg := h.numericConfigs[t]

	msg := tgbotapi.NewMessage(chatID, message)
	msg.ReplyMarkup = tgbotapi.ForceReply{
		ForceReply:            true,
		InputFieldPlaceholder: cfg.placeholder,
		Selective:             true,
	}

	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения sendError: %v", err)
		return
	}

	h.sendCancelButton(chatID, t, sentMsg.MessageID)
}
