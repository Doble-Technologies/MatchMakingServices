package views

type FriendDetail struct {
	UserID   uint64 `gorm:"type:bigint;not null" json:"user_id"`
	Username string `gorm:"type:text;" json:"username"`
}
