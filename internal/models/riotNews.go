package models

import "time"

type RiotNews struct {
	NewsID      int64     `gorm:"column:news_id;primaryKey;autoIncrement" json:"newsId"`
	Title       string    `gorm:"column:title;type:text;not null" json:"title"`
	Category    string    `gorm:"column:category;type:varchar(100);not null" json:"category"`
	PublishedAt time.Time `gorm:"column:published_at;not null" json:"publishedAt"`
	Link        string    `gorm:"column:link;type:text;uniqueIndex;not null" json:"link"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	ImageUrl    string    `gorm:"column:image_url;type:varchar("`
}

// TableName overrides the default pluralization.
func (RiotNews) TableName() string {
	return "riot_news"
}
