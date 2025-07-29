// Пакет bot содержит компоненты для работы с Telegram Bot API
package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// mainKeyboard создает и возвращает кастомную клавиатуру для основного меню бота
// 
// Клавиатура представляет собой Reply-клавиатуру (появляется вместо стандартной клавиатуры)
// и содержит три кнопки, сгруппированные в два ряда:
// 
// Первый ряд:
// - "Добавить отжимания" - для добавления новых записей
// - "Статистика за сегодня" - для просмотра дневной статистики
// 
// Второй ряд:
// - "Статистика за всё время" - для просмотра общей статистики
// 
// Возвращает:
// - tgbotapi.ReplyKeyboardMarkup: настроенную клавиатуру для основного меню
func mainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		// Первый ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("+ за день"),
		),
		// Второй ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("+ за один подход"),
		),
		// Третий ряд кнопок
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сброс"),
			tgbotapi.NewKeyboardButton("Статистика"),
		),
	)
}