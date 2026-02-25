package presenter

import (
	"fmt"
	"strings"
	"time"
	"trackerbot/repository"
)

type LeaderboardItem struct {
	Username string
	Count    int
}

type FullStatViewModel struct {
	TodayTotal       int
	TotalAllTime     int
	DailyNorm        int
	FirstWorkoutDate *time.Time
	Leaderboard      []LeaderboardItem
}

type AddPushupsViewModel struct {
	AddedCount int
	Total      int
	DailyNorm  int
	Completed  bool
	HasLeader  bool
	Leader     string
}

type MaxRepsViewModel struct {
	Count      int
	DailyNorm  int
	Rank       string
	RepsToNext int
	History    []repository.MaxRepsHistoryItem
	Record     *repository.MaxRepsHistoryItem
}

func FormatWelcomeMessage(maxReps int) string {
	baseMsg := `👋 <b>Добро пожаловать в PushUpper!</b>

Это Telegram-бот для удобного учёта ваших отжиманий и прогресса 💪

<b>Я помогу вам:</b>
• 📊 Следить за ежедневным прогрессом  
• 🎯 Рассчитать персональную дневную норму  
• 📈 Отслеживать рост силы со временем  

🚀 <b>Чтобы начать:</b>
1️⃣ Пройдите <b>«🎯 Тест максимальных отжиманий»</b> — бот рассчитает вашу норму  
2️⃣ Добавляйте подходы через <b>«➕ Добавить отжимания»</b>  
3️⃣ Анализируйте результаты в <b>«📈 Мой прогресс»</b>  

📖 Подробная инструкция — в разделе <b>/info</b>`

	if maxReps == 0 {
		return baseMsg
	}

	return baseMsg + `

Выберите действие ниже 👇`
}

func FormatInfoMessage() string {
	return `🤖 <b>Инструкция по использованию PushUpper</b>

🎯 <b>Основные функции</b>

<b>➕ Добавить отжимания</b>
Записывайте ежедневные отжимания в общую статистику
Показывает текущий прогресс выполнения дневной нормы
Участвуйте в соревновании — кто первый выполнит норму сегодня

<b>⚙️ Дополнительное меню</b>
Настройки, статистика и прогресс

<b>🎯 Тест максимальных отжиманий</b>
Определите ваш рекорд в одном подходе
На основе результата устанавливается персональная дневная норма
Получите свой ранг силы и увидите прогресс до следующего уровня
Рекомендуется обновлять каждые 1–2 недели
Не влияет на выполнение дневной нормы и статистику

<b>📊 Статистика</b>
Сегодня — ваш прогресс и процент выполнения нормы
Общая — сумма всех отжиманий за всё время
Рейтинг — таблица лидеров среди всех пользователей

<b>📈 Мой прогресс</b>
График и список всех ваших рекордов за подход
Отслеживайте динамику роста силы

<b>📝 Установить норму</b>
Ручная установка индивидуальной дневной нормы
Полезно если хотите тренироваться по собственному плану

💡 <b>Советы по использованию</b>

1. Начните с теста — определите свой текущий уровень
2. Регулярно добавляйте отжимания — даже небольшие подходы
3. Обновляйте рекорд раз в неделю
4. Следите за прогрессом через историю и графики

🚀 <b>Начните сейчас с кнопки «🎯 Тест максимальных отжиманий»</b>

📅 <b>Рекомендованная частота тренировок по рангу</b>

<i>Формат: количество отжиманий за один подход → ваш ранг → рекомендуемое число тренировок в неделю</i>

<b>1–4</b>  💤 <b>Сонная муха</b>  
<i>1–2 тренировки в неделю</i>

<b>5–9</b>  🌱 <b>Росток силы</b>  
<i>2–3 тренировки в неделю</i>

<b>10–14</b>  🐜 <b>Трудяга</b>  
<i>2–3 тренировки в неделю</i>

<b>15–19</b>  🚀 <b>Стажёр космоса</b>  
<i>3 тренировки в неделю</i>

<b>20–24</b>  🚀 <b>Ракета-носитель</b>  
<i>3–4 тренировки в неделю</i>

<b>25–29</b>  ⚔️ <b>Рыцарь света</b>  
<i>3–4 тренировки в неделю</i>

<b>30–39</b>  🛡️ <b>Непробиваемый</b>  
<i>3–4 тренировки в неделю</i>

<b>40–49</b>  ⚡ <b>Гроза пола</b>  
<i>4–5 тренировок в неделю</i>

<b>50–64</b>  🏹 <b>Адепт упорства</b>  
<i>4–5 тренировок в неделю</i>

<b>65–79</b>  🌌 <b>Победитель гравитации</b>  
<i>4–6 тренировок в неделю</i>

<b>80–99</b>  🏆 <b>Легенда горизонтов</b>  
<i>4–6 тренировок в неделю</i>

<b>100+</b>  🌟 <b>ВЛАСТЕЛИН ОТЖИМАНИЙ</b>  
<i>5–6 тренировок в неделю</i>

⚠️ <i>6–7 раз в неделю допустимо только при хорошем восстановлении и без боли в суставах</i>`
}

