package bus

import (
	"fmt"
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
	)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	return nc, nil
}
