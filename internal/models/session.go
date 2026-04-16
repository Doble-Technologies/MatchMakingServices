package models

import "time"

type Session struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement:true" json:"id"`
	UserID    uint64    `gorm:"type:bigint;not null" json:"user_id"` //BIG INT NOT VARCHAR
	Token     string    `gorm:"type:varchar(255);not null" json:"token"`
	ExpiresAt time.Time `gorm:"autoCreateTime" json:"expires_at"`
	CreatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
