package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDailyNorm(t *testing.T) {
	tests := []struct {
		maxReps int
		want    int
	}{
		{-10, MinDailyPushups},
		{0, MinDailyPushups},
		{5, 40},
		{10, 45},
		{15, 55},
		{25, 85},
		{35, 115},
		{50, 160},
		{80, 220},
		{150, MaxDailyPushups}, // больше 100
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := CalculateDailyNorm(tt.maxReps)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetUserRank(t *testing.T) {
	tests := []struct {
		maxReps int
		want    string
	}{
		{0, "💤 Сонная муха"},
		{5, "🌱 Росток силы"},
		{12, "🐜 Трудяга"},
		{18, "🚀 Стажёр космоса"},
		{22, "🚀 Ракета-носитель"},
		{40, "⚡ Гроза пола"},
		{100, "🌟 ВЛАСТЕЛИН ОТЖИМАНИЙ"},
		{150, "🌟 ВЛАСТЕЛИН ОТЖИМАНИЙ"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := GetUserRank(tt.maxReps)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetRepsToNextRank(t *testing.T) {
	tests := []struct {
		maxReps int
		want    int
	}{
		{0, 5},
		{5, 5},
		{10, 5},
		{25, 5},  // реально до LordOfPushUps = 100
		{100, 0}, // последний ранг
		{150, 0}, // за пределами
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := GetRepsToNextRank(tt.maxReps)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateNextTarget(t *testing.T) {
	tests := []struct {
		currentMax int
		want       int
	}{
		{-5, 1},
		{0, 1},
		{4, 1},
		{5, 2},
		{14, 2},
		{15, 3},
		{29, 3},
		{30, 2},
		{49, 2},
		{50, 1},
		{99, 1},
		{100, 0},
		{200, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := CalculateNextTarget(tt.currentMax)
			assert.Equal(t, tt.want, got)
		})
	}
}
