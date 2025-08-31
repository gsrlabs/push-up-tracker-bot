package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainKeyboard принимает статус уведомлений и возвращает клавиатуру в старом формате
func MainKeyboard(notificationsEnabled bool) tgbotapi.ReplyKeyboardMarkup {
	var notificationButton tgbotapi.KeyboardButton
	if notificationsEnabled {
		notificationButton = tgbotapi.NewKeyboardButton("🔕 Отключить напоминания")
	} else {
		notificationButton = tgbotapi.NewKeyboardButton("🔔 Включить напоминания")
	}
	
	return tgbotapi.NewReplyKeyboard(
		// Первый ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить отжимания"),
		),
		// Второй ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Определить норму"),
		),
		// Третий ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Установить норму"),
			tgbotapi.NewKeyboardButton("Сброс"),
			tgbotapi.NewKeyboardButton("Статистика"),
		),
		tgbotapi.NewKeyboardButtonRow(
			notificationButton,
		),
	)
}

func StartKeyboard() tgbotapi.ReplyKeyboardMarkup{
return tgbotapi.NewReplyKeyboard(
		// Первый ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Start"),
		),
	)
}