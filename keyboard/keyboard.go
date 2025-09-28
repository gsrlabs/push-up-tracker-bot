package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainKeyboard - основная клавиатура с двумя кнопками
func MainKeyboard(notificationsEnabled bool) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить отжимания"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Дополнительно"),
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
            tgbotapi.NewKeyboardButton("🎯 Обновить прогресс"),
        ),
        // Второй ряд - дополнительные функции
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("📈 История прогресса"),
            tgbotapi.NewKeyboardButton("📊 Статистика"),
        ),
        // Третий ряд - управление уведомлениями
        tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
            notificationButton,
        ),
    )
}


func CancelInlineKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel_input"),
		),
	)
}

