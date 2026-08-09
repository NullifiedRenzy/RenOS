package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

// validateFilePath checks that a file path does not contain traversal components.
func validateFilePath(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}
	if strings.Contains(path, "\x00") {
		return false
	}
	return true
}

type createTransferPayload struct {
	TransferType string `json:"transfer_type"`
	Filename     string `json:"filename"`
	RemotePath   string `json:"remote_path"`
	TotalSize    int64  `json:"total_size"`
	ChunkSize    int    `json:"chunk_size"`
}

func createFileTransferHandler(transfers *db.FileTransferRepository, devices *db.DeviceRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		if _, err := devices.FindByID(r.Context(), deviceID); err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}

		claims := r.Context().Value(claimsKey).(*auth.Claims)

		var req createTransferPayload
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}
		if req.TransferType != "upload" && req.TransferType != "download" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "transfer_type must be upload or download")
			return
		}
		if req.Filename == "" || req.RemotePath == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "filename and remote_path required")
			return
		}
		if !validateFilePath(req.RemotePath) || !validateFilePath(req.Filename) {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid path: traversal components not allowed")
			return
		}
		if req.TotalSize <= 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "total_size must be positive")
			return
		}
		if req.ChunkSize <= 0 {
			req.ChunkSize = 1048576 // 1MB default
		}
		totalChunks := int((req.TotalSize + int64(req.ChunkSize) - 1) / int64(req.ChunkSize))

		transfer, err := transfers.Create(r.Context(), deviceID, claims.UserID, req.TransferType, req.Filename, req.RemotePath, req.TotalSize, req.ChunkSize, totalChunks)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create file transfer")
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "file_transfer."+req.TransferType, "device", deviceID)
		hub.Broadcast(ws.Message{
			Type:     "file_transfer.created",
			DeviceID: deviceID,
			Data: map[string]interface{}{
				"transfer_id":   transfer.ID,
				"transfer_type": transfer.TransferType,
				"filename":      transfer.Filename,
				"remote_path":   transfer.RemotePath,
				"total_chunks":  transfer.TotalChunks,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_transfer": transfer,
		})
	}
}

func getTransferHandler(transfers *db.FileTransferRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transferID := chi.URLParam(r, "transferId")
		if transferID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "transfer id required")
			return
		}

		transfer, err := transfers.FindByID(r.Context(), transferID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File transfer not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_transfer": transfer,
		})
	}
}

func listTransfersHandler(transfers *db.FileTransferRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		limit := 50
		offset := 0
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if v := r.URL.Query().Get("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		list, total, err := transfers.ListByDevice(r.Context(), deviceID, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list file transfers")
			return
		}
		if list == nil {
			list = []*db.FileTransfer{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_transfers": list,
			"total":          total,
			"limit":          limit,
			"offset":         offset,
		})
	}
}

func uploadChunkHandler(transfers *db.FileTransferRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transferID := chi.URLParam(r, "transferId")
		chunkNumStr := chi.URLParam(r, "chunkNum")
		chunkNum, err := strconv.Atoi(chunkNumStr)
		if err != nil || chunkNum < 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid chunk number")
			return
		}

		transfer, err := transfers.FindByID(r.Context(), transferID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File transfer not found")
			return
		}
		if transfer.Status != "pending" && transfer.Status != "uploading" {
			writeError(w, http.StatusConflict, "CONFLICT", "Transfer is not in an uploadable state")
			return
		}
		if chunkNum >= transfer.TotalChunks {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "chunk number exceeds total chunks")
			return
		}

		// Update status to uploading on first chunk
		if transfer.Status == "pending" {
			if _, err := transfers.UpdateStatus(r.Context(), transferID, "uploading"); err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update transfer status")
				return
			}
		}

		// Read chunk data (max 10MB)
		r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to read chunk data")
			return
		}

		// Compute checksum
		h := sha256.Sum256(data)
		checksum := hex.EncodeToString(h[:])

		if err := transfers.StoreChunk(r.Context(), transferID, chunkNum, data, checksum); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to store chunk")
			return
		}

		if _, err := transfers.IncrementChunks(r.Context(), transferID, 1); err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update chunk count")
			return
		}

		audits.Create(r.Context(), "user", "", "file_transfer.chunk_upload", "file_transfer", transferID)
		hub.Broadcast(ws.Message{
			Type:     "file_transfer.chunk_uploaded",
			DeviceID: transfer.DeviceID,
			Data: map[string]interface{}{
				"transfer_id":     transferID,
				"chunk_number":    chunkNum,
				"uploaded_chunks": transfer.UploadedChunks + 1,
				"total_chunks":    transfer.TotalChunks,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"chunk_number":    chunkNum,
			"checksum":        checksum,
			"uploaded_chunks": transfer.UploadedChunks + 1,
			"total_chunks":    transfer.TotalChunks,
		})
	}
}

