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
	Pool             *pgxpool.Pool
	NC               *nats.Conn
	RDB              *redis.Client
	Cfg              *config.Config
	CA               *ca.CA
	Users            *db.UserRepository
	Devices          *db.DeviceRepository
	Commands         *db.CommandRepository
	Containers       *db.ContainerRepository
	ContainerActions *db.ContainerActionRepository
	DockerImages     *db.DockerImageRepository
	DockerVolumes    *db.DockerVolumeRepository
	DockerNetworks   *db.DockerNetworkRepository
	SystemdServices      *db.SystemdServiceRepository
	ServiceActions       *db.ServiceActionRepository
	NetworkInterfaces    *db.NetworkInterfaceRepository
	StorageDevices       *db.StorageDeviceRepository
	VirtualMachines      *db.VirtualMachineRepository
	UpdateStatus         *db.UpdateStatusRepository
	FileActions      *db.FileActionRepository
	Audits           *db.AuditRepository
	AdminID          string
	Hub              *ws.Hub
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
		r.Post("/commands/pending", pendingCommandsHandler(deps.Commands, deps.Devices))
		r.Post("/commands/{cmdId}/result", submitResultHandler(deps.Commands, deps.Devices, deps.Audits, deps.Hub))
		r.Post("/devices/{id}/containers", syncContainersHandler(deps.Containers, deps.Devices))
		r.Post("/devices/{id}/docker-images", syncDockerImagesHandler(deps.DockerImages, deps.Devices))
		r.Post("/devices/{id}/docker-volumes", syncDockerVolumesHandler(deps.DockerVolumes, deps.Devices))
		r.Post("/devices/{id}/docker-networks", syncDockerNetworksHandler(deps.DockerNetworks, deps.Devices))
		r.Post("/devices/{id}/systemd-services", syncSystemdServicesHandler(deps.SystemdServices, deps.Devices))
		r.Post("/devices/{id}/network-interfaces", syncNetworkInterfacesHandler(deps.NetworkInterfaces, deps.Devices))
		r.Post("/devices/{id}/storage-devices", syncStorageDevicesHandler(deps.StorageDevices, deps.Devices))
		r.Post("/devices/{id}/virtual-machines", syncVirtualMachinesHandler(deps.VirtualMachines, deps.Devices))
		r.Post("/devices/{id}/updates", syncUpdatesHandler(deps.UpdateStatus, deps.Devices))
		r.Post("/container-actions/pending", pendingContainerActionsHandler(deps.ContainerActions, deps.Devices))
		r.Post("/container-actions/{actionId}/result", submitContainerActionResultHandler(deps.ContainerActions, deps.Devices, deps.Audits, deps.Hub))
		r.Post("/service-actions/pending", pendingServiceActionsHandler(deps.ServiceActions, deps.Devices))
		r.Post("/service-actions/{actionId}/result", submitServiceActionResultHandler(deps.ServiceActions, deps.Devices, deps.Audits, deps.Hub))
		r.Post("/file-actions/pending", pendingFileActionsHandler(deps.FileActions, deps.Devices))
		r.Post("/file-actions/{actionId}/result", submitFileActionResultHandler(deps.FileActions, deps.Devices, deps.Audits, deps.Hub))

		// Admin-facing — requires JWT auth
		r.Route("/devices", func(r chi.Router) {
			r.Use(authRequired(deps.Cfg.JWTSecret))
			r.Get("/", listDevicesHandler(deps.Devices))
			r.Get("/{id}", getDeviceHandler(deps.Devices))
			r.Post("/enroll-token", generateEnrollTokenHandler(deps.Cfg))
			r.Route("/{id}/commands", func(r chi.Router) {
				r.Post("/", requestCommandHandler(deps.Commands, deps.Devices, deps.Audits, deps.Hub))
				r.Get("/", listCommandsHandler(deps.Commands))
				r.Get("/{cmdId}", getCommandHandler(deps.Commands))
			})
			r.Get("/{id}/containers", listContainersHandler(deps.Containers))
			r.Get("/{id}/docker-images", listDockerImagesHandler(deps.DockerImages))
			r.Get("/{id}/docker-volumes", listDockerVolumesHandler(deps.DockerVolumes))
			r.Get("/{id}/docker-networks", listDockerNetworksHandler(deps.DockerNetworks))
			r.Get("/{id}/systemd-services", listSystemdServicesHandler(deps.SystemdServices))
			r.Get("/{id}/network-interfaces", listNetworkInterfacesHandler(deps.NetworkInterfaces))
			r.Get("/{id}/storage-devices", listStorageDevicesHandler(deps.StorageDevices))
			r.Get("/{id}/virtual-machines", listVirtualMachinesHandler(deps.VirtualMachines))
			r.Post("/{id}/vms/{domainName}/{action}", requestVMActionHandler(deps.VirtualMachines, deps.Devices, deps.Audits, deps.Hub))
			r.Get("/{id}/updates", listUpdatesHandler(deps.UpdateStatus))
			r.Post("/{id}/updates/{action}", requestUpdateActionHandler(deps.UpdateStatus, deps.Devices, deps.Audits, deps.Hub))
			r.Post("/{id}/services/{unitName}/{action}", requestServiceActionHandler(deps.ServiceActions, deps.Devices, deps.Audits, deps.Hub))
			r.Post("/{id}/containers/{containerId}/{action}", requestContainerActionHandler(deps.ContainerActions, deps.Devices, deps.Audits, deps.Hub))
			r.Post("/{id}/files/{actionType}", requestFileActionHandler(deps.FileActions, deps.Devices, deps.Audits, deps.Hub))
			r.Get("/file-actions/{actionId}", getFileActionResultHandler(deps.FileActions))
		})
	})

	return r
}
