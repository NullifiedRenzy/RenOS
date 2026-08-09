package developer

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWebhookSubscription_Fields(t *testing.T) {
	ws := &WebhookSubscription{
		ID:          "sub-1",
		UserID:      "user-1",
		URL:         "https://example.com/hook",
		Events:      []string{"device.online", "alert.fired"},
		Secret:      "whsec_test123",
		Active:      true,
		Headers:     map[string]string{"X-Custom": "value"},
		RetryPolicy: 3,
		CreatedAt:   time.Now(),
	}

	if ws.URL != "https://example.com/hook" { t.Error("expected URL") }
	if len(ws.Events) != 2 { t.Error("expected 2 events") }
	if ws.Headers["X-Custom"] != "value" { t.Error("expected custom header") }
	if ws.RetryPolicy != 3 { t.Error("expected retry policy 3") }
}

func TestWebhookDelivery_Fields(t *testing.T) {
	d := &WebhookDelivery{
		ID:             "del-1",
		SubscriptionID: "sub-1",
		Event:          "device.online",
		Payload:        []byte(`{"device_id":"d1"}`),
		StatusCode:     200,
		Response:       "OK",
		Success:        true,
		RetryCount:     0,
		DeliveredAt:    time.Now(),
	}

	if !d.Success { t.Error("expected success") }
	if d.StatusCode != 200 { t.Error("expected 200") }
}

func TestAPIKey_Fields(t *testing.T) {
	key := &APIKey{
		ID:        "key-1",
		UserID:    "user-1",
		Name:      "My Key",
		KeyHash:   "abc123hash",
		KeyPrefix: "renos_ab",
		Scopes:    []string{"read", "write", "deploy"},
		Active:    true,
		CreatedAt: time.Now(),
	}

	if key.Name != "My Key" { t.Error("expected name") }
	if len(key.Scopes) != 3 { t.Error("expected 3 scopes") }
	if !key.Active { t.Error("expected active") }
}

func TestKnownEvents(t *testing.T) {
	if len(KnownEvents) == 0 {
		t.Error("expected known events to be non-empty")
	}

	// Check some expected events exist
	found := map[string]bool{}
	for _, e := range KnownEvents {
		found[e] = true
	}

	required := []string{
		"device.online", "device.offline",
		"alert.fired", "alert.resolved",
		"deploy.started", "deploy.completed",
	}
	for _, r := range required {
		if !found[r] {
			t.Errorf("expected event %q in KnownEvents", r)
		}
	}
}

func TestGetAvailableEvents(t *testing.T) {
	events := GetAvailableEvents()
	if len(events) == 0 {
		t.Error("expected non-empty events")
	}

	// Should be sorted
	for i := 1; i < len(events); i++ {
		if events[i] < events[i-1] {
			t.Errorf("events not sorted: %s < %s", events[i], events[i-1])
		}
	}
}

func TestWebhookSubscriptionJSON(t *testing.T) {
	ws := &WebhookSubscription{
		ID:     "sub-1",
		URL:    "https://example.com",
		Events: []string{"device.online"},
		Active: true,
	}

	data, err := json.Marshal(ws)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded WebhookSubscription
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ID != ws.ID { t.Error("ID mismatch") }
	if decoded.URL != ws.URL { t.Error("URL mismatch") }
	if len(decoded.Events) != len(ws.Events) { t.Error("Events length mismatch") }
}

func TestAPIKeyJSON(t *testing.T) {
	key := &APIKey{
		ID:     "key-1",
		Name:   "Test",
		Scopes: []string{"read"},
		Active: true,
	}

	data, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded APIKey
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != "Test" { t.Error("name mismatch") }
	if !decoded.Active { t.Error("expected active") }
}

func TestNewRepositories(t *testing.T) {
	_ = NewWebhookSubscriptionRepository(nil)
	_ = NewWebhookDeliveryRepository(nil)
	_ = NewAPIKeyRepository(nil)
}
