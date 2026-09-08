package ws

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/renos-project/renos/core/internal/auth"
)

var allowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:8443",
	"https://localhost:8443",
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Allow direct WS clients (CLI, agent)
	}
	if u, err := url.Parse(origin); err == nil && sameHost(u.Host, r.Host) {
		return true
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return strings.HasPrefix(origin, "https://") && strings.HasSuffix(origin, ".renos.local")
}

func sameHost(originHost, requestHost string) bool {
	if originHost == "" || requestHost == "" {
		return false
	}
	if strings.EqualFold(originHost, requestHost) {
		return true
	}
	originURL, originErr := url.Parse("//" + originHost)
	requestURL, requestErr := url.Parse("//" + requestHost)
	if originErr != nil || requestErr != nil {
		return false
	}
	return originURL.Hostname() != "" &&
		requestURL.Hostname() != "" &&
		strings.EqualFold(originURL.Hostname(), requestURL.Hostname())
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

type connectionTracker struct {
	mu    sync.Mutex
	count map[string]int
}

var connTracker = &connectionTracker{count: make(map[string]int)}

var maxConnsPerUser = 10

func SetMaxConnsPerUser(n int) {
	if n > 0 {
		maxConnsPerUser = n
	}
}

func (ct *connectionTracker) tryAcquire(userID string) bool {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if ct.count[userID] >= maxConnsPerUser {
		return false
	}
	ct.count[userID]++
	return true
}

func (ct *connectionTracker) release(userID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.count[userID]--
	if ct.count[userID] <= 0 {
		delete(ct.count, userID)
	}
}

func ServeWS(hub *Hub, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(jwtSecret, tokenStr)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !connTracker.tryAcquire(claims.UserID) {
			http.Error(w, "Too many connections", http.StatusTooManyRequests)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			connTracker.release(claims.UserID)
			slog.Warn("ws upgrade failed", "error", err, "origin", r.Header.Get("Origin"), "host", r.Host)
			return
		}

		client := &Client{
			hub:      hub,
			conn:     conn,
			send:     make(chan []byte, 256),
			userID:   claims.UserID,
			userRole: claims.Role,
		}

		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
