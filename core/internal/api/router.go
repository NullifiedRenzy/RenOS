package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"github.com/renos-project/renos/core/internal/ai"
	"github.com/renos-project/renos/core/internal/ca"
	"github.com/renos-project/renos/core/internal/config"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/developer"
	"github.com/renos-project/renos/core/internal/docs"
	"github.com/renos-project/renos/core/internal/notify"
	"github.com/renos-project/renos/core/internal/observability"
	"github.com/renos-project/renos/core/internal/plugin"
	"github.com/renos-project/renos/core/internal/policy"
	"github.com/renos-project/renos/core/internal/ws"
)

type Dependencies struct {
	Pool                     *pgxpool.Pool
	NC                       *nats.Conn
	RDB                      *redis.Client
	Cfg                      *config.Config
	CA                       *ca.CA
	Users                    *db.UserRepository
	Devices                  *db.DeviceRepository
	Commands                 *db.CommandRepository
	Containers               *db.ContainerRepository
	ContainerActions         *db.ContainerActionRepository
	DockerImages             *db.DockerImageRepository
	DockerVolumes            *db.DockerVolumeRepository
	DockerNetworks           *db.DockerNetworkRepository
	SystemdServices          *db.SystemdServiceRepository
	ServiceActions           *db.ServiceActionRepository
	NetworkInterfaces        *db.NetworkInterfaceRepository
	StorageDevices           *db.StorageDeviceRepository
	VirtualMachines          *db.VirtualMachineRepository
	UpdateStatus             *db.UpdateStatusRepository
	Plugins                  *db.PluginRepository
	AIConversations          *db.AIConversationRepository
	FileActions              *db.FileActionRepository
	FileTransfers            *db.FileTransferRepository
	FilePermissions          *db.FilePermissionRepository
	Processes                *db.ProcessRepository
	ComposeStacks            *db.ComposeStackRepository
	ContainerMetrics         *db.ContainerMetricsRepository
	Proxmox                  *db.ProxmoxRepository
	TerminalSessions         *db.TerminalSessionRepository
	TerminalRecordings       *db.TerminalRecordingRepository
	Audits                   *db.AuditRepository
	Groups                   *db.GroupRepository
	GroupActions             *db.GroupActionRepository
	Jobs                     *db.JobRepository
	Schedules                *db.ScheduleRepository
	ScheduleRuns             *db.ScheduleRunRepository
	Policies                 *db.PolicyRepository
	PolicyEnforcer           *policy.Enforcer
	SystemLogs               *db.SystemLogRepository
	Executions               *db.ExecutionRepository
	InstalledPlugins         *db.InstalledPluginRepository
	PluginLoader             *plugin.Loader
	RenAI                    *ai.RenAI
	DeviceTags               *db.DeviceTagRepository
	DeviceInventory          *db.DeviceInventoryRepository
	Compliance               *db.ComplianceRepository
	BulkOperations           *db.BulkOperationRepository
	Workflows                *db.WorkflowRepository
	WorkflowSteps            *db.WorkflowStepRepository
	WorkflowRuns             *db.WorkflowRunRepository
	WorkflowStepRuns         *db.WorkflowStepRunRepository
	WorkflowTemplates        *db.WorkflowTemplateRepository
	MonitoringMetrics        *db.MonitoringMetricRepository
	AlertRules               *db.AlertRuleRepository
	AlertInstances           *db.AlertInstanceRepository
	MonitoringDashboards     *db.MonitoringDashboardRepository
	MonitoringWidgets        *db.MonitoringWidgetRepository
	MonitoringLogs           *db.MonitoringLogRepository
	WidgetData               widgetDataProvider
	VaultSecrets             *db.VaultSecretRepository
	VaultVersions            *db.VaultSecretVersionRepository
	VaultKeys                *db.VaultKeyRepository
	VaultPolicies            *db.VaultPolicyRepository
	VaultGrants              *db.VaultGrantRepository
	VaultAudit               *db.VaultAuditRepository
	NotifProviders           *db.NotificationProviderRepository
	NotifTemplates           *db.NotificationTemplateRepository
	NotifRules               *db.NotificationRuleRepository
	NotifHistory             *db.NotificationHistoryRepository
	InAppNotifications       *db.InAppNotificationRepository
	Notifier                 *notify.Notifier
	MarketplacePlugins       *db.MarketplacePluginRepository
	Search                   *db.SearchRepository
	MarketplaceInstallations *db.MarketplaceInstallationRepository
	MarketplaceHistory       *db.MarketplaceInstallHistoryRepository
	UpdateHistory            *db.UpdateHistoryRepository
	UpdateBackups            *db.UpdateBackupRepository
	ClusterNodes             *db.ClusterNodeRepository
	ClusterElections         *db.ClusterElectionRepository
	ClusterTasks             *db.ClusterTaskRepository
	ClusterState             *db.ClusterStateRepository
	ClusterMetrics           *db.ClusterMetricsRepository
	PushRegistrations        *db.PushRegistrationRepository
	MobileSessions           *db.MobileSessionRepository
	OfflineSync              *db.OfflineSyncRepository
	DeviceApprovals          *db.DeviceApprovalRepository
	MobileSummaries          *db.MobileSummaryRepository
	WebhookSubs              *developer.WebhookSubscriptionRepository
	WebhookDeliveries        *developer.WebhookDeliveryRepository
	DeveloperKeys            *developer.APIKeyRepository
	IdentityProviders        *db.IdentityProviderRepository
	OAuth2Apps               *db.OAuth2AppRepository
	APITokens                *db.APITokenRepository
	IAMRoles                 *db.IAMRoleRepository
	IAMRoleAssignments       *db.IAMRoleAssignmentRepository
	ApprovalWorkflows        *db.ApprovalWorkflowRepository
	ApprovalRequests         *db.ApprovalRequestRepository
	ApprovalDecisions        *db.ApprovalDecisionRepository
	IAMAudit                 *db.IAMAuditRepository
	AdminID                  string
	Hub                      *ws.Hub
	CrashDir                 string
	BackupsDir               string
}

