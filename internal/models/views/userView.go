package views

type UserView struct {
	UserID   uint64 `gorm:"primaryKey;bigint;not null" json:"user_id"`
	Username string `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Xp       uint64 `gorm:"type:bigint;not null" json:"xp"`
	Avatar   string `gorm:"type:text;" json:"avatar"`
	Bio      string `gorm:"type:text;" json:"bio"`
}
