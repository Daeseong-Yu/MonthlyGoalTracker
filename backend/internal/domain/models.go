package domain

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"size:100;not null;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Goal struct {
	ID        uint       `gorm:"primaryKey"`
	UserID    uint       `gorm:"not null;index"`
	User      User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Title     string     `gorm:"size:100;not null"`
	StartDate time.Time  `gorm:"type:date;not null;index;check:chk_goals_date_range,end_date IS NULL OR start_date <= end_date"`
	EndDate   *time.Time `gorm:"type:date;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DailyMemo struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_daily_memos_user_date"`
	User      User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Date      time.Time `gorm:"type:date;not null;uniqueIndex:idx_daily_memos_user_date"`
	Memo      string    `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GoalCheck struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index:idx_goal_checks_user_date"`
	User      User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GoalID    uint      `gorm:"not null;uniqueIndex:idx_goal_checks_goal_date"`
	Goal      Goal      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Date      time.Time `gorm:"type:date;not null;uniqueIndex:idx_goal_checks_goal_date;index:idx_goal_checks_user_date"`
	CreatedAt time.Time
}