// Role constants used across route registration.
const (
	roleAdmin    = "admin"
	roleOperator = "operator"
	roleViewer   = "viewer"
)

func NewRouter(deps *Dependencies) *chi.Mux {
	r := chi.NewRouter()

	// Apply config values to rate limiters and body size
	InitRateLimiters(deps.Cfg)
	maxBodySize = int64(deps.Cfg.MaxBodySize)

	// Apply config to WS connection tracker
	ws.SetMaxConnsPerUser(deps.Cfg.MaxWSConnsPerUser)

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(securityHeadersMW)
	r.Use(corsMW)
	r.Use(bodyLimitMW)
	r.Use(compressionHintMiddleware)
	r.Use(requestIDMW)
	r.Use(requestLogger)
	r.Use(recovery)
	r.Use(connTracker.middleware)
	r.Use(slowLogger.middleware)
	r.Use(timingHeaderMW)
	r.Use(methodOverrideMW)
	r.Use(csrfProtectionMW)
	r.Use(observability.TracingMiddleware)
	r.Use(observability.LogCorrelationMiddleware)

	// ── Observability endpoints (auth-free) ──
	if deps.Cfg.PprofEnabled {
		r.Group(func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(roleRequired(deps.Audits, roleAdmin))
			r.HandleFunc(deps.Cfg.PprofPath+"/*", pprofHandler())
		})
	}
	r.Get("/livez", observability.LivezHandler())
	r.Get("/startupz", observability.StartupzHandler(deps.Pool, deps.RDB))

	// ── API Documentation ──
	r.Get("/api/openapi.json", docs.OpenAPIJSONHandler())
	r.Get("/api/openapi.yaml", docs.OpenAPIYAMLHandler())
	r.Get("/api/docs", docs.DocsHandler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(observability.MetricsMiddleware)
		r.Get("/healthz", healthHandler(deps.Pool, deps.NC, deps.RDB))
		r.Get("/readyz", readyHandler(deps.Pool, deps.NC, deps.RDB))
		r.Get("/version", versionHandler())
		r.Get("/stats", serverStatsHandler())
		r.Get("/stats/memory", memoryStatsHandler())
		r.Get("/stats/gc", gcStatsHandler())
		r.Get("/stats/connections", connectionStatsHandler())
		r.Get("/stats/slow-requests", slowRequestStatsHandler())
		r.Get("/stats/env", envInfoHandler())
		r.Get("/debug/request", debugRequestHandler())
		r.Get("/debug/config", configSummaryHandler(deps))

		// Health dashboard — weighted health summary (viewer+)
		r.Group(func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/health/dashboard", healthDashboardHandler(deps))
		})

		// Onboarding — first-run wizard status and completion
		r.Group(func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/onboarding/status", onboardingStatusHandler(deps))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/onboarding/complete", completeOnboardingHandler(deps))
		})

		// Demo mode — seed demo dataset (admin only)
		r.Group(func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/demo/status", demoStatusHandler(deps))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/demo/seed", seedDemoHandler(deps))
		})

		// Global search — aggregate results across all entities (viewer+)
		r.Group(func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/search", searchHandler(deps.Search))
		})

		// Rate limit login specifically
		r.With(rateLimitMW(loginLimiter)).Post("/auth/login", loginHandler(deps.Users, deps.Cfg.JWTSecret, deps.Audits))
		r.Group(func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.Post("/auth/mfa/setup", setupMFAHandler(deps.Users, deps.Audits))
			r.Post("/auth/mfa/enable", enableMFAHandler(deps.Users, deps.Audits))
			r.Post("/auth/mfa/disable", disableMFAHandler(deps.Users, deps.Audits))
		})

		// WebSocket — JWT auth via query param, rate limited
		r.With(rateLimitMW(wsConnectLimiter)).Get("/ws", ws.ServeWS(deps.Hub, deps.Cfg.JWTSecret))
		r.With(rateLimitMW(wsConnectLimiter)).Get("/devices/{id}/terminal/ws", terminalWebSocketHandler(deps.TerminalSessions, deps.TerminalRecordings, deps.Devices, deps.Audits, deps.NC, deps.Hub, deps.Cfg.JWTSecret))

		// Agent-facing — rate limited, uses device_secret auth
		r.Group(func(r chi.Router) {
			r.Use(rateLimitMW(apiLimiter))
			r.Post("/devices/enroll", enrollHandler(deps.CA, deps.Devices, deps.Audits, deps.Cfg, deps.AdminID))
			r.Post("/devices/connect", connectHandler(deps.Devices, deps.Audits, deps.Hub))
			r.Post("/devices/heartbeat", heartbeatHandler(deps.Devices, deps.Audits, deps.Hub))
			r.Post("/commands/pending", pendingCommandsHandler(deps.Commands, deps.Devices))
			r.Post("/commands/{cmdId}/result", submitResultHandler(deps.Commands, deps.Devices, deps.GroupActions, deps.Audits, deps.Hub))
			r.Post("/devices/{id}/containers", syncContainersHandler(deps.Containers, deps.Devices))
			r.Post("/devices/{id}/docker-images", syncDockerImagesHandler(deps.DockerImages, deps.Devices))
			r.Post("/devices/{id}/docker-volumes", syncDockerVolumesHandler(deps.DockerVolumes, deps.Devices))
			r.Post("/devices/{id}/docker-networks", syncDockerNetworksHandler(deps.DockerNetworks, deps.Devices))
			r.Post("/devices/{id}/systemd-services", syncSystemdServicesHandler(deps.SystemdServices, deps.Devices))
			r.Post("/devices/{id}/network-interfaces", syncNetworkInterfacesHandler(deps.NetworkInterfaces, deps.Devices))
			r.Post("/devices/{id}/storage-devices", syncStorageDevicesHandler(deps.StorageDevices, deps.Devices))
			r.Post("/devices/{id}/virtual-machines", syncVirtualMachinesHandler(deps.VirtualMachines, deps.Devices))
			r.Post("/devices/{id}/updates", syncUpdatesHandler(deps.UpdateStatus, deps.Devices))
			r.Post("/devices/{id}/plugins", syncPluginsHandler(deps.Plugins, deps.Devices))
			r.Post("/container-actions/pending", pendingContainerActionsHandler(deps.ContainerActions, deps.Devices))
			r.Post("/container-actions/{actionId}/result", submitContainerActionResultHandler(deps.ContainerActions, deps.Devices, deps.Audits, deps.Hub))
			r.Post("/service-actions/pending", pendingServiceActionsHandler(deps.ServiceActions, deps.Devices))
			r.Post("/service-actions/{actionId}/result", submitServiceActionResultHandler(deps.ServiceActions, deps.Devices, deps.Audits, deps.Hub))
			r.Post("/file-actions/pending", pendingFileActionsHandler(deps.FileActions, deps.Devices))
			r.Post("/file-actions/{actionId}/result", submitFileActionResultHandler(deps.FileActions, deps.Devices, deps.Audits, deps.Hub))
			r.Post("/file-transfers/pending", getTransferForAgentHandler(deps.FileTransfers, deps.Devices))
			r.Post("/file-transfers/{transferId}/chunks/{chunkNum}", uploadChunkHandler(deps.FileTransfers, deps.Audits, deps.Hub))
			r.Post("/file-transfers/{transferId}/assemble", assembleTransferHandler(deps.FileTransfers, deps.Audits, deps.Hub))
			r.Post("/devices/{id}/compose-stacks", syncComposeStacksHandler(deps.ComposeStacks, deps.Devices))
			r.Post("/devices/{id}/container-metrics", syncContainerMetricsHandler(deps.ContainerMetrics, deps.Devices))
			r.Post("/devices/{id}/proxmox", syncProxmoxHandler(deps.Proxmox, deps.Devices))
		})

		// User-facing — requires JWT auth, rate limited
		r.Route("/devices", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listDevicesHandler(deps.Devices, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getDeviceHandler(deps.Devices, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/enroll-token", generateEnrollTokenHandler(deps.Cfg, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Route("/{id}/commands", func(r chi.Router) {
				r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", requestCommandHandler(deps.Commands, deps.Devices, deps.Audits, deps.Hub, deps.PolicyEnforcer))
				r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listCommandsHandler(deps.Commands))
				r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{cmdId}", getCommandHandler(deps.Commands))
			})
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/containers", listContainersHandler(deps.Containers))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/docker-images", listDockerImagesHandler(deps.DockerImages))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/docker-volumes", listDockerVolumesHandler(deps.DockerVolumes))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/docker-networks", listDockerNetworksHandler(deps.DockerNetworks))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/systemd-services", listSystemdServicesHandler(deps.SystemdServices))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/network-interfaces", listNetworkInterfacesHandler(deps.NetworkInterfaces))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/storage-devices", listStorageDevicesHandler(deps.StorageDevices))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/virtual-machines", listVirtualMachinesHandler(deps.VirtualMachines))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/vms/{domainName}/{action}", requestVMActionHandler(deps.VirtualMachines, deps.Devices, deps.Audits, deps.Hub, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/updates", listUpdatesHandler(deps.UpdateStatus))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/updates/{action}", requestUpdateActionHandler(deps.UpdateStatus, deps.Devices, deps.Audits, deps.Hub, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/plugins", listPluginsHandler(deps.Plugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/plugins/{name}/{action}", requestPluginActionHandler(deps.Plugins, deps.Devices, deps.Audits, deps.Hub, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/executions", createExecutionHandler(deps.Executions, deps.Devices, deps.Audits, deps.NC, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/executions", listExecutionsHandler(deps.Executions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/executions/{execId}", getExecutionHandler(deps.Executions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/executions/{execId}/output", getExecutionOutputHandler(deps.Executions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/executions/{execId}/cancel", cancelExecutionHandler(deps.Executions, deps.NC, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/services/{unitName}/{action}", requestServiceActionHandler(deps.ServiceActions, deps.Devices, deps.Audits, deps.Hub, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/containers/{containerId}/{action}", requestContainerActionHandler(deps.ContainerActions, deps.Devices, deps.Audits, deps.Hub, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/files/{actionType}", requestFileActionHandler(deps.FileActions, deps.Devices, deps.Audits, deps.Hub, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/files", listFileActionsHandler(deps.FileActions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/file-actions/{actionId}", getFileActionResultHandler(deps.FileActions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/file-transfers", createFileTransferHandler(deps.FileTransfers, deps.Devices, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/file-transfers", listTransfersHandler(deps.FileTransfers))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/file-transfers/{transferId}", getTransferHandler(deps.FileTransfers))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/file-transfers/{transferId}/chunks/{chunkNum}", uploadChunkHandler(deps.FileTransfers, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/file-transfers/{transferId}/chunks/{chunkNum}", downloadChunkHandler(deps.FileTransfers))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/file-transfers/{transferId}/assemble", assembleTransferHandler(deps.FileTransfers, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/file-transfers/{transferId}/cancel", cancelTransferHandler(deps.FileTransfers, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/{id}/file-permissions", upsertFilePermissionHandler(deps.FilePermissions, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/file-permissions", listFilePermissionsHandler(deps.FilePermissions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/file-permissions/{userId}", getFilePermissionForUserHandler(deps.FilePermissions))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/{id}/file-permissions/{userId}", deleteFilePermissionHandler(deps.FilePermissions, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/compose-stacks", listComposeStacksHandler(deps.ComposeStacks))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/container-metrics", getLatestContainerMetricsHandler(deps.ContainerMetrics))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/container-metrics/{containerId}", getContainerMetricHistoryHandler(deps.ContainerMetrics))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/processes", listProcessesHandler(deps.Processes))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/processes/search", searchProcessesHandler(deps.Processes))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/processes/history", getProcessHistoryHandler(deps.Processes))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/processes/aggregates", getProcessAggregatesHandler(deps.Processes))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/processes/actions", createProcessActionHandler(deps.Processes, deps.Devices, deps.Audits, deps.NC, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/processes/actions", listProcessActionsHandler(deps.Processes))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/connections", listProxmoxConnectionsHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/proxmox/connections", createProxmoxConnectionHandler(deps.Proxmox, deps.Devices, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{id}/proxmox/connections/{connectionId}", deleteProxmoxConnectionHandler(deps.Proxmox, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/nodes", listProxmoxNodesHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/vms", listProxmoxVMsHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/lxcs", listProxmoxLXCsHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/storage", listProxmoxStorageHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/networks", listProxmoxNetworksHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/snapshots", listProxmoxSnapshotsHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/proxmox/tasks", listProxmoxTasksHandler(deps.Proxmox))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/proxmox/power", proxmoxPowerActionHandler(deps.Proxmox, deps.Devices, deps.Audits, deps.NC, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/terminal/sessions", listTerminalSessionsHandler(deps.TerminalSessions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/terminal/sessions/{sessionId}", getTerminalSessionHandler(deps.TerminalSessions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/terminal/sessions/{sessionId}/recording", getTerminalRecordingHandler(deps.TerminalRecordings))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/terminal/sessions", listTerminalSessionsByUserHandler(deps.TerminalSessions))
		})

		r.Route("/groups", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listGroupsHandler(deps.Groups, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", createGroupHandler(deps.Groups, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getGroupHandler(deps.Groups, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/{id}", updateGroupHandler(deps.Groups, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{id}", deleteGroupHandler(deps.Groups, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/devices", addDeviceToGroupHandler(deps.Groups, deps.Devices, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{id}/devices/{deviceId}", removeDeviceFromGroupHandler(deps.Groups, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/commands", runGroupCommandHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/containers/{containerId}/{action}", groupContainerActionHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/services/{unitName}/{action}", groupServiceActionHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/updates/{action}", groupUpdateActionHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/actions", listGroupActionsHandler(deps.GroupActions))
		})

		r.Route("/group-actions", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{actionId}", getGroupActionHandler(deps.GroupActions))
		})

		r.Route("/multi-device", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/commands", multiDeviceCommandHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/containers/{containerId}/{action}", groupContainerActionHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/services/{unitName}/{action}", groupServiceActionHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/updates/{action}", groupUpdateActionHandler(deps.Groups, deps.GroupActions, deps.Jobs, deps.Audits, deps.PolicyEnforcer))
		})

		r.Route("/jobs", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listJobsHandler(deps.Jobs, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/stats", jobStatsHandler(deps.Jobs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getJobHandler(deps.Jobs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/cancel", cancelJobHandler(deps.Jobs, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/retry", retryJobHandler(deps.Jobs, deps.GroupActions, deps.Groups, deps.Audits))
		})

		r.Route("/schedules", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listSchedulesHandler(deps.Schedules))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", createScheduleHandler(deps.Schedules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getScheduleHandler(deps.Schedules))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/{id}", updateScheduleHandler(deps.Schedules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{id}", deleteScheduleHandler(deps.Schedules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/enable", enableScheduleHandler(deps.Schedules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/disable", disableScheduleHandler(deps.Schedules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/runs", listScheduleRunsHandler(deps.ScheduleRuns))
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listUsersHandler(deps.Users))
			r.With(roleRequired(deps.Audits, roleAdmin)).Put("/{id}/role", updateUserRoleHandler(deps.Users, deps.Audits))
		})

		r.Route("/policies", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listPoliciesHandler(deps.Policies))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/", createPolicyHandler(deps.Policies, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getPolicyHandler(deps.Policies))
			r.With(roleRequired(deps.Audits, roleAdmin)).Put("/{id}", updatePolicyHandler(deps.Policies, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/{id}", deletePolicyHandler(deps.Policies, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/{id}/enable", enablePolicyHandler(deps.Policies, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/{id}/disable", disablePolicyHandler(deps.Policies, deps.Audits))
		})

		r.Route("/ai", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/chat", chatHandler(deps.RenAI, deps.Audits, deps.Notifier))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/stream-chat", streamChatHandler(deps.RenAI, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/conversations", listConversationsHandler(deps.RenAI))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/conversations/{id}", getConversationHandler(deps.RenAI))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/conversations/{id}", deleteConversationHandler(deps.RenAI))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/conversations/search", searchConversationsHandler(deps.RenAI))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/settings", aiSettingsHandler())
		})

		// --- Observability ---

		if deps.Cfg.DeepHealthEnabled {
			r.Get(deps.Cfg.DeepHealthPath, observability.DeepHealthHandler(deps.Pool, deps.NC, deps.RDB))
		}

		if deps.Cfg.SLOEnabled {
			r.Get(deps.Cfg.SLOPath, observability.SLOHandler())
		}

		r.Route("/metrics", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", metricsHandler(deps.Jobs, deps.Policies, deps.ScheduleRuns))
		})

		r.Route("/logs", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listLogsHandler(deps.SystemLogs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", createLogHandler(deps.SystemLogs))
		})

		// --- Automation Engine ---
		r.Route("/workflows", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listWorkflowsHandler(deps.Workflows))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/runs", listAllWorkflowRunsHandler(deps.WorkflowRuns))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/templates", listWorkflowTemplatesHandler(deps.WorkflowTemplates))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/templates/{id}", getWorkflowTemplateHandler(deps.WorkflowTemplates))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getWorkflowHandler(deps.Workflows))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", createWorkflowHandler(deps.Workflows, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/{id}", updateWorkflowHandler(deps.Workflows, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{id}", deleteWorkflowHandler(deps.Workflows, deps.WorkflowSteps, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/toggle", toggleWorkflowHandler(deps.Workflows, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/execute", executeWorkflowHandler(deps.Workflows, deps.WorkflowRuns, deps.WorkflowSteps, deps.WorkflowStepRuns, deps.Audits, deps.Notifier))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/steps", listWorkflowStepsHandler(deps.WorkflowSteps))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/steps", createWorkflowStepHandler(deps.WorkflowSteps, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{id}/steps", deleteWorkflowStepsHandler(deps.WorkflowSteps, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/runs", listWorkflowRunsHandler(deps.WorkflowRuns))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}/runs/{runId}", getWorkflowRunHandler(deps.WorkflowRuns, deps.WorkflowStepRuns))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/templates/{id}/create", createWorkflowFromTemplateHandler(deps.Workflows, deps.WorkflowSteps, deps.WorkflowTemplates, deps.Audits))
		})

		// --- Fleet Management ---
		r.Route("/fleet", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/dashboard", fleetDashboardHandler(deps.Devices, deps.DeviceInventory, deps.Compliance, deps.Groups, deps.DeviceTags))
		})

		r.Route("/tags", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/keys", listTagKeysHandler(deps.DeviceTags))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/search", findDevicesByTagHandler(deps.DeviceTags))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/devices/{deviceId}", listDeviceTagsHandler(deps.DeviceTags))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/devices/{deviceId}", setDeviceTagHandler(deps.DeviceTags, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/devices/{deviceId}/{tagKey}", removeDeviceTagHandler(deps.DeviceTags, deps.Audits))
		})

		r.Route("/inventory", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listDeviceInventoryHandler(deps.DeviceInventory))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/devices/{deviceId}", getDeviceInventoryHandler(deps.DeviceInventory))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/devices/{deviceId}", upsertDeviceInventoryHandler(deps.DeviceInventory, deps.Audits))
		})

		r.Route("/compliance", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/checks", listComplianceChecksHandler(deps.Compliance))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/checks", createComplianceCheckHandler(deps.Compliance, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/results", listComplianceResultsHandler(deps.Compliance))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/results", upsertComplianceResultHandler(deps.Compliance, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/summary", complianceSummaryHandler(deps.Compliance))
		})

		r.Route("/bulk", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", createBulkOperationHandler(deps.BulkOperations, deps.Audits, deps.Hub))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listBulkOperationsHandler(deps.BulkOperations))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getBulkOperationHandler(deps.BulkOperations))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/cancel", cancelBulkOperationHandler(deps.BulkOperations, deps.Audits))
		})

		// --- Notifications ---
		r.Route("/notifications", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/providers", listNotificationProvidersHandler(deps.NotifProviders))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/providers", createNotificationProviderHandler(deps.NotifProviders, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/providers/{id}/toggle", toggleNotificationProviderHandler(deps.NotifProviders, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/providers/{id}", deleteNotificationProviderHandler(deps.NotifProviders, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/templates", listNotificationTemplatesHandler(deps.NotifTemplates))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/templates", createNotificationTemplateHandler(deps.NotifTemplates, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/templates/{id}", deleteNotificationTemplateHandler(deps.NotifTemplates, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/rules", listNotificationRulesHandler(deps.NotifRules))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/rules", createNotificationRuleHandler(deps.NotifRules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/rules/{id}", deleteNotificationRuleHandler(deps.NotifRules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/history", listNotificationHistoryHandler(deps.NotifHistory))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/stats", notificationStatsHandler(deps.NotifHistory))
			// In-app notification center
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/inapp", listInAppNotificationsHandler(deps.InAppNotifications))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/inapp/unread_count", unreadCountHandler(deps.InAppNotifications))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Post("/inapp/{id}/read", markNotificationReadHandler(deps.InAppNotifications))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Post("/inapp/read-all", markAllNotificationsReadHandler(deps.InAppNotifications))
		})

		// --- RenVault ---
		r.Route("/vault", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/secrets", listVaultSecretsHandler(deps.VaultSecrets))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/secrets", createVaultSecretHandler(deps.VaultSecrets, deps.VaultVersions, deps.VaultAudit, deps.Audits, deps.Cfg.EncryptionKey))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/secrets/{id}", getVaultSecretHandler(deps.VaultSecrets))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/secrets/{id}", updateVaultSecretHandler(deps.VaultSecrets, deps.VaultVersions, deps.VaultAudit, deps.Audits, deps.Cfg.EncryptionKey))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/secrets/{id}", deleteVaultSecretHandler(deps.VaultSecrets, deps.VaultAudit, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/secrets/{id}/versions", listVaultSecretVersionsHandler(deps.VaultVersions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/secrets/{id}/versions/{version}", getVaultSecretVersionHandler(deps.VaultVersions))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/keys", listVaultKeysHandler(deps.VaultKeys))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/keys/{id}/toggle", toggleVaultKeyHandler(deps.VaultKeys, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/policies", listVaultPoliciesHandler(deps.VaultPolicies))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/audit", vaultAuditLogHandler(deps.VaultAudit))
		})

		// --- Monitoring Platform ---
		r.Route("/monitoring", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.Route("/metrics", func(r chi.Router) {
				r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", recordMetricHandler(deps.MonitoringMetrics, deps.Audits))
				r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", queryMetricsHandler(deps.MonitoringMetrics))
				r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/device/{deviceId}", latestMetricsByDeviceHandler(deps.MonitoringMetrics))
			})
		})

		r.Route("/alerts", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/rules", listAlertRulesHandler(deps.AlertRules))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/rules", createAlertRuleHandler(deps.AlertRules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/rules/{id}", updateAlertRuleHandler(deps.AlertRules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/rules/{id}", deleteAlertRuleHandler(deps.AlertRules, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/active", listActiveAlertsHandler(deps.AlertInstances))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/recent", listRecentAlertsHandler(deps.AlertInstances))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/summary", alertSummaryHandler(deps.AlertInstances))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/acknowledge", acknowledgeAlertHandler(deps.AlertInstances, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/resolve", resolveAlertHandler(deps.AlertInstances, deps.Audits))
		})

		r.Route("/dashboards", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listMonitoringDashboardsHandler(deps.MonitoringDashboards))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{id}", getMonitoringDashboardHandler(deps.MonitoringDashboards, deps.MonitoringWidgets, deps.WidgetData))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/", createMonitoringDashboardHandler(deps.MonitoringDashboards, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Put("/{id}", updateMonitoringDashboardHandler(deps.MonitoringDashboards, deps.MonitoringWidgets, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{id}/widgets", createMonitoringWidgetHandler(deps.MonitoringWidgets, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/widgets/{id}", deleteMonitoringWidgetHandler(deps.MonitoringWidgets, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{id}", deleteMonitoringDashboardHandler(deps.MonitoringDashboards, deps.MonitoringWidgets, deps.Audits))
		})

		r.Route("/monitoring/logs", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listMonitoringLogsHandler(deps.MonitoringLogs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/stats", monitoringLogStatsHandler(deps.MonitoringLogs))
		})

		// --- Plugin Platform (Admin) ---
		r.Route("/plugin-platform", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listInstalledPluginsHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{name}", getInstalledPluginHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/{name}/enable", enableInstalledPluginHandler(deps.PluginLoader))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/{name}/disable", disableInstalledPluginHandler(deps.PluginLoader))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/{name}", removeInstalledPluginHandler(deps.PluginLoader))
			r.With(roleRequired(deps.Audits, roleAdmin)).Put("/{name}/config", updateInstalledPluginConfigHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{name}/routes", listInstalledPluginRoutesHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{name}/pages", listInstalledPluginPagesHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{name}/workers", listInstalledPluginWorkersHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{name}/schedules", listInstalledPluginSchedulesHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/sidebar/all", listInstalledPluginSidebarHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/routes/all", listAllEnabledPluginRoutesHandler(deps.InstalledPlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/pages/all", listAllEnabledPluginPagesHandler(deps.InstalledPlugins))
		})

		// Dynamic plugin API routes (catch-all for plugin handlers)
		r.Route("/plugins", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.HandleFunc("/*", DynamicPluginRoute(deps.InstalledPlugins, deps.PluginLoader))
		})

		// --- Deployment Platform ---
		r.Route("/deploy", func(r chi.Router) {
			r.Use(rateLimitMW(apiLimiter))
			r.Get("/preflight", preflightCheckHandler())
			r.Get("/info", systemInfoHandler())
			r.Group(func(r chi.Router) {
				r.Use(authRequired(deps.Cfg.JWTSecret))
				r.Use(roleRequired(deps.Audits, roleAdmin))
				r.Get("/status", installationStatusHandler())
				r.Get("/stream", installationStreamHandler())
				r.Post("/start", startInstallationHandler())
				r.Post("/rollback", rollbackHandler())
			})
		})

		// --- Updates ---
		r.Route("/updates", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", getVersionInfoHandler())
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Get("/check", checkForUpdatesHandler(deps.UpdateHistory, deps.Notifier))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/plan", planUpdateHandler(deps.UpdateHistory))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/start", startUpdateHandler(deps.UpdateHistory, deps.UpdateBackups, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/rollback", rollbackUpdateHandler(deps.UpdateHistory, deps.UpdateBackups, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/history", listUpdateHistoryHandler(deps.UpdateHistory))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/backups", listUpdateBackupsHandler(deps.UpdateBackups))
			r.With(roleRequired(deps.Audits, roleAdmin)).Put("/channel", setUpdateChannelHandler())
		})

		// --- Cluster ---
		r.Route("/cluster", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/dashboard", clusterDashboardHandler(deps.ClusterNodes, deps.ClusterMetrics, deps.ClusterTasks, deps.ClusterElections))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/nodes", listClusterNodesHandler(deps.ClusterNodes))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/nodes", registerClusterNodeHandler(deps.ClusterNodes))
			r.With(roleRequired(deps.Audits, roleAdmin)).Put("/nodes/status", updateClusterNodeStatusHandler(deps.ClusterNodes))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/tasks", listClusterTasksHandler(deps.ClusterTasks))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/tasks", createClusterTaskHandler(deps.ClusterTasks))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/tasks/rebalance", rebalanceClusterTasksHandler(deps.ClusterTasks))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/state", clusterStateHandler(deps.ClusterState))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/state", setClusterStateHandler(deps.ClusterState))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/elections", clusterElectionsHandler(deps.ClusterElections))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/metrics", recordClusterMetricsHandler(deps.ClusterMetrics))
		})

		// --- Mobile Platform ---
		r.Route("/mobile", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.Post("/push/register", registerPushHandler(deps.PushRegistrations))
			r.Get("/push", listPushRegistrationsHandler(deps.PushRegistrations))
			r.Delete("/push", deletePushRegistrationHandler(deps.PushRegistrations))
			r.Get("/dashboard", mobileDashboardHandler(deps.Devices, deps.AlertInstances, deps.ClusterNodes))
			r.Get("/devices", mobileDeviceListHandler(deps.Devices))
			r.Get("/alerts/summary", mobileAlertsSummaryHandler(deps.AlertInstances))
			r.HandleFunc("/sync", mobileOfflineSyncHandler(deps.OfflineSync))
			r.Get("/approvals", listDeviceApprovalsHandler(deps.DeviceApprovals))
			r.Post("/approvals", requestDeviceApprovalHandler(deps.DeviceApprovals))
			r.Post("/approvals/approve", approveDeviceHandler(deps.DeviceApprovals))
			r.Post("/approvals/reject", rejectDeviceHandler(deps.DeviceApprovals))
			r.Get("/sessions", mobileSessionsHandler(deps.MobileSessions))
			r.Delete("/sessions", deleteMobileSessionHandler(deps.MobileSessions))
			r.Get("/summary", mobileSummaryHandler(deps.MobileSummaries, deps.Devices, deps.AlertInstances, deps.ClusterNodes, deps.ClusterMetrics))
		})

		// --- Marketplace ---
		r.Route("/marketplace", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/", listMarketplacePluginsHandler(deps.MarketplacePlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/installed", listMarketplaceInstallationsHandler(deps.MarketplaceInstallations))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/{name}", getMarketplacePluginHandler(deps.MarketplacePlugins))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{name}/install", installMarketplacePluginHandler(deps.MarketplacePlugins, deps.MarketplaceInstallations, deps.MarketplaceHistory, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{name}/upgrade", upgradeMarketplacePluginHandler(deps.MarketplacePlugins, deps.MarketplaceInstallations, deps.MarketplaceHistory, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/{name}/rollback", rollbackMarketplacePluginHandler(deps.MarketplaceInstallations, deps.MarketplaceHistory, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/{name}", uninstallMarketplacePluginHandler(deps.MarketplaceInstallations, deps.Audits))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/installations/{id}/history", marketplaceInstallHistoryHandler(deps.MarketplaceHistory))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/sync", syncMarketplaceHandler(deps.MarketplacePlugins, deps.Audits))
		})

		// --- Identity & Access Management ---
		r.Route("/iam", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			// Identity Providers
			r.With(roleRequired(deps.Audits, roleAdmin)).Get("/providers", listIdentityProvidersHandler(deps.IdentityProviders))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/providers", createIdentityProviderHandler(deps.IdentityProviders, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/providers/{id}", deleteIdentityProviderHandler(deps.IdentityProviders, deps.IAMAudit))
			// OAuth2 Apps
			r.With(roleRequired(deps.Audits, roleAdmin)).Get("/oauth2/apps", listOAuth2AppsHandler(deps.OAuth2Apps))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/oauth2/apps", createOAuth2AppHandler(deps.OAuth2Apps, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/oauth2/apps/{id}", deleteOAuth2AppHandler(deps.OAuth2Apps, deps.IAMAudit))
			// API Tokens
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/tokens", listAPITokensHandler(deps.APITokens))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Post("/tokens", createAPITokenHandler(deps.APITokens, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Delete("/tokens/{id}", deleteAPITokenHandler(deps.APITokens, deps.IAMAudit))
			// RBAC Roles
			r.With(roleRequired(deps.Audits, roleAdmin)).Get("/roles", listIAMRolesHandler(deps.IAMRoles))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/roles", createIAMRoleHandler(deps.IAMRoles, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/roles/{id}", deleteIAMRoleHandler(deps.IAMRoles, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin)).Get("/roles/assignments", listIAMRoleAssignmentsHandler(deps.IAMRoleAssignments))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/roles/assignments", createIAMRoleAssignmentHandler(deps.IAMRoleAssignments, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/roles/assignments/{id}", deleteIAMRoleAssignmentHandler(deps.IAMRoleAssignments, deps.IAMAudit))
			// Approval Workflows
			r.With(roleRequired(deps.Audits, roleAdmin)).Get("/approvals/workflows", listApprovalWorkflowsHandler(deps.ApprovalWorkflows))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/approvals/workflows", createApprovalWorkflowHandler(deps.ApprovalWorkflows, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin)).Delete("/approvals/workflows/{id}", deleteApprovalWorkflowHandler(deps.ApprovalWorkflows, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Get("/approvals/requests", listApprovalRequestsHandler(deps.ApprovalRequests))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/approvals/requests/{id}/approve", approveRequestHandler(deps.ApprovalRequests, deps.ApprovalDecisions, deps.IAMAudit))
			r.With(roleRequired(deps.Audits, roleAdmin)).Post("/approvals/requests/{id}/reject", rejectRequestHandler(deps.ApprovalRequests, deps.ApprovalDecisions, deps.IAMAudit))
			// Audit
			r.With(roleRequired(deps.Audits, roleAdmin)).Get("/audit", iamAuditLogHandler(deps.IAMAudit))
		})

		// --- Developer Platform ---
		r.Route("/developer", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Use(rateLimitMW(apiLimiter))
			// Webhooks
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Get("/webhooks", listWebhookSubscriptionsHandler(deps.WebhookSubs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/webhooks", createWebhookSubscriptionHandler(deps.WebhookSubs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/webhooks", deleteWebhookSubscriptionHandler(deps.WebhookSubs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/webhooks/toggle", toggleWebhookSubscriptionHandler(deps.WebhookSubs))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Get("/webhooks/events", webhookEventsHandler())
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Get("/webhooks/deliveries", webhookDeliveriesHandler(deps.WebhookDeliveries))
			// API Keys
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator, roleViewer)).Get("/api-keys", listAPIKeysHandler(deps.DeveloperKeys))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/api-keys", createAPIKeyHandler(deps.DeveloperKeys))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Delete("/api-keys", deleteAPIKeyHandler(deps.DeveloperKeys))
			r.With(roleRequired(deps.Audits, roleAdmin, roleOperator)).Post("/api-keys/toggle", toggleAPIKeyHandler(deps.DeveloperKeys))
		})
	})

	return r
}
