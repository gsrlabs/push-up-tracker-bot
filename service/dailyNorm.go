package service

import (
	"math"
)
	
const (
	// Базовые ограничения
	MinDailyPushups       = 40   // Минимальная дневная норма
	MaxDailyPushups       = 250  // Максимальный безопасный предел
	AbsoluteMaxPushups    = 500  // Абсолютный максимум для профессионалов (с предупреждением)

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
	if maxReps < 1 {
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
