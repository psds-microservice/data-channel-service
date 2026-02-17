package grpc

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/psds-microservice/data-channel-service/internal/model"
	"github.com/psds-microservice/data-channel-service/internal/service"
	"github.com/psds-microservice/data-channel-service/pkg/gen/data_channel_service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Deps — зависимости gRPC-сервера (D: зависимость от абстракций).
type Deps struct {
	Data service.DataServicer
}

// Server implements data_channel_service.DataChannelServiceServer
type Server struct {
	data_channel_service.UnimplementedDataChannelServiceServer
	Deps
}

// NewServer создаёт gRPC-сервер с внедрёнными сервисами
func NewServer(deps Deps) *Server {
	return &Server{Deps: deps}
}

func (s *Server) mapError(err error) error {
	if err == nil {
		return nil
	}
	log.Printf("grpc: error: %v", err)
	return status.Error(codes.Internal, err.Error())
}

func toProtoDataMessage(msg *model.ChannelMessage) *data_channel_service.DataMessage {
	if msg == nil {
		return nil
	}
	return &data_channel_service.DataMessage{
		Id:       msg.ID.String(),
		SenderId: msg.UserID.String(),
		Content:  string(msg.Payload),
		Type:     msg.Kind,
	}
}

func (s *Server) GetHistory(ctx context.Context, req *data_channel_service.GetHistoryRequest) (*data_channel_service.GetHistoryResponse, error) {
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid session_id")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 100
	}
	messages, err := s.Data.GetHistory(sessionID, limit)
	if err != nil {
		return nil, s.mapError(err)
	}
	protoMessages := make([]*data_channel_service.DataMessage, len(messages))
	for i, msg := range messages {
		protoMessages[i] = toProtoDataMessage(&msg)
	}
	return &data_channel_service.GetHistoryResponse{
		Messages: protoMessages,
	}, nil
}

func (s *Server) UploadFile(ctx context.Context, req *data_channel_service.UploadFileRequest) (*data_channel_service.UploadFileResponse, error) {
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid session_id")
	}
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	if req.GetFilename() == "" {
		return nil, status.Error(codes.InvalidArgument, "filename is required")
	}
	if len(req.GetContent()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "content is required")
	}
	// Сохраняем файл (в реальности нужно сохранить в storage и вернуть путь)
	// Пока используем пустой storagePath
	file, err := s.Data.SaveFile(sessionID, userID, req.GetFilename(), "application/octet-stream", int64(len(req.GetContent())), "")
	if err != nil {
		return nil, s.mapError(err)
	}
	return &data_channel_service.UploadFileResponse{
		FileId: file.ID.String(),
		Url:    "/data/file/" + file.ID.String(),
	}, nil
}
