package notify

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/renos-project/renos/core/internal/db"
)

type fakeCreator struct {
	created []*db.InAppNotification
}

func (f *fakeCreator) Create(_ context.Context, n *db.InAppNotification) error {
	f.created = append(f.created, n)
	return nil
}

func TestResolveRecipient(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		adminID string
		want    string
	}{
		{"explicit user wins", "user-1", "admin-1", "user-1"},
		{"falls back to admin", "", "admin-1", "admin-1"},
		{"empty when neither", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveRecipient(tt.userID, tt.adminID); got != tt.want {
				t.Errorf("resolveRecipient(%q, %q) = %q, want %q", tt.userID, tt.adminID, got, tt.want)
			}
		})
	}
}

func TestNotifierBuildsNotification(t *testing.T) {
	fake := &fakeCreator{}
	notifier := &Notifier{repo: fake, AdminID: "admin-1"}
	notifier.Notify(t.Context(), "user-9", "workflow", "Run finished", "All steps ok", "workflow", "wf-1", map[string]any{"steps": 3})

	if len(fake.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fake.created))
	}
	got := fake.created[0]
	if got.UserID != "user-9" {
		t.Errorf("UserID = %q, want user-9", got.UserID)
	}
	if got.Type != "workflow" {
		t.Errorf("Type = %q, want workflow", got.Type)
	}
	if got.EntityID != "wf-1" {
		t.Errorf("EntityID = %q, want wf-1", got.EntityID)
	}
	var meta map[string]int
	if err := json.Unmarshal(got.Metadata, &meta); err != nil {
		t.Fatal(err)
	}
	if meta["steps"] != 3 {
		t.Errorf("metadata steps = %v, want 3", meta["steps"])
	}
}

func TestNotifierFallsBackToAdmin(t *testing.T) {
	fake := &fakeCreator{}
	notifier := &Notifier{repo: fake, AdminID: "admin-1"}
	notifier.Notify(t.Context(), "", "system", "t", "b", "", "", nil)
	if len(fake.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(fake.created))
	}
	if fake.created[0].UserID != "admin-1" {
		t.Errorf("UserID = %q, want admin-1", fake.created[0].UserID)
	}
}

func TestNotifierSwallowsEmptyRecipient(t *testing.T) {
	fake := &fakeCreator{}
	notifier := &Notifier{repo: fake, AdminID: ""}
	notifier.Notify(t.Context(), "", "system", "t", "b", "", "", nil)
	if len(fake.created) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(fake.created))
	}
}
