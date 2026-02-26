package model

import "time"

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
	History    []MaxRepsHistoryItem
	Record     *MaxRepsHistoryItem
}

type FullStatViewModel struct {
	TodayTotal       int
	TotalAllTime     int
	DailyNorm        int
	FirstWorkoutDate *time.Time
	Leaderboard      []LeaderboardItem
}

type MaxRepsHistoryItem struct {
	Date    time.Time
	MaxReps int
}

type LeaderboardItem struct {
	Rank     int
	Username string
	Count    int
}

