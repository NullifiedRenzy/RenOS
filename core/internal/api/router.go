package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"github.com/renos-project/renos/core/internal/ca"
	"github.com/renos-project/renos/core/internal/config"
	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/ws"
)

type Dependencies struct {
	Pool    *pgxpool.Pool
	NC      *nats.Conn
	RDB     *redis.Client
	Cfg     *config.Config
	CA      *ca.CA
	Users   *db.UserRepository
	Devices *db.DeviceRepository
	Audits  *db.AuditRepository
	AdminID string
	Hub     *ws.Hub
}

func NewRouter(deps *Dependencies) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(recovery)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", healthHandler(deps.Pool, deps.NC, deps.RDB))
		r.Get("/readyz", readyHandler(deps.Pool, deps.NC, deps.RDB))

		r.Post("/auth/login", loginHandler(deps.Users, deps.Cfg.JWTSecret))

		// WebSocket — JWT auth via query param
		r.Get("/ws", ws.ServeWS(deps.Hub, deps.Cfg.JWTSecret))

		// Agent-facing — no JWT auth (uses enrollment token or device_secret)
		r.Post("/devices/enroll", enrollHandler(deps.CA, deps.Devices, deps.Audits, deps.Cfg, deps.AdminID))
		r.Post("/devices/connect", connectHandler(deps.Devices, deps.Audits, deps.Hub))
		r.Post("/devices/heartbeat", heartbeatHandler(deps.Devices, deps.Audits, deps.Hub))

		// Admin-facing — requires JWT auth
		r.Route("/devices", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Get("/", listDevicesHandler(deps.Devices))
			r.Get("/{id}", getDeviceHandler(deps.Devices))
			r.Post("/enroll-token", generateEnrollTokenHandler(deps.Cfg))
		})
	})

	return r
}
