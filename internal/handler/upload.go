package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/psds-microservice/data-channel-service/internal/service"
)

const maxMultipartMemory = 32 << 20 // 32 MiB

// UploadFileMultipart обрабатывает POST /data/file с multipart/form-data (session_id, user_id, file).
// Возвращает JSON: {"id": "...", "filename": "...", "url": "..."} для совместимости с тестами и клиентами.
func UploadFileMultipart(dataSvc *service.DataService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			http.Error(w, "invalid multipart form", http.StatusBadRequest)
			return
		}
		sessionIDStr := r.FormValue("session_id")
		userIDStr := r.FormValue("user_id")
		if sessionIDStr == "" || userIDStr == "" {
			http.Error(w, "session_id and user_id required", http.StatusBadRequest)
			return
		}
		sessionID, err := uuid.Parse(sessionIDStr)
		if err != nil {
			http.Error(w, "invalid session_id", http.StatusBadRequest)
			return
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file required", http.StatusBadRequest)
			return
		}
		_ = file.Close()
		filename := header.Filename
		if filename == "" {
			filename = "file"
		}
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		f, err := dataSvc.SaveFile(sessionID, userID, filename, contentType, header.Size, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":       f.ID.String(),
			"filename": f.Filename,
			"url":      "/data/file/" + f.ID.String(),
		})
	}
}
