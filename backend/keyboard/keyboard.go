package keyboard

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainKeyboard - основная клавиатура с двумя кнопками
func MainKeyboard() tgbotapi.ReplyKeyboardMarkup {
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
func SettingsKeyboard() tgbotapi.ReplyKeyboardMarkup {

	return tgbotapi.NewReplyKeyboard(
        // Первый ряд - основные настройки
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("🎯 Тест максимальных отжиманий"),
        ),
		 // Второй ряд - основные настройки
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("📝 Установить норму"),
			tgbotapi.NewKeyboardButton("📊 Статистика"),
        ),
        // Третий ряд - дополнительные функции
        tgbotapi.NewKeyboardButtonRow(
            tgbotapi.NewKeyboardButton("📈 Мой прогресс"),
        ),
        // Четвертый ряд - управление уведомлениями
        tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⬅️ Назад"),
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

