package notify

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/renos-project/renos/core/internal/db"
)

// creator is the minimal surface Notifier needs; it lets tests inject a fake.
type creator interface {
	Create(ctx context.Context, n *db.InAppNotification) error
}

type providerLister interface {
	ListEnabled(ctx context.Context) ([]*db.NotificationProvider, error)
}

type historyCreator interface {
	Create(ctx context.Context, h *db.NotificationHistoryEntry) error
}

type userFinder interface {
	FindByID(ctx context.Context, id string) (*db.User, error)
}

// Notifier produces in-app notifications. It is deliberately dependency-free
// beyond the repository so it can be injected anywhere in the API server.
type Notifier struct {
	repo      creator
	providers providerLister
	history   historyCreator
	users     userFinder
	// AdminID is the fallback recipient when an event has no associated user.
	AdminID     string
	emailSender emailSender
}

func New(repo *db.InAppNotificationRepository, adminID string) *Notifier {
	return &Notifier{repo: repo, AdminID: adminID, emailSender: sendSMTPEmail}
}

func (n *Notifier) WithEmail(providers providerLister, history historyCreator, users userFinder) *Notifier {
	n.providers = providers
	n.history = history
	n.users = users
	if n.emailSender == nil {
		n.emailSender = sendSMTPEmail
	}
	return n
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
	if n.providers != nil && n.history != nil && n.users != nil && n.emailSender != nil {
		go n.dispatchEmail(recipient, typ, title, body)
	}
}

func (n *Notifier) dispatchEmail(userID, typ, title, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	providers, err := n.providers.ListEnabled(ctx)
	if err != nil {
		slog.Warn("list notification providers for email", "error", err)
		return
	}
	for _, provider := range providers {
		if provider == nil || !provider.Enabled || !isEmailProvider(provider.ProviderType) {
			continue
		}
		cfg, err := parseSMTPConfig(provider.Config)
		if err != nil {
			n.recordEmailHistory(ctx, provider.ID, typ, title, body, err)
			continue
		}
		recipients := cfg.recipients()
		if len(recipients) == 0 {
			user, err := n.users.FindByID(ctx, userID)
			if err != nil {
				n.recordEmailHistory(ctx, provider.ID, typ, title, body, err)
				continue
			}
			if strings.TrimSpace(user.Email) != "" {
				recipients = []string{strings.TrimSpace(user.Email)}
			}
		}
		err = n.emailSender(ctx, cfg, recipients, title, body)
		n.recordEmailHistory(ctx, provider.ID, typ, title, body, err)
	}
}

func (n *Notifier) recordEmailHistory(ctx context.Context, providerID, typ, title, body string, sendErr error) {
	status := "sent"
	var errMsg string
	var sentAt *time.Time
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
	} else {
		now := time.Now().UTC()
		sentAt = &now
	}
	if err := n.history.Create(ctx, &db.NotificationHistoryEntry{
		ProviderID:   &providerID,
		EventType:    typ,
		Subject:      title,
		Body:         body,
		Status:       status,
		ErrorMessage: errMsg,
		SentAt:       sentAt,
	}); err != nil {
		slog.Warn("record email notification history", "error", err)
	}
}

func isEmailProvider(providerType string) bool {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case "email", "smtp":
		return true
	default:
		return false
	}
}
