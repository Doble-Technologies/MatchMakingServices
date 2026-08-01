package models

import "time"

type UserDetail struct {
	UserID    uint64    `gorm:"primaryKey;bigint" json:"user_id"`
	Xp        uint64    `gorm:"type:bigint;not null" json:"xp"`
	Avatar    string    `gorm:"type:text;" json:"avatar"`
	Bio       string    `gorm:"type:text;" json:"bio"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
