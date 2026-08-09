package bus

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

func NewConnection(natsURL string) (*nats.Conn, error) {
	nc, err := nats.Connect(natsURL,
		nats.Timeout(10*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.Name("renos-core"),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			slog.Info("nats reconnected")
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			slog.Error("nats connection closed permanently")
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			slog.Error("nats error", "error", err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	return nc, nil
}
