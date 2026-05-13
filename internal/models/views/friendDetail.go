package views

type FriendDetail struct {
	UserID   uint64 `gorm:"type:bigint;not null" json:"user_id"`
	UserName string `gorm:"type:text;" json:"user_name"`
}
