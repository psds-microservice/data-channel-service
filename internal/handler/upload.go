package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/psds-microservice/data-channel-service/internal/service"
)

const maxMultipartMemory = 32 << 20 // 32 MiB
const maxFileSizeBytes = 50 << 20   // 50 MiB
const maxFilenameLen = 255

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
		defer file.Close()
		if header.Size < 0 || header.Size > maxFileSizeBytes {
			http.Error(w, "file size exceeds limit or invalid", http.StatusBadRequest)
			return
		}
		// Path traversal protection: use only base name, no directory components.
		rawName := header.Filename
		if rawName == "" {
			rawName = "file"
		}
		filename := filepath.Base(strings.TrimSpace(rawName))
		if filename == "" || filename == "." || strings.Contains(filename, "..") {
			filename = "file"
		}
		if len(filename) > maxFilenameLen {
			filename = filename[:maxFilenameLen]
		}
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if err := dataSvc.ValidateFile(filename, header.Size, ""); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
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
