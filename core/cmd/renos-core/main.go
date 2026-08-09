package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renos-project/renos/core/internal/ai"
	_ "github.com/renos-project/renos/core/internal/ai/providers"
	"github.com/renos-project/renos/core/internal/api"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/bus"
	"github.com/renos-project/renos/core/internal/ca"
	"github.com/renos-project/renos/core/internal/cache"
	"github.com/renos-project/renos/core/internal/config"
	"github.com/renos-project/renos/core/internal/crashreport"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/developer"
	"github.com/renos-project/renos/core/internal/execctrl"
	"github.com/renos-project/renos/core/internal/observability"
	"github.com/renos-project/renos/core/internal/notify"
	"github.com/renos-project/renos/core/internal/plugin"
	"github.com/renos-project/renos/core/internal/policy"
	"github.com/renos-project/renos/core/internal/scheduler"
	"github.com/renos-project/renos/core/internal/telemetry"
	"github.com/renos-project/renos/core/internal/worker"
	"github.com/renos-project/renos/core/internal/ws"
)

var version = "dev"
var commit = "unknown"
var buildTime = "unknown"

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	// Configure the crash reporter so panics are persisted to the crash
	// directory before the process dies.
	crashreport.Configure(crashreport.Options{
		App:       "renos-core",
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
		CrashDir:  cfg.CrashDir,
		LogFile:   os.Getenv("RENOS_LOG_FILE"),
	})
	defer crashreport.CapturePanic()

	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("RenOS Core %s (commit: %s, built: %s)\n", version, commit, buildTime)
		return
	}

	if err := config.Validate(cfg); err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	api.Version = version
	api.Commit = commit
	api.BuildTime = buildTime

	slog.Info("starting RenOS Core", "version", version, "port", cfg.Port, "production", cfg.IsProduction())

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if len(os.Args) > 2 && os.Args[2] == "down" {
			steps := 1
			if len(os.Args) > 3 {
				if s, err := strconv.Atoi(os.Args[3]); err == nil && s > 0 {
					steps = s
				}
			}
			if err := db.RollbackMigrations(cfg.DatabaseURL, steps); err != nil {
				slog.Error("rollback failed", "error", err)
				os.Exit(1)
			}
			slog.Info("rollback complete", "steps", steps)
			return
		}
		if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations complete")
		return
	}

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("connected to database")

	nc, err := bus.NewConnection(cfg.NATSUrl)
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Close()
	slog.Info("connected to NATS")

	var rdb *redis.Client
	if cfg.RedisURL != "" {
		client, err := cache.NewClient(cfg.RedisURL)
		if err != nil {
			slog.Warn("failed to connect to Redis, continuing without cache", "error", err)
		} else {
			rdb = client
			defer rdb.Close()
			slog.Info("connected to Redis")
		}
	}

	caInstance, err := ca.LoadOrGenerate(cfg.CACertDir, cfg.CACertTTL)
	if err != nil {
		slog.Error("failed to initialize CA", "error", err)
		os.Exit(1)
	}
	slog.Info("CA initialized", "subject", caInstance.Cert.Subject.CommonName)

	users := db.NewUserRepository(pool)
	devices := db.NewDeviceRepository(pool)
	commands := db.NewCommandRepository(pool)
	containers := db.NewContainerRepository(pool)
	containerActions := db.NewContainerActionRepository(pool)
	dockerImages := db.NewDockerImageRepository(pool)
	dockerVolumes := db.NewDockerVolumeRepository(pool)
	dockerNetworks := db.NewDockerNetworkRepository(pool)
	systemdServices := db.NewSystemdServiceRepository(pool)
	serviceActions := db.NewServiceActionRepository(pool)
	fileActions := db.NewFileActionRepository(pool)
	fileTransfers := db.NewFileTransferRepository(pool)
	filePermissions := db.NewFilePermissionRepository(pool)
	networkInterfaces := db.NewNetworkInterfaceRepository(pool)
	storageDevices := db.NewStorageDeviceRepository(pool)
	virtualMachines := db.NewVirtualMachineRepository(pool)
	updateStatus := db.NewUpdateStatusRepository(pool)
	plugins := db.NewPluginRepository(pool)
	aiConversations := db.NewAIConversationRepository(pool)
	audits := db.NewAuditRepository(pool)
	groups := db.NewGroupRepository(pool)
	groupActions := db.NewGroupActionRepository(pool)
	jobs := db.NewJobRepository(pool)
	schedules := db.NewScheduleRepository(pool)
	scheduleRuns := db.NewScheduleRunRepository(pool)
	policies := db.NewPolicyRepository(pool)
	eventHub := ws.NewHub()
	go eventHub.Run()
	slog.Info("WebSocket hub started")

	systemLogs := db.NewSystemLogRepository(pool)
	executions := db.NewExecutionRepository(pool)
	processes := db.NewProcessRepository(pool)
	composeStacks := db.NewComposeStackRepository(pool)
	containerMetrics := db.NewContainerMetricsRepository(pool)
	proxmox := db.NewProxmoxRepository(pool)
	terminalSessions := db.NewTerminalSessionRepository(pool)
	terminalRecordings := db.NewTerminalRecordingRepository(pool)
	installedPlugins := db.NewInstalledPluginRepository(pool)
	deviceTags := db.NewDeviceTagRepository(pool)
	deviceInventory := db.NewDeviceInventoryRepository(pool)
	compliance := db.NewComplianceRepository(pool)
	bulkOperations := db.NewBulkOperationRepository(pool)
	workflows := db.NewWorkflowRepository(pool)
	workflowSteps := db.NewWorkflowStepRepository(pool)
	workflowRuns := db.NewWorkflowRunRepository(pool)
	workflowStepRuns := db.NewWorkflowStepRunRepository(pool)
	workflowTemplates := db.NewWorkflowTemplateRepository(pool)
	monitoringMetrics := db.NewMonitoringMetricRepository(pool)
	alertRules := db.NewAlertRuleRepository(pool)
	alertInstances := db.NewAlertInstanceRepository(pool)
	monitoringDashboards := db.NewMonitoringDashboardRepository(pool)
	monitoringWidgets := db.NewMonitoringWidgetRepository(pool)
	monitoringLogs := db.NewMonitoringLogRepository(pool)
	widgetData := &api.DefaultWidgetDataProvider{
		Devices:    devices,
		Alerts:     alertInstances,
		Logs:       monitoringLogs,
		Workflows:  workflows,
		Runs:       workflowRuns,
		AI:         aiConversations,
		Containers: containers,
		Proxmox:    proxmox,
		Metrics:    monitoringMetrics,
	}
	vaultSecrets := db.NewVaultSecretRepository(pool)
	vaultVersions := db.NewVaultSecretVersionRepository(pool)
	vaultKeys := db.NewVaultKeyRepository(pool)
	vaultPolicies := db.NewVaultPolicyRepository(pool)
	vaultGrants := db.NewVaultGrantRepository(pool)
	vaultAudit := db.NewVaultAuditRepository(pool)
	notifProviders := db.NewNotificationProviderRepository(pool)
	notifTemplates := db.NewNotificationTemplateRepository(pool)
	notifRules := db.NewNotificationRuleRepository(pool)
	notifHistory := db.NewNotificationHistoryRepository(pool)
	inAppNotifications := db.NewInAppNotificationRepository(pool)
	marketplacePlugins := db.NewMarketplacePluginRepository(pool)
	marketplaceInstallations := db.NewMarketplaceInstallationRepository(pool)
	marketplaceHistory := db.NewMarketplaceInstallHistoryRepository(pool)
	searchRepo := db.NewSearchRepository(pool)
	identityProviders := db.NewIdentityProviderRepository(pool)
	oauth2Apps := db.NewOAuth2AppRepository(pool)
	apiTokens := db.NewAPITokenRepository(pool)
	iamRoles := db.NewIAMRoleRepository(pool)
	iamRoleAssignments := db.NewIAMRoleAssignmentRepository(pool)
	approvalWorkflows := db.NewApprovalWorkflowRepository(pool)
	approvalRequests := db.NewApprovalRequestRepository(pool)
	approvalDecisions := db.NewApprovalDecisionRepository(pool)
	iamAudit := db.NewIAMAuditRepository(pool)
	updateHistory := db.NewUpdateHistoryRepository(pool)
	updateBackups := db.NewUpdateBackupRepository(pool)
	clusterNodes := db.NewClusterNodeRepository(pool)
	clusterElections := db.NewClusterElectionRepository(pool)
	clusterTasks := db.NewClusterTaskRepository(pool)
	clusterState := db.NewClusterStateRepository(pool)
	clusterMetrics := db.NewClusterMetricsRepository(pool)
	pushRegistrations := db.NewPushRegistrationRepository(pool)
	mobileSessions := db.NewMobileSessionRepository(pool)
	offlineSync := db.NewOfflineSyncRepository(pool)
	deviceApprovals := db.NewDeviceApprovalRepository(pool)
	mobileSummaries := db.NewMobileSummaryRepository(pool)
	webhookSubs := developer.NewWebhookSubscriptionRepository(pool)
	webhookDeliveries := developer.NewWebhookDeliveryRepository(pool)
	developerKeys := developer.NewAPIKeyRepository(pool)
	policyEnforcer := policy.NewEnforcer(policies, audits, systemLogs, eventHub)

	// Initialize RenAI
	aiCtxBuilder := ai.NewContextBuilder(devices, commands, containers, composeStacks, processes, jobs, schedules, policies, installedPlugins, systemLogs, containerMetrics, proxmox)
	aiProviderConfig := ai.ProviderConfig{
		Type:      cfg.AIProvider,
		APIKey:    getAIKey(cfg),
		BaseURL:   getAIEndpoint(cfg),
		Model:     getAIModel(cfg),
		Timeout:   cfg.AITimeout,
		MaxTokens: cfg.AIMaxTokens,
	}
	aiProvider, err := ai.NewProvider(cfg.AIProvider, aiProviderConfig)
	if err != nil {
		slog.Warn("failed to initialize AI provider, AI features will be unavailable", "error", err, "provider", cfg.AIProvider)
	}
	renai := ai.NewRenAI(aiProvider, aiCtxBuilder, aiConversations)
	slog.Info("RenAI initialized", "provider", cfg.AIProvider)

	execctrl.SetRetryableCodes(cfg.RetryableErrorCodes)

	adminID, err := seedAdmin(users, cfg)
	if err != nil {
		slog.Error("failed to seed admin user", "error", err)
		os.Exit(1)
	}
	slog.Info("admin user ready", "email", cfg.AdminEmail)

	notifier := notify.New(inAppNotifications, adminID)

	// ── Initialize Observability & SRE subsystems ──
	obsCfg := &observability.ObservabilityConfig{
		Tracing: observability.TracingConfig{
			Enabled:     cfg.TracingEnabled,
			Exporter:    cfg.TracingExporter,
			Endpoint:    cfg.TracingEndpoint,
			ServiceName: cfg.ServiceName,
			Environment: cfg.Environment,
			SampleRate:  cfg.TracingSampleRate,
		},
		Metrics: observability.MetricsConfig{
			Enabled:     cfg.MetricsEnabled,
			Path:        cfg.MetricsPath,
			Aggregation: 15 * time.Second,
		},
		Pprof: observability.PprofConfig{
			Enabled: cfg.PprofEnabled,
			Path:    cfg.PprofPath,
		},
		SLO: observability.SLOConfig{
			Enabled: cfg.SLOEnabled,
			Path:    cfg.SLOPath,
		},
		Health: observability.HealthConfig{
			DeepEnabled: cfg.DeepHealthEnabled,
			Path:        cfg.DeepHealthPath,
		},
	}
	obsHandler, obsCleanup := observability.InitAll(obsCfg, &observability.Options{
		DB:      pool,
		NC:      nc,
		RDB:     rdb,
		Version: version,
	})
	defer obsCleanup()

	// Mount observability endpoints on main router
	if cfg.MetricsEnabled {
		http.Handle(cfg.MetricsPath, obsHandler)
	}
	slog.Info("observability subsystem initialized",
		"tracing", cfg.TracingEnabled,
		"metrics", cfg.MetricsEnabled,
		"pprof", cfg.PprofEnabled,
	)

	deps := &api.Dependencies{
		Pool:                     pool,
		NC:                       nc,
		RDB:                      rdb,
		Cfg:                      cfg,
		CA:                       caInstance,
		Users:                    users,
		Devices:                  devices,
		Commands:                 commands,
		Containers:               containers,
		ContainerActions:         containerActions,
		DockerImages:             dockerImages,
		DockerVolumes:            dockerVolumes,
		DockerNetworks:           dockerNetworks,
		SystemdServices:          systemdServices,
		ServiceActions:           serviceActions,
		FileActions:              fileActions,
		FileTransfers:            fileTransfers,
		FilePermissions:          filePermissions,
		NetworkInterfaces:        networkInterfaces,
		StorageDevices:           storageDevices,
		VirtualMachines:          virtualMachines,
		UpdateStatus:             updateStatus,
		Plugins:                  plugins,
		AIConversations:          aiConversations,
		Audits:                   audits,
		Groups:                   groups,
		GroupActions:             groupActions,
		Jobs:                     jobs,
		Schedules:                schedules,
		ScheduleRuns:             scheduleRuns,
		Policies:                 policies,
		PolicyEnforcer:           policyEnforcer,
		SystemLogs:               systemLogs,
		Executions:               executions,
		Processes:                processes,
		ComposeStacks:            composeStacks,
		ContainerMetrics:         containerMetrics,
		Proxmox:                  proxmox,
		TerminalSessions:         terminalSessions,
		TerminalRecordings:       terminalRecordings,
		InstalledPlugins:         installedPlugins,
		DeviceTags:               deviceTags,
		DeviceInventory:          deviceInventory,
		Compliance:               compliance,
		BulkOperations:           bulkOperations,
		Workflows:                workflows,
		WorkflowSteps:            workflowSteps,
		WorkflowRuns:             workflowRuns,
		WorkflowStepRuns:         workflowStepRuns,
		WorkflowTemplates:        workflowTemplates,
		MonitoringMetrics:        monitoringMetrics,
		AlertRules:               alertRules,
		AlertInstances:           alertInstances,
		MonitoringDashboards:     monitoringDashboards,
		MonitoringWidgets:        monitoringWidgets,
		MonitoringLogs:           monitoringLogs,
		WidgetData:               widgetData,
		VaultSecrets:             vaultSecrets,
		VaultVersions:            vaultVersions,
		VaultKeys:                vaultKeys,
		VaultPolicies:            vaultPolicies,
		VaultGrants:              vaultGrants,
		VaultAudit:               vaultAudit,
		NotifProviders:           notifProviders,
		NotifTemplates:           notifTemplates,
		NotifRules:               notifRules,
		NotifHistory:             notifHistory,
		InAppNotifications:       inAppNotifications,
		Notifier:                 notifier,
		MarketplacePlugins:       marketplacePlugins,
		Search:                   searchRepo,
		MarketplaceInstallations: marketplaceInstallations,
		MarketplaceHistory:       marketplaceHistory,
		IdentityProviders:        identityProviders,
		OAuth2Apps:               oauth2Apps,
		APITokens:                apiTokens,
		IAMRoles:                 iamRoles,
		IAMRoleAssignments:       iamRoleAssignments,
		ApprovalWorkflows:        approvalWorkflows,
		ApprovalRequests:         approvalRequests,
		ApprovalDecisions:        approvalDecisions,
		IAMAudit:                 iamAudit,
		UpdateHistory:            updateHistory,
		UpdateBackups:            updateBackups,
		ClusterNodes:             clusterNodes,
		ClusterElections:         clusterElections,
		ClusterTasks:             clusterTasks,
		ClusterState:             clusterState,
		ClusterMetrics:           clusterMetrics,
		PushRegistrations:        pushRegistrations,
		MobileSessions:           mobileSessions,
		OfflineSync:              offlineSync,
		DeviceApprovals:          deviceApprovals,
		MobileSummaries:          mobileSummaries,
		WebhookSubs:              webhookSubs,
		WebhookDeliveries:        webhookDeliveries,
		DeveloperKeys:            developerKeys,
		RenAI:                    renai,
		AdminID:                  adminID,
		Hub:                      eventHub,
		CrashDir:                 cfg.CrashDir,
		BackupsDir:               cfg.BackupsDir,
	}

	router := api.NewRouter(deps)

	// ── Production readiness: telemetry (opt-in, anonymous) ──
	telemetrySvc := setupTelemetry(cfg, devices, workflows, installedPlugins)
	defer telemetrySvc.Stop()

	// Initialize plugin loader
	pluginLoader := plugin.NewLoader(cfg.PluginDir, installedPlugins)
	deps.PluginLoader = pluginLoader
	if err := pluginLoader.Start(context.Background()); err != nil {
		slog.Error("failed to start plugin loader", "error", err)
	} else {
		slog.Info("plugin loader started", "dir", cfg.PluginDir)
	}

	// Subscribe to execution output and completion from agents
	if _, err := nc.Subscribe("exec.output.*", api.HandleExecutionOutput(nc, executions, eventHub)); err != nil {
		slog.Error("failed to subscribe to exec.output", "error", err)
	}
	if _, err := nc.Subscribe("exec.complete.*", api.HandleExecutionComplete(nc, executions, audits, eventHub)); err != nil {
		slog.Error("failed to subscribe to exec.complete", "error", err)
	}
	slog.Info("NATS execution subscribers registered")

	// Subscribe to process snapshots and action results from agents
	if _, err := nc.Subscribe("process.snapshot.*", api.HandleProcessSnapshot(processes, eventHub, processes)); err != nil {
		slog.Error("failed to subscribe to process.snapshot", "error", err)
	}
	if _, err := nc.Subscribe("process.result.*", api.HandleProcessActionResult(processes, audits, eventHub)); err != nil {
		slog.Error("failed to subscribe to process.result", "error", err)
	}
	slog.Info("NATS process subscribers registered")

	wp := worker.NewPool(jobs, commands, containerActions, serviceActions, groups, groupActions, audits, systemLogs, eventHub, devices, notifier,
		cfg.PoolSize,
		cfg.MaxConcurrentJobs,
		cfg.MaxConcurrentDeviceActions,
		cfg.DeviceActionCooldown,
		cfg.CircuitBreakerThreshold,
		cfg.CircuitBreakerResetAfter,
		cfg.DefaultMaxJobDuration)
	wp.Start()

	sched := scheduler.New(schedules, scheduleRuns, jobs, groupActions, groups, audits, systemLogs, eventHub, policyEnforcer)
	sched.Start()

	// Terminal session idle timeout reaper (30 minutes)
	go api.TerminalSessionReaper(terminalSessions, nc, 30*time.Minute)
	slog.Info("terminal session reaper started", "timeout", "30m")

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	offlineCtx, offlineCancel := context.WithCancel(context.Background())
	go func() {
		<-done
		offlineCancel()
	}()

	go func() {
		checkInterval := cfg.AgentOfflineTimeout / 3
		if checkInterval < time.Second {
			checkInterval = time.Second
		}
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()
		slog.Info("offline checker started", "interval", checkInterval, "timeout", cfg.AgentOfflineTimeout)
		for {
			select {
			case <-ticker.C:
				threshold := fmt.Sprintf("%d seconds", int(cfg.AgentOfflineTimeout.Seconds()))
				ids, err := devices.MarkDevicesOffline(offlineCtx, threshold)
				if err != nil {
					slog.Error("offline check failed", "error", err)
				} else if len(ids) > 0 {
					slog.Info("devices marked offline", "count", len(ids))
					for _, id := range ids {
						eventHub.Broadcast(ws.Message{
							Type:     "device.offline",
							DeviceID: id,
						})
						if dev, err := devices.FindByID(offlineCtx, id); err == nil && dev != nil {
							notifier.Notify(offlineCtx, dev.UserID, "alert", "Device offline",
								dev.Hostname+" went offline. It may need attention.", "device", id,
								map[string]any{"hostname": dev.Hostname})
						}
					}
				}
			case <-offlineCtx.Done():
				slog.Info("offline checker stopped")
				return
			}
		}
	}()

	<-done
	slog.Info("shutting down gracefully")

	sched.Stop()
	wp.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
}