func downloadChunkHandler(transfers *db.FileTransferRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transferID := chi.URLParam(r, "transferId")
		chunkNumStr := chi.URLParam(r, "chunkNum")
		chunkNum, err := strconv.Atoi(chunkNumStr)
		if err != nil || chunkNum < 0 {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid chunk number")
			return
		}

		transfer, err := transfers.FindByID(r.Context(), transferID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File transfer not found")
			return
		}
		if transfer.Status == "pending" {
			if _, err := transfers.UpdateStatus(r.Context(), transferID, "downloading"); err != nil {
				writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update transfer status")
				return
			}
		}

		data, err := transfers.GetChunk(r.Context(), transferID, chunkNum)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Chunk not found")
			return
		}

		h := sha256.Sum256(data)
		checksum := hex.EncodeToString(h[:])

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Chunk-Number", strconv.Itoa(chunkNum))
		w.Header().Set("X-Chunk-Checksum", checksum)
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Write(data)
	}
}

func assembleTransferHandler(transfers *db.FileTransferRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transferID := chi.URLParam(r, "transferId")
		if transferID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "transfer id required")
			return
		}

		transfer, err := transfers.FindByID(r.Context(), transferID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "File transfer not found")
			return
		}

		if transfer.UploadedChunks < transfer.TotalChunks {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "not all chunks uploaded yet")
			return
		}

		transfer, err = transfers.UpdateStatus(r.Context(), transferID, "assembling")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to update transfer status")
			return
		}

		transfer, err = transfers.UpdateStatus(r.Context(), transferID, "completed")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to complete transfer")
			return
		}

		audits.Create(r.Context(), "user", "", "file_transfer.completed", "file_transfer", transferID)
		hub.Broadcast(ws.Message{
			Type:     "file_transfer.completed",
			DeviceID: transfer.DeviceID,
			Data: map[string]interface{}{
				"transfer_id": transferID,
				"filename":    transfer.Filename,
				"remote_path": transfer.RemotePath,
				"total_size":  transfer.TotalSize,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_transfer": transfer,
		})
	}
}

func cancelTransferHandler(transfers *db.FileTransferRepository, audits *db.AuditRepository, hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transferID := chi.URLParam(r, "transferId")
		if transferID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "transfer id required")
			return
		}

		transfer, err := transfers.UpdateStatus(r.Context(), transferID, "cancelled")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to cancel transfer")
			return
		}

		_ = transfers.DeleteChunks(r.Context(), transferID)

		audits.Create(r.Context(), "user", "", "file_transfer.cancelled", "file_transfer", transferID)
		hub.Broadcast(ws.Message{
			Type:     "file_transfer.cancelled",
			DeviceID: transfer.DeviceID,
			Data: map[string]interface{}{
				"transfer_id": transferID,
			},
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_transfer": transfer,
		})
	}
}

func getTransferForAgentHandler(transfers *db.FileTransferRepository, devices *db.DeviceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req deviceAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
			return
		}

		device, apiErr := verifyDeviceSecret(devices, req)
		if apiErr != nil {
			writeError(w, apiErr.status, apiErr.code, apiErr.message)
			return
		}

		limit := 5
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 20 {
				limit = n
			}
		}

		list, _, err := transfers.ListByDevice(r.Context(), device.ID, limit, 0)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list transfers")
			return
		}
		if list == nil {
			list = []*db.FileTransfer{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"file_transfers": list,
		})
	}
}

// FileTransferReaper periodically cancels stale file transfers.
func FileTransferReaper(transfers *db.FileTransferRepository, maxAge time.Duration) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ctx := context.Background()
		count, err := transfers.CancelStale(ctx, maxAge)
		if err == nil && count > 0 {
			slog.Info("cancelled stale file transfers", "count", count)
		}
	}
}
