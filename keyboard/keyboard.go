package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainKeyboard - основная клавиатура с двумя кнопками
func MainKeyboard(notificationsEnabled bool) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить отжимания"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🛠️ Настройки"),
		),
	)
}

// SettingsKeyboard - клавиатура с дополнительными функциями
func SettingsKeyboard(notificationsEnabled bool) tgbotapi.ReplyKeyboardMarkup {
	var notificationButton tgbotapi.KeyboardButton
	if notificationsEnabled {
		notificationButton = tgbotapi.NewKeyboardButton("🔕 Отключить напоминания")
	} else {
		notificationButton = tgbotapi.NewKeyboardButton("🔔 Включить напоминания")
	}
	
	return tgbotapi.NewReplyKeyboard(
		// Первый ряд - основные настройки
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📝 Установить норму"),
			tgbotapi.NewKeyboardButton("🎯 Определить норму"),
		),
		// Второй ряд - дополнительные функции
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔄 Сброс"),
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
		// Третий ряд - управление уведомлениями
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
			notificationButton,
		),
	)
}

// // SimpleKeyboard - упрощенная клавиатура (по желанию)
// func SimpleKeyboard() tgbotapi.ReplyKeyboardMarkup {
// 	return tgbotapi.NewReplyKeyboard(
// 		tgbotapi.NewKeyboardButtonRow(
// 			tgbotapi.NewKeyboardButton("Добавить отжимания"),
// 			tgbotapi.NewKeyboardButton("🛠️ Настройки"),
// 		),
// 	)
// }

func StartKeyboard() tgbotapi.ReplyKeyboardMarkup{
return tgbotapi.NewReplyKeyboard(
		// Первый ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/start"),
		),
	)
}