package ws

import (
	"encoding/json"
	"log/slog"
	"strings"
)

type Message struct {
	Type     string      `json:"type"`
	DeviceID string      `json:"device_id"`
	Data     interface{} `json:"data,omitempty"`
	Role     string      `json:"-"` // minimum role required to receive this message
}

type Hub struct {
	clients     map[*Client]bool
	broadcast   chan Message
	register    chan *Client
	unregister  chan *Client
	clientCount chan chan int
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan Message, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clientCount: make(chan chan int),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			slog.Info("ws client connected", "user_id", client.userID, "role", client.userRole, "total", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				slog.Info("ws client disconnected", "user_id", client.userID, "total", len(h.clients))
			}

		case reply := <-h.clientCount:
			reply <- len(h.clients)

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				slog.Error("ws marshal failed", "error", err)
				continue
			}
			for client := range h.clients {
				// Skip clients whose role doesn't meet the minimum requirement
				if msg.Role != "" && !hasRole(client.userRole, msg.Role) {
					continue
				}
				select {
				case client.send <- data:
				default:
					// Client send buffer full — evict slow client
					slog.Warn("ws client send buffer full, disconnecting", "user_id", client.userID)
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

var roleLevel = map[string]int{
	"viewer":   0,
	"user":     1,
	"operator": 1,
	"admin":    2,
}

func hasRole(userRole, required string) bool {
	if required == "" {
		return true
	}
	u, ok1 := roleLevel[strings.ToLower(userRole)]
	r, ok2 := roleLevel[strings.ToLower(required)]
	if !ok1 || !ok2 {
		return false
	}
	return u >= r
}

func (h *Hub) Broadcast(msg Message) {
	select {
	case h.broadcast <- msg:
	default:
		slog.Warn("ws broadcast channel full, dropping message", "type", msg.Type)
	}
}

func (h *Hub) ClientCount() int {
	reply := make(chan int, 1)
	h.clientCount <- reply
	return <-reply
}
