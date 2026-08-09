package api

import (
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/renos-project/renos/core/internal/db"
)

// onboardingMarkerKey is the Redis / in-memory key that records whether the
// first-run wizard has been completed.
const onboardingMarkerKey = "renos:onboarding:complete"

// onboardingCompleteFallback is the in-memory fallback used when Redis is
// unavailable. It makes the completion endpoint idempotent per-process.
var onboardingCompleteFallback sync.Map

// onboardingStep describes a single wizard step in the onboarding status.
type onboardingStep struct {
	Done        bool   `json:"done"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// onboardingStatus is the full payload returned by the status/complete endpoints.
type onboardingStatus struct {
	OnboardingComplete     bool                      `json:"onboarding_complete"`
	AdminUser              bool                      `json:"admin_user"`
	AIProviderConfigured   bool                      `json:"ai_provider_configured"`
	SMTPConfigured         bool                      `json:"smtp_configured"`
	NotificationsConfigure bool                      `json:"notifications_configured"`
	DeviceEnrolled         bool                      `json:"device_enrolled"`
	PluginsInstalled       int                       `json:"plugins_installed"`
	BackupsEnabled         bool                      `json:"backups_enabled"`
	HasWorkflows           int                       `json:"has_workflows"`
	Steps                  map[string]onboardingStep `json:"steps"`
}

// onboardingStatusHandler returns the current first-run onboarding status.
func onboardingStatusHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := buildOnboardingStatus(deps, r)
		writeJSONResponse(w, http.StatusOK, status)
	}
}

// completeOnboardingHandler marks the first-run wizard as complete. It is
// idempotent and persists the marker in Redis when available, otherwise it
// falls back to an in-memory singleton. The full status is returned.
func completeOnboardingHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.RDB != nil {
			if err := deps.RDB.Set(r.Context(), onboardingMarkerKey, "true", 0).Err(); err == nil {
				writeJSONResponse(w, http.StatusOK, buildOnboardingStatus(deps, r))
				return
			}
		}
		onboardingCompleteFallback.Store(onboardingMarkerKey, "true")
		writeJSONResponse(w, http.StatusOK, buildOnboardingStatus(deps, r))
	}
}

func buildOnboardingStatus(deps *Dependencies, r *http.Request) onboardingStatus {
	deviceCount := countAll(deps.Devices, r)
	pluginCount := countInstalledPlugins(deps.InstalledPlugins, r)
	workflowCount := countWorkflows(deps.Workflows, r)

	notificationsConfigured := hasNotificationProviders(deps.NotifProviders, r)
	smtpConfigured := hasSMTPProvider(deps.NotifProviders, r)

	return onboardingStatus{
		OnboardingComplete:     isOnboardingComplete(deps.RDB, r),
		AdminUser:              true, // admin is seeded at startup
		AIProviderConfigured:   isAIProviderConfigured(deps),
		SMTPConfigured:         smtpConfigured,
		NotificationsConfigure: notificationsConfigured,
		DeviceEnrolled:         deviceCount > 0,
		PluginsInstalled:       pluginCount,
		BackupsEnabled:         os.Getenv("RENOS_BACKUP_ENABLED") == "true",
		HasWorkflows:           workflowCount,
		Steps: map[string]onboardingStep{
			"admin_account": {
				Done:        true,
				Title:       "Admin account",
				Description: "The administrator account is ready to use.",
			},
			"ai_provider": {
				Done:        isAIProviderConfigured(deps),
				Title:       "Connect an AI provider",
				Description: "Configure an OpenAI, Anthropic, Google or Ollama provider to enable RenAI.",
			},
			"notifications": {
				Done:        notificationsConfigured,
				Title:       "Configure notifications",
				Description: "Add an email or messaging provider so alerts reach you.",
			},
			"devices": {
				Done:        deviceCount > 0,
				Title:       "Enroll a device",
				Description: "Enroll your first server or edge device with the RenOS agent.",
			},
			"plugins": {
				Done:        pluginCount > 0,
				Title:       "Install plugins",
				Description: "Install plugins to extend RenOS with new capabilities.",
			},
			"backups": {
				Done:        os.Getenv("RENOS_BACKUP_ENABLED") == "true",
				Title:       "Enable backups",
				Description: "Turn on automated backups to protect your data.",
			},
			"workflows": {
				Done:        workflowCount > 0,
				Title:       "Create a workflow",
				Description: "Automate recurring operations with your first workflow.",
			},
		},
	}
}

// isOnboardingComplete reports whether the onboarding wizard has been
// completed, checking Redis first and falling back to the in-memory marker.
func isOnboardingComplete(rdb *redis.Client, r *http.Request) bool {
	if rdb != nil {
		val, err := rdb.Get(r.Context(), onboardingMarkerKey).Result()
		if err == nil {
			return strings.EqualFold(val, "true")
		}
	}
	if v, ok := onboardingCompleteFallback.Load(onboardingMarkerKey); ok {
		return v == "true"
	}
	return false
}

// isAIProviderConfigured reports whether an AI provider key, an Ollama URL or
// a custom endpoint is configured and RenAI was initialized with a provider.
func isAIProviderConfigured(deps *Dependencies) bool {
	if deps.Cfg == nil {
		return false
	}
	provider := deps.Cfg.AIProvider
	switch provider {
	case "openai":
		if deps.Cfg.AIOpenAIKey == "" {
			return false
		}
	case "anthropic":
		if deps.Cfg.AIAnthropicKey == "" {
			return false
		}
	case "google":
		if deps.Cfg.AIGoogleKey == "" {
			return false
		}
	case "ollama":
		if deps.Cfg.AIOllamaURL == "" {
			return false
		}
	default:
		if deps.Cfg.AIAIEndpoint == "" {
			return false
		}
	}
	return deps.RenAI != nil && deps.RenAI.GetProvider() != nil
}

func countAll(devices *db.DeviceRepository, r *http.Request) int {
	if devices == nil {
		return 0
	}
	list, err := devices.FindAll(r.Context())
	if err != nil {
		return 0
	}
	return len(list)
}

func countInstalledPlugins(installed *db.InstalledPluginRepository, r *http.Request) int {
	if installed == nil {
		return 0
	}
	list, err := installed.ListAll(r.Context())
	if err != nil {
		return 0
	}
	return len(list)
}

func countWorkflows(workflows *db.WorkflowRepository, r *http.Request) int {
	if workflows == nil {
		return 0
	}
	list, err := workflows.ListAll(r.Context())
	if err != nil {
		return 0
	}
	return len(list)
}

func hasNotificationProviders(providers *db.NotificationProviderRepository, r *http.Request) bool {
	if providers == nil {
		return false
	}
	list, err := providers.ListEnabled(r.Context())
	if err != nil {
		return false
	}
	return len(list) > 0
}

func hasSMTPProvider(providers *db.NotificationProviderRepository, r *http.Request) bool {
	if providers == nil {
		return false
	}
	list, err := providers.ListEnabled(r.Context())
	if err != nil {
		return false
	}
	for _, p := range list {
		t := strings.ToLower(p.ProviderType)
		if strings.Contains(t, "smtp") || strings.Contains(t, "email") || strings.Contains(t, "mail") {
			return true
		}
	}
	return false
}
