package models

import "time"

type LeaguePatchNote struct {
	PatchNoteID int64 `gorm:"column:patch_note_id;primaryKey;autoIncrement"`

	NewsID int64 `gorm:"column:news_id;not null;index"`

	Title        *string `gorm:"column:title;type:varchar(255)"`
	Summary      *string `gorm:"column:summary;type:text"`
	ChangeDetail *string `gorm:"column:change_detail;type:text"`

	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoCreateTime;autoUpdateTime"`

	RiotNews RiotNews `gorm:"foreignKey:NewsID;references:NewsID;constraint:OnDelete:CASCADE;"`
}

func (LeaguePatchNote) TableName() string {
	return "league_patch_note"
}
