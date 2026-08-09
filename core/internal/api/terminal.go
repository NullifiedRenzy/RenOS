package api

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	mathrand "math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"

	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

var terminalUpgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		// Only allow exact localhost matches, not wildcard subdomains
		return strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "https://localhost:") ||
			origin == "http://localhost" || origin == "https://localhost"
	},
}

// --- NATS message types ---

type terminalOpenMsg struct {
	SessionID string `json:"session_id"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
	Shell     string `json:"shell"`
}

type terminalInputMsg struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"` // base64
}

type terminalOutputMsg struct {
	SessionID string `json:"session_id"`
	Data      string `json:"data"` // base64
}

type terminalResizeMsg struct {
	SessionID string `json:"session_id"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
}

type terminalCloseMsg struct {
	SessionID string `json:"session_id"`
}

type terminalErrorMsg struct {
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
}

// --- WebSocket message types (browser ↔ Core) ---

type wsTerminalMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// --- Handlers ---

func terminalWebSocketHandler(
	terminalSessions *db.TerminalSessionRepository,
	terminalRecordings *db.TerminalRecordingRepository,
	devices *db.DeviceRepository,
	audits *db.AuditRepository,
	nc *nats.Conn,
	hub *ws.Hub,
	jwtSecret string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			http.Error(w, "device id required", http.StatusBadRequest)
			return
		}

		// Authenticate
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ValidateToken(jwtSecret, tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Check device exists and is online
		device, err := devices.FindByID(r.Context(), deviceID)
		if err != nil {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		if device.Status != "online" {
			http.Error(w, "device is not online", http.StatusConflict)
			return
		}

		// Parse initial size
		cols := 80
		rows := 24
		if v := r.URL.Query().Get("cols"); v != "" {
			if n, err := parseTerminalInt(v); err == nil && n > 0 && n <= 500 {
				cols = n
			}
		}
		if v := r.URL.Query().Get("rows"); v != "" {
			if n, err := parseTerminalInt(v); err == nil && n > 0 && n <= 200 {
				rows = n
			}
		}
		shell := r.URL.Query().Get("shell")

		// Generate session token
		sessionToken := generateSessionToken()

		// Create DB session
		session, err := terminalSessions.Create(
			r.Context(), deviceID, claims.UserID, sessionToken, shell,
			r.RemoteAddr, r.UserAgent(), cols, rows,
		)
		if err != nil {
			slog.Error("failed to create terminal session", "error", err)
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}

		audits.Create(r.Context(), "user", claims.UserID, "terminal.open", "device", deviceID)

		// Upgrade to WebSocket
		conn, err := terminalUpgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("terminal ws upgrade failed", "error", err)
			terminalSessions.UpdateStatus(r.Context(), session.ID, "error", "ws upgrade failed")
			return
		}

		defer func() {
			conn.Close()
			terminalSessions.UpdateStatus(r.Context(), session.ID, "closed", "client disconnected")
			// Notify agent to close PTY
			closeMsg := terminalCloseMsg{SessionID: session.ID}
			nc.Publish("terminal.close."+deviceID, mustMarshal(closeMsg))
			slog.Info("terminal session closed", "session_id", session.ID, "device_id", deviceID)
		}()

		// Send session info to browser
		sessInfo := map[string]interface{}{
			"session_id": session.ID,
			"device_id":  deviceID,
			"cols":       cols,
			"rows":       rows,
		}
		sessData, _ := json.Marshal(sessInfo)
		conn.WriteMessage(websocket.TextMessage, mustMarshal(wsTerminalMsg{Type: "session", Data: sessData}))

		// Open PTY on agent via NATS
		openMsg := terminalOpenMsg{
			SessionID: session.ID,
			Cols:      cols,
			Rows:      rows,
			Shell:     shell,
		}
		if err := nc.Publish("terminal.open."+deviceID, mustMarshal(openMsg)); err != nil {
			slog.Error("failed to publish terminal.open", "error", err)
			conn.WriteMessage(websocket.TextMessage, mustMarshal(wsTerminalMsg{Type: "error", Data: mustMarshal(map[string]string{"error": "failed to open terminal on agent"})}))
			return
		}

		// Subscribe to terminal output for this session
		outputSub, err := nc.Subscribe("terminal.output."+session.ID, func(msg *nats.Msg) {
			var out terminalOutputMsg
			if err := json.Unmarshal(msg.Data, &out); err != nil {
				return
			}

			// Record output
			if data, err := base64.StdEncoding.DecodeString(out.Data); err == nil && len(data) > 0 {
				terminalRecordings.Insert(context.Background(), session.ID, "output", data)
			}
			terminalSessions.UpdateLastActive(context.Background(), session.ID)

			// Forward to browser
			wsMsg := wsTerminalMsg{
				Type: "output",
				Data: mustMarshal(map[string]string{"data": out.Data}),
			}
			conn.WriteMessage(websocket.TextMessage, mustMarshal(wsMsg))
		})
		if err != nil {
			slog.Error("failed to subscribe to terminal.output", "error", err)
			return
		}
		defer outputSub.Unsubscribe()

		// Subscribe to terminal errors for this session
		errorSub, err := nc.Subscribe("terminal.error."+session.ID, func(msg *nats.Msg) {
			var errMsg terminalErrorMsg
			if err := json.Unmarshal(msg.Data, &errMsg); err != nil {
				return
			}

			wsMsg := wsTerminalMsg{
				Type: "error",
				Data: mustMarshal(map[string]string{"error": errMsg.Error}),
			}
			conn.WriteMessage(websocket.TextMessage, mustMarshal(wsMsg))
		})
		if err != nil {
			slog.Error("failed to subscribe to terminal.error", "error", err)
			return
		}
		defer errorSub.Unsubscribe()

		// Read from browser WebSocket
		conn.SetReadLimit(65536)
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					slog.Warn("terminal ws read error", "error", err)
				}
				break
			}

			var wsMsg wsTerminalMsg
			if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
				continue
			}

			switch wsMsg.Type {
			case "input":
				var inputMsg struct {
					Data string `json:"data"`
				}
				if err := json.Unmarshal(wsMsg.Data, &inputMsg); err != nil {
					continue
				}

				// Record input
				if data, err := base64.StdEncoding.DecodeString(inputMsg.Data); err == nil && len(data) > 0 {
					terminalRecordings.Insert(r.Context(), session.ID, "input", data)
				}

				natsInput := terminalInputMsg{
					SessionID: session.ID,
					Data:      inputMsg.Data,
				}
				nc.Publish("terminal.input."+deviceID, mustMarshal(natsInput))

			case "resize":
				var resizeMsg struct {
					Cols int `json:"cols"`
					Rows int `json:"rows"`
				}
				if err := json.Unmarshal(wsMsg.Data, &resizeMsg); err != nil {
					continue
				}
				if resizeMsg.Cols <= 0 || resizeMsg.Cols > 500 || resizeMsg.Rows <= 0 || resizeMsg.Rows > 200 {
					continue
				}

				terminalSessions.UpdateSize(r.Context(), session.ID, resizeMsg.Cols, resizeMsg.Rows)

				natsResize := terminalResizeMsg{
					SessionID: session.ID,
					Cols:      resizeMsg.Cols,
					Rows:      resizeMsg.Rows,
				}
				nc.Publish("terminal.resize."+deviceID, mustMarshal(natsResize))

			case "close":
				// Browser-initiated close
				break
			}
		}
	}
}

// --- REST endpoints for session listing ---

func listTerminalSessionsHandler(terminalSessions *db.TerminalSessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "id")
		if deviceID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "device id required")
			return
		}

		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := parseTerminalInt(v); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		list, err := terminalSessions.ListByDevice(r.Context(), deviceID, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list terminal sessions")
			return
		}
		if list == nil {
			list = []*db.TerminalSession{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": list,
		})
	}
}

func listTerminalSessionsByUserHandler(terminalSessions *db.TerminalSessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(claimsKey).(*auth.Claims)

		limit := 50
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := parseTerminalInt(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}

		list, err := terminalSessions.ListByUser(r.Context(), claims.UserID, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list terminal sessions")
			return
		}
		if list == nil {
			list = []*db.TerminalSession{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": list,
		})
	}
}

func getTerminalSessionHandler(terminalSessions *db.TerminalSessionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionId")
		if sessionID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "session id required")
			return
		}

		session, err := terminalSessions.FindByID(r.Context(), sessionID)
		if err != nil {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Terminal session not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session": session,
		})
	}
}

func getTerminalRecordingHandler(terminalRecordings *db.TerminalRecordingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := chi.URLParam(r, "sessionId")
		if sessionID == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "session id required")
			return
		}

		recordings, err := terminalRecordings.ListBySession(r.Context(), sessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list recordings")
			return
		}
		if recordings == nil {
			recordings = []*db.TerminalRecording{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"recordings": recordings,
		})
	}
}

// --- Idle timeout reaper ---

func TerminalSessionReaper(terminalSessions *db.TerminalSessionRepository, nc *nats.Conn, timeout time.Duration) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		affected, err := terminalSessions.CloseStaleSessions(ctx, timeout)
		if err != nil {
			slog.Error("terminal session reaper failed", "error", err)
			continue
		}
		if affected > 0 {
			slog.Info("terminal sessions timed out", "count", affected)
			// Note: we don't notify agents here since they'll detect the closed NATS subscription
		}
	}
}

func parseTerminalInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func generateSessionToken() string {
	b := make([]byte, 32)
	_, err := cryptorand.Read(b)
	if err != nil {
		fallback := make([]byte, 32)
		for i := range fallback {
			fallback[i] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"[mathrand.Intn(62)]
		}
		return base64.RawURLEncoding.EncodeToString(fallback)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
