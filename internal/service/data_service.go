package service

import (
	"github.com/google/uuid"
	"github.com/psds-microservice/data-channel-service/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DataService struct {
	db *gorm.DB
}

func NewDataService(db *gorm.DB) *DataService {
	return &DataService{db: db}
}

func (s *DataService) AppendMessage(sessionID, userID uuid.UUID, kind string, payload datatypes.JSON) error {
	return s.db.Create(&model.ChannelMessage{
		SessionID: sessionID,
		UserID:    userID,
		Kind:      kind,
		Payload:   payload,
	}).Error
}

func (s *DataService) GetHistory(sessionID uuid.UUID, limit int) ([]model.ChannelMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	var list []model.ChannelMessage
	err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Limit(limit).Find(&list).Error
	return list, err
}

func (s *DataService) SaveFile(sessionID, userID uuid.UUID, filename, contentType string, sizeBytes int64, storagePath string) (*model.ChannelFile, error) {
	f := &model.ChannelFile{
		SessionID:   sessionID,
		UserID:      userID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		StoragePath: storagePath,
	}
	return f, s.db.Create(f).Error
}
