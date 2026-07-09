package ws

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.broadcast == nil {
		t.Error("expected broadcast channel")
	}
	if hub.register == nil {
		t.Error("expected register channel")
	}
	if hub.unregister == nil {
		t.Error("expected unregister channel")
	}
}

func TestHubRegisterAndBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: "test-user",
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast(Message{
		Type:     "test.event",
		DeviceID: "device-1",
		Data:     map[string]string{"key": "value"},
	})

	select {
	case msg := <-client.send:
		var received Message
		if err := json.Unmarshal(msg, &received); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		if received.Type != "test.event" {
			t.Errorf("expected type 'test.event', got '%s'", received.Type)
		}
		if received.DeviceID != "device-1" {
			t.Errorf("expected device_id 'device-1', got '%s'", received.DeviceID)
		}
		data, ok := received.Data.(map[string]interface{})
		if !ok {
			t.Fatal("expected Data to be map[string]interface{}")
		}
		if data["key"] != "value" {
			t.Errorf("expected data.key 'value', got '%v'", data["key"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}
}

func TestHubUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		userID: "test-user",
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	if len(hub.clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(hub.clients))
	}

	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	if len(hub.clients) != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", len(hub.clients))
	}

	hub.Broadcast(Message{Type: "after.unregister", DeviceID: "d1"})
	time.Sleep(50 * time.Millisecond)

	select {
	case <-client.send:
		t.Error("unregistered client should not receive messages")
	default:
	}
}

func TestHubBroadcastMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	clients := make([]*Client, 5)
	for i := range clients {
		clients[i] = &Client{
			hub:    hub,
			send:   make(chan []byte, 256),
			userID: "test-user",
		}
		hub.register <- clients[i]
	}
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast(Message{Type: "multi.event", DeviceID: "d1"})

	for i, client := range clients {
		select {
		case <-client.send:
		case <-time.After(time.Second):
			t.Fatalf("client %d did not receive broadcast", i)
		}
	}
}

func TestHubBroadcastChannelFull(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	staticMsg := Message{Type: "static", DeviceID: "d1"}

	hub.Broadcast(staticMsg)
	hub.Broadcast(staticMsg)

	_ = staticMsg
}
