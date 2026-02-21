package service

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/psds-microservice/data-channel-service/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const maxFileSizeBytes = 50 << 20   // 50 MiB
const maxFilenameLen = 255
const maxStoragePathLen = 2048

// DataServicer — интерфейс для gRPC Deps (Dependency Inversion).
type DataServicer interface {
	GetHistory(sessionID uuid.UUID, limit int) ([]model.ChannelMessage, error)
	SaveFile(sessionID, userID uuid.UUID, filename, contentType string, sizeBytes int64, storagePath string) (*model.ChannelFile, error)
}

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

// ValidateFile checks filename, size and storagePath for security (path traversal, size limit).
func (s *DataService) ValidateFile(filename string, sizeBytes int64, storagePath string) error {
	if sizeBytes < 0 || sizeBytes > maxFileSizeBytes {
		return errors.New("file size exceeds limit or invalid")
	}
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "" || base == "." || strings.Contains(base, "..") {
		return errors.New("invalid filename")
	}
	if len(base) > maxFilenameLen {
		return errors.New("filename too long")
	}
	if storagePath != "" {
		clean := filepath.Clean(storagePath)
		if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
			return errors.New("invalid storage path")
		}
		if len(clean) > maxStoragePathLen {
			return errors.New("storage path too long")
		}
	}
	return nil
}

func (s *DataService) SaveFile(sessionID, userID uuid.UUID, filename, contentType string, sizeBytes int64, storagePath string) (*model.ChannelFile, error) {
	if err := s.ValidateFile(filename, sizeBytes, storagePath); err != nil {
		return nil, err
	}
	// Use base name only for storage (path traversal protection).
	safeName := filepath.Base(strings.TrimSpace(filename))
	if safeName == "" || safeName == "." {
		safeName = "file"
	}
	if len(safeName) > maxFilenameLen {
		safeName = safeName[:maxFilenameLen]
	}
	f := &model.ChannelFile{
		SessionID:   sessionID,
		UserID:      userID,
		Filename:    safeName,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		StoragePath: storagePath,
	}
	return f, s.db.Create(f).Error
}
