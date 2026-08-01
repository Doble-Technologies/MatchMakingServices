package views

import "time"

type UserProfile struct {
	UserID    uint64    `gorm:"primaryKey;bigint" json:"user_id"`
	Username  string    `gorm:"type:text;" json:"username"`
	Xp        uint64    `gorm:"type:bigint;not null" json:"xp"`
	Avatar    string    `gorm:"type:text;" json:"avatar"`
	Bio       string    `gorm:"type:text;" json:"bio"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