func FormatFullStat(vm *FullStatViewModel) string {

	var builder strings.Builder

	// --- Сегодня ---
	if vm.TodayTotal > 0 {

		_, _ = fmt.Fprintf(
			&builder,
			"📊 Сегодня ты отжался %s\n",
			FormatTimesWord(vm.TodayTotal),
		)

		_, _ = fmt.Fprintf(
			&builder,
			"Твоя дневная норма: %s\n%s\n\n",
			FormatTimesWord(vm.DailyNorm),
			GenerateProgressBar(vm.TodayTotal, vm.DailyNorm, 10),
		)
	}

	// --- За всё время ---
	if vm.TotalAllTime > 0 {

		_, _ = fmt.Fprintf(
			&builder, "💪 За всё время ты отжался: %s\n",
			FormatTimesWord(vm.TotalAllTime),
		)

		if vm.FirstWorkoutDate != nil {
			_, _ = fmt.Fprintf(
				&builder, "Первая тренировка: %s\n\n",
				vm.FirstWorkoutDate.Format("02.01.2006"),
			)

		}
	} else {
		_, _ = builder.WriteString("Ты ещё не начинал тренироваться\n\n")
	}

	// --- Лидерборд ---
	if len(vm.Leaderboard) > 0 {
		_, _ = builder.WriteString("🏆 Статистика за сегодня:\n\n")

		for i, item := range vm.Leaderboard {

			_, _ = fmt.Fprintf(
				&builder, "%d. %s: %d\n",
				i+1,
				item.Username,
				item.Count,
			)
		}
	}

	return builder.String()
}

func FormatAddPushups(vm *AddPushupsViewModel) string {

	var builder strings.Builder

	_, _ = fmt.Fprintf(
		&builder, "✅ Добавлено: %d отжиманий!\n📈 Твой прогресс: %d/%d\n",
		vm.AddedCount,
		vm.Total,
		vm.DailyNorm,
	)

	if vm.Completed {
		_, _ = builder.WriteString("\n🎯 Ты выполнил дневную норму!\n")
		return builder.String()
	}

	if !vm.HasLeader {
		builder.WriteString(
			"\n❌ Никто еще не выполнил норму сегодня.\nМожет, ты будешь первым? 💪\n",
		)
		return builder.String()
	}

	_, _ = fmt.Fprintf(
		&builder,
		"\n🎯 %s уже выполнил норму!\nА ты не отставай, присоединяйся! 🚀\n",
		vm.Leader,
	)

	return builder.String()
}

