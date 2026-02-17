package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ChannelMessage struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID uuid.UUID      `gorm:"type:uuid;not null;index" json:"session_id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Kind      string         `gorm:"type:varchar(32);not null;default:'chat'" json:"kind"`
	Payload   datatypes.JSON `gorm:"type:jsonb;not null" json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

func (ChannelMessage) TableName() string { return "channel_messages" }

type ChannelFile struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SessionID   uuid.UUID `gorm:"type:uuid;not null;index" json:"session_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Filename    string    `gorm:"type:varchar(255);not null" json:"filename"`
	ContentType string    `gorm:"type:varchar(128)" json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StoragePath string    `gorm:"type:varchar(512)" json:"storage_path"`
	CreatedAt   time.Time `json:"created_at"`
}

func (ChannelFile) TableName() string { return "channel_files" }
