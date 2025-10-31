package service

import (
	"fmt"
	"math"
	"strings"
)
	
const (
	// Базовые ограничения
	MinDailyPushups       = 40   // Минимальная дневная норма
	MaxDailyPushups       = 250  // Максимальный безопасный предел
	AbsoluteMaxPushups    = 500  // Абсолютный максимум для профессионалов

	// Уровни подготовки (макс. отжимания за подход)
	StartingThreshold     = 10   // ≤10 - стартовый уровень
	BeginnerThreshold     = 20   // ≤20 - начальный уровень
	IntermediateThreshold = 30   // ≤30 - средний уровень
	AdvancedThreshold     = 40   // ≤40 - продвинутый
	ExpertThreshold       = 50   // ≤51+ - эксперт

	// Коэффициенты для расчета нормы
	BaseCoefficient      = 5.0   // Стартовый коэффициент
	CoefficientStep      = 0.025 // Шаг уменьшения коэффициента
	MinCoefficient       = 2.5   // Минимальный коэффициент

	// Рекомендации ACSM
	ACSMIntensityRatio   = 0.7   // 70% от максимума за подход
	RecoveryHours        = 48    // Часы отдыха между тренировками
)

// CalculateDailyNorm рассчитывает дневную норму с уменьшающимся коэффициентом
// Аргументы:
//   maxReps - максимальное количество отжиманий за один подход
// Возвращает:
//   дневную норму (целое число, кратное 5)
func CalculateDailyNorm(maxReps int) int {
	// Защита от нереалистичных значений
	if maxReps <= 0 {
		return MinDailyPushups
	}
	if maxReps > 100 {
		return MaxDailyPushups
	}

	// Прогрессивная формула с уменьшающимся коэффициентом
	coefficient := getSmoothCoefficient(maxReps)
	rawNorm := float64(maxReps) * coefficient

	// Округление до ближайшего кратного 5
	norm := int(math.Round(rawNorm/5)) * 5

	// Применение граничных условий
	return clamp(norm, MinDailyPushups, MaxDailyPushups)
}


func getSmoothCoefficient(maxReps int) float64 {
    // Определяем базовый коэффициент на основе средних значений ACSM
	// Согласно рекомендациям ACSM (American College of Sports Medicine)
    var base float64
    
    switch {
    case maxReps <= StartingThreshold:  // Новички (ACSM: 30-50)
        base = 4  // (30+50)/2 / 10 = 4
    case maxReps <= BeginnerThreshold:  // Начальный уровень (ACSM: 40-60)
        base = 2.5  // (40+60)/2 / 20 = 2.5
    case maxReps <= IntermediateThreshold:  // Средний уровень (ACSM: 60-80)
        base = 2.33 // (60+80)/2 / 30 ≈ 2.33
    case maxReps <= AdvancedThreshold:  // Интенсивные тренировки (ACSM: 80-120)
        base = 2.5  // (80+120)/2 / 40 = 2.5
    case maxReps <= ExpertThreshold:  // Продвинутые (ACSM: 120-150)
        base = 2.7  // (120+150)/2 / 50 = 2.7
    default:            // Профессионалы (ACSM: 150-250)
        base = 2.5  // (150+250)/2 / 80 ≈ 2.5 (для maxReps=80)
    }
    
    // Плавное уменьшение коэффициента между границами
    smoothBase := BaseCoefficient - CoefficientStep*float64(maxReps)
    
    // Компромисс между плавностью и соответствием ACSM
    finalCoeff := (base + smoothBase) / 2
    
    return math.Max(math.Min(finalCoeff, BaseCoefficient), MinCoefficient)
}

// Вспомогательная функция для ограничения диапазона
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}


// GetUserRank определяет ранг пользователя на основе его maxReps
// Константы для порогов рангов
const (
    RankSleepyFly      = 0
    RankSprout         = 5 // +5
    RankWorker         = 10 // +5
    RankTrainee        = 15 // +5
    RankRocket         = 20 // +5
    RankKnight         = 25 // +5
    RankImpenetrable   = 30 // +5
    RankThunder        = 40 // +10
    RankAdept          = 50 // +10
    RankGravity        = 65 // +15
    RankLegend         = 80 // +15
    LordOfPushUps      = 100 // +20
)

