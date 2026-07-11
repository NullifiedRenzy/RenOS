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

	"github.com/renos-project/renos/core/internal/api"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/bus"
	"github.com/renos-project/renos/core/internal/ca"
	"github.com/renos-project/renos/core/internal/cache"
	"github.com/renos-project/renos/core/internal/config"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("starting RenOS Core", "version", "0.1.0", "port", cfg.Port)

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
	audits := db.NewAuditRepository(pool)

	adminID, err := seedAdmin(users, cfg)
	if err != nil {
		slog.Error("failed to seed admin user", "error", err)
		os.Exit(1)
	}
	slog.Info("admin user ready", "email", cfg.AdminEmail)

	eventHub := ws.NewHub()
	go eventHub.Run()
	slog.Info("WebSocket hub started")

	deps := &api.Dependencies{
		Pool:             pool,
		NC:               nc,
		RDB:              rdb,
		Cfg:              cfg,
		CA:               caInstance,
		Users:            users,
		Devices:          devices,
		Commands:         commands,
		Containers:       containers,
		ContainerActions: containerActions,
		DockerImages:     dockerImages,
		DockerVolumes:    dockerVolumes,
		DockerNetworks:   dockerNetworks,
		SystemdServices:  systemdServices,
		ServiceActions:   serviceActions,
		FileActions:      fileActions,
		Audits:           audits,
		AdminID:          adminID,
		Hub:              eventHub,
	}

	router := api.NewRouter(deps)
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
