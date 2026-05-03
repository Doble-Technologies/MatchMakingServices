package models

import "time"

//The friend table relation

type Friend struct {
	UserID       uint64    `gorm:"type:bigint;not null" json:"user_id"`
	FriendUserID string    `gorm:"type:bigint;not null" json:"friend_user_id"`
	Status       string    `gorm:"type:text;" json:"status"`
	CreatedAt    time.Time `gorm:"autoUpdateTime" json:"created_at"`
}