type UserRank struct {
    threshold int
    rank string
} 

// Ранги пользователя в порядке возрастания
var userRanks = []UserRank{
    {RankSleepyFly, "💤 Сонная муха"},
    {RankSprout, "🌱 Росток силы"},
    {RankWorker, "🐜 Трудяга"},
    {RankTrainee, "🚀 Стажёр космоса"},
    {RankRocket, "🚀 Ракета-носитель"},
    {RankKnight, "⚔️ Рыцарь света"},
    {RankImpenetrable, "🛡️ Непробиваемый"},
    {RankThunder, "⚡ Гроза пола"},
    {RankAdept, "🏹 Адепт упорства"},
    {RankGravity, "🌌 Победитель гравитации"},
    {RankLegend, "🏆 Легенда горизонтов"},
    {LordOfPushUps, "🌟 ВЛАСТЕЛИН ОТЖИМАНИЙ"},
}



// GetUserRank определяет ранг пользователя на основе его maxReps
func GetUserRank(maxReps int) string {
    for i := len(userRanks) - 1; i >= 0; i-- {
        if maxReps >= userRanks[i].threshold {
            return userRanks[i].rank
        }
    }
    return "🌟 ВЛАСТЕЛИН ОТЖИМАНИЙ"
}

// GetRepsToNextRank возвращает количество отжиманий до следующего ранга
func GetRepsToNextRank(maxReps int) int {
    currentRankIndex := -1
    
    // Находим индекс текущего ранга
    for i := len(userRanks) - 1; i >= 0; i-- {
        if maxReps >= userRanks[i].threshold {
            currentRankIndex = i
            break
        }
    }
    
    // Если текущий ранг - последний или пользователь вышел за пределы
    if currentRankIndex == len(userRanks)-1 || maxReps > LordOfPushUps {
        return 0
    }
    
    // Если не нашли ранг (маловероятно, но на всякий случай)
    if currentRankIndex == -1 {
        return RankSprout - maxReps
    }
    
    // Следующий ранг
    nextRank := userRanks[currentRankIndex+1]
    return nextRank.threshold - maxReps
}



// CalculateNextTarget рассчитывает, на сколько минимум нужно увеличить maxReps на новой неделе.
// Аргумент:
//   currentMaxReps - текущий максимальный показатель за один подход
// Возвращает:
//   рекомендуемое минимальное увеличение (целое число)
func CalculateNextTarget(currentMaxReps int) int {
    // Защита от нереалистичных значений
    if currentMaxReps < 0 {
        return 1
    }

    // Определение шага прогрессии в зависимости от текущего уровня
    switch {
    case currentMaxReps < 5: 
        // Уровень: Начинающий (очень низкий)
        return 1
    case currentMaxReps >= 5 && currentMaxReps < 15:
        // Уровень: Развивающийся (низкий)
        return 2
    case currentMaxReps >= 15 && currentMaxReps < 30:
        // Уровень: Средний
        return 3
    case currentMaxReps >= 30 && currentMaxReps < 50:
        // Уровень: Продвинутый
        return 2 // Уменьшаем шаг, так как прогресс замедляется
    case currentMaxReps >= 50 && currentMaxReps < 100:
        // Уровень: Опытный
        return 1
    default:
        // Уровень: Мастер (100+)
        // На очень высоком уровне регулярное увеличение недельного максимума затруднительно.
        // Целесообразнее работать над другими параметрами (взрывная сила, вариации).
        return 0
    }
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

// FormatTimesWord склоняет слово "раз" (адаптированная версия)
func FormatTimesWord(n int) string {
    return formatTimeUnit(n, "раз", "раза", "раз")
}

// FormatHoursCompact склоняет слово "час"
func FormatHoursCompact(hours int) string {
    return formatTimeUnit(hours, "час", "часа", "часов")
}

// FormatDaysCompact склоняет слово "день"  
func FormatDaysCompact(days int) string {
    return formatTimeUnit(days, "день", "дня", "дней")
}