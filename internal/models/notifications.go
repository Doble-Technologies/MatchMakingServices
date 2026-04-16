package models

import "time"

type Notification struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	UserID    string    `gorm:"type:bigint;not null" json:"user_id"`
	Icon      string    `gorm:"type:varchar(32);" json:"icon"`
	Title     string    `gorm:"type:text;" json:"title"`
	CreatedAt time.Time `gorm:"autoUpdateTime" json:"created_at"`
	Unread    string    `gorm:"type:bool;" json:"unread"`
}