func seedAdmin(users *db.UserRepository, cfg *config.Config) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := users.ExistsByEmail(ctx, cfg.AdminEmail)
	if err != nil {
		return "", fmt.Errorf("check admin exists: %w", err)
	}

	if exists {
		user, err := users.FindByEmail(ctx, cfg.AdminEmail)
		if err != nil {
			return "", fmt.Errorf("find existing admin: %w", err)
		}
		return user.ID, nil
	}

	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		return "", fmt.Errorf("hash admin password: %w", err)
	}

	user, err := users.Create(ctx, cfg.AdminEmail, hash, "Admin", "admin")
	if err != nil {
		return "", fmt.Errorf("create admin user: %w", err)
	}

	slog.Info("seeded admin user", "id", user.ID, "email", user.Email)
	return user.ID, nil
}

func getAIKey(cfg *config.Config) string {
	switch cfg.AIProvider {
	case "openai":
		return cfg.AIOpenAIKey
	case "anthropic":
		return cfg.AIAnthropicKey
	case "google":
		return cfg.AIGoogleKey
	default:
		return ""
	}
}

func getAIEndpoint(cfg *config.Config) string {
	switch cfg.AIProvider {
	case "ollama":
		return cfg.AIOllamaURL
	case "openai":
		return cfg.AIOpenAIBaseURL
	case "lmstudio", "vllm", "openrouter":
		return cfg.AIAIEndpoint
	default:
		return ""
	}
}

