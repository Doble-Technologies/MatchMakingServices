package models

import "time"

//The friend table relation

type Friend struct {
	UserID       uint64    `gorm:"type:bigint;not null" json:"user_id"`
	FriendUserID uint64    `gorm:"type:bigint;not null" json:"friend_user_id"`
	Status       string    `gorm:"type:text;" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}
