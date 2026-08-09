package notify

import (
	"context"
	"encoding/json"

	"github.com/renos-project/renos/core/internal/db"
)

// creator is the minimal surface Notifier needs; it lets tests inject a fake.
type creator interface {
	Create(ctx context.Context, n *db.InAppNotification) error
}

// Notifier produces in-app notifications. It is deliberately dependency-free
// beyond the repository so it can be injected anywhere in the API server.
type Notifier struct {
	repo creator
	// AdminID is the fallback recipient when an event has no associated user.
	AdminID string
}

func New(repo *db.InAppNotificationRepository, adminID string) *Notifier {
	return &Notifier{repo: repo, AdminID: adminID}
}

// resolveRecipient returns the effective recipient for an event: the explicit
// user, or the admin when absent, or "" when neither exists.
func resolveRecipient(userID, adminID string) string {
	if userID != "" {
		return userID
	}
	return adminID
}

// Notify sends a notification to a specific user. It never blocks: failures
// are swallowed to avoid breaking the event path.
func (n *Notifier) Notify(ctx context.Context, userID, typ, title, body, entityType, entityID string, metadata map[string]any) {
	recipient := resolveRecipient(userID, n.AdminID)
	if recipient == "" {
		return
	}
	var meta json.RawMessage
	if metadata == nil {
		meta = json.RawMessage(`{}`)
	} else {
		b, err := json.Marshal(metadata)
		if err != nil {
			meta = json.RawMessage(`{}`)
		} else {
			meta = b
		}
	}
	n.repo.Create(ctx, &db.InAppNotification{
		UserID:     recipient,
		Title:      title,
		Body:       body,
		Type:       typ,
		EntityType: entityType,
		EntityID:   entityID,
		Metadata:   meta,
	})
}