func getAIModel(cfg *config.Config) string {
	if cfg.AIAIModel != "" {
		return cfg.AIAIModel
	}
	switch cfg.AIProvider {
	case "ollama":
		return cfg.AIOllamaModel
	case "openai":
		return cfg.AIOpenAIModel
	case "anthropic":
		return cfg.AIAnthropicModel
	case "google":
		return cfg.AIGoogleModel
	default:
		return ""
	}
}

// setupTelemetry starts the opt-in anonymous telemetry sender. It is a no-op
// unless RENOS_TELEMETRY_ENABLED=true. The returned sender is safe to call
// Stop() on in all cases.
func setupTelemetry(cfg *config.Config, devices *db.DeviceRepository, workflows *db.WorkflowRepository, installedPlugins *db.InstalledPluginRepository) *telemetry.Telemetry {
	hostname, _ := os.Hostname()
	svc := telemetry.New(telemetry.Config{
		Enabled:  cfg.TelemetryEnabled,
		Endpoint: cfg.TelemetryEndpoint,
		Version:  version,
		Hostname: hostname,
	})

	if !cfg.TelemetryEnabled {
		slog.Info("telemetry disabled (opt-in; set RENOS_TELEMETRY_ENABLED=true to enable)")
		return svc
	}

	svc.Start(context.Background(), func() telemetry.Counts {
		return collectTelemetryCounts(devices, workflows, installedPlugins)
	})
	return svc
}

// collectTelemetryCounts gathers the anonymous counters included in each
// telemetry ping. Failures are swallowed — telemetry must never interfere
// with normal operation.
func collectTelemetryCounts(devices *db.DeviceRepository, workflows *db.WorkflowRepository, installedPlugins *db.InstalledPluginRepository) telemetry.Counts {
	counts := telemetry.Counts{}
	ctx := context.Background()
	if devices != nil {
		if list, err := devices.FindAll(ctx); err == nil {
			counts.Devices = len(list)
		}
	}
	if workflows != nil {
		if list, err := workflows.ListAll(ctx); err == nil {
			counts.Workflows = len(list)
		}
	}
	if installedPlugins != nil {
		if list, err := installedPlugins.ListAll(ctx); err == nil {
			counts.Plugins = len(list)
		}
	}
	return counts
}