func FormatMaxReps(vm *MaxRepsViewModel) string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(
		&builder,
		"✅ Твой результат: %d отжиманий за подход!\n\n",
		vm.Count,
	)

	_, _ = fmt.Fprintf(
		&builder, "🔔 Дневная норма установлена: %d\n\n",
		vm.DailyNorm,
	)

	_, _ = fmt.Fprintf(
		&builder,
		"🎖️ Твой текущий ранг: %s!\n\n",
		vm.Rank,
	)

	if vm.RepsToNext > 0 {

		_, _ = fmt.Fprintf(
			&builder,
			"🎯 До следующего ранга тебе осталось: +%d\n\n",
			vm.RepsToNext,
		)
	}

	if vm.Record != nil {

		_, _ = fmt.Fprintf(
			&builder,
			"💪 Твой рекорд: %s → %d отжиманий!\n\n",
			vm.Record.Date.Format("02.01.2006"),
			vm.Record.MaxReps,
		)
	}

	if len(vm.History) >= 2 {
		_, _ = builder.WriteString("📝 Твой предыдущий результат:\n")
		prev := vm.History[1]

		_, _ = fmt.Fprintf(
			&builder,
			"• %s → %d\n",
			prev.Date.Format("02.01.2006"),
			prev.MaxReps,
		)

		latest := vm.History[0].MaxReps
		previous := vm.History[1].MaxReps
		switch {
		case latest > previous:
			_, _ = fmt.Fprintf(&builder, "\n🎉 Прогресс: +%d отжиманий!", latest-previous)
		case latest == previous:
			_, _ = builder.WriteString("\n📊 Стабильный результат!")
		}
	} else {
		_, _ = builder.WriteString("\n🎯 Это твой первый рекорд! Начнем отслеживать прогресс!")
	}

	return builder.String()
}

func FormatProgressHistory(history []repository.MaxRepsHistoryItem) string {
	if len(history) == 0 {
		return "📊 История прогресса пуста.\nИспользуй \"🎯 Тест максимальных отжиманий\", чтобы начать отслеживать прогресс!"
	}

	var builder strings.Builder
	_, _ = builder.WriteString("📈 Твоя история прогресса максимальных отжиманий:\n\n")

	for i := 0; i < len(history); i++ {
		item := history[len(history)-1-i]

		_, _ = fmt.Fprintf(
			&builder,
			"%d. %s → %d отжиманий\n",
			i+1,
			item.Date.Format("02.01.2006"),
			item.MaxReps,
		)

	}

	// Анализ общего прогресса
	if len(history) > 1 {
		first := history[len(history)-1].MaxReps
		last := history[0].MaxReps
		progress := last - first

		_, _ = builder.WriteString("\n📊 Общий прогресс: ")
		switch {
		case progress > 0:
			_, _ = fmt.Fprintf(&builder, "+%d отжиманий! 🚀", progress)
		case progress < 0:
			_, _ = fmt.Fprintf(&builder, "%d отжиманий 📉", progress)
		default:
			builder.WriteString("стабильно! 🎯")
		}
	}

	return builder.String()
}

func GenerateProgressBar(current, total, barWidth int) string {
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

	return fmt.Sprintf("Прогресс за день: [%s] %d%%%s", bar, percentText, suffix)
}

// formatTimeUnit универсальная функция для склонения числительных
func formatTimeUnit(value int, one, two, five string) string {
	if value == 0 {
		return ""
	}

	lastDigit := value % 10
	lastTwoDigits := value % 100

	// Исключения для 11-14
	if lastTwoDigits >= 11 && lastTwoDigits <= 14 {
		return fmt.Sprintf("%d %s", value, five)
	}

	switch lastDigit {
	case 1:
		return fmt.Sprintf("%d %s", value, one)
	case 2, 3, 4:
		return fmt.Sprintf("%d %s", value, two)
	default:
		return fmt.Sprintf("%d %s", value, five)
	}
}

// FormatTimesWord склоняет слово "раз"
func FormatTimesWord(n int) string {
	return formatTimeUnit(n, "раз", "раза", "раз")
}
