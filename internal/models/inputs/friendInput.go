package inputs

type FriendInput struct {
	UserID       uint64 `gorm:"type:bigint;not null" json:"user_id"`
	FriendUserID uint64 `gorm:"type:bigint;not null" json:"friend_user_id"`
	Status       string `gorm:"type:text;" json:"status"`
}

func (FriendInput) TableName() string {
	return "friends"
}
