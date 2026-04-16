package models

type NotificationInput struct {
	UserID string `gorm:"type:bigint;not null" json:"user_id"`
	Icon   string `gorm:"type:varchar(32);" json:"icon"`
	Title  string `gorm:"type:text;" json:"title"`
	Unread string `gorm:"type:bool;" json:"unread"`
}

// TableName overrides the default naming convention
func (NotificationInput) TableName() string {
	return "notifications"
}
