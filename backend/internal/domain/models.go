package domain

import "time"

type Goal struct {
	ID        uint       `gorm:"primaryKey"`
	Title     string     `gorm:"size:100;not null"`
	StartDate time.Time  `gorm:"type:date;not null;index;check:chk_goals_date_range,end_date IS NULL OR start_date <= end_date"`
	EndDate   *time.Time `gorm:"type:date;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DailyMemo struct {
	ID        uint      `gorm:"primaryKey"`
	Date      time.Time `gorm:"type:date;not null;uniqueIndex"`
	Memo      string    `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GoalCheck struct {
	ID        uint      `gorm:"primaryKey"`
	GoalID    uint      `gorm:"not null;uniqueIndex:idx_goal_checks_goal_date"`
	Goal      Goal      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Date      time.Time `gorm:"type:date;not null;uniqueIndex:idx_goal_checks_goal_date"`
	CreatedAt time.Time
}
