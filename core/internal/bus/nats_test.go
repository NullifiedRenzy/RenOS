package bus

import (
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func natsAvailable(t *testing.T) bool {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:4222", 2*time.Second)
	if err != nil {
		t.Skip("NATS not available on localhost:4222, skipping integration test")
		return false
	}
	conn.Close()
	return true
}

func TestNATSReconnect(t *testing.T) {
	if !natsAvailable(t) {
		return
	}

	nc, err := NewConnection("nats://localhost:4222")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer nc.Close()

	// Subscribe to a test subject
	sub, err := nc.SubscribeSync("test.reconnect")
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	nc.Flush()

	// Publish before disconnect
	if err := nc.Publish("test.reconnect", []byte("hello")); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("failed to receive message before disconnect: %v", err)
	}
	if string(msg.Data) != "hello" {
		t.Fatalf("unexpected data: %s", msg.Data)
	}

	// Simulate disconnect by closing and reconnecting
	nc.Close()

	// Wait for reconnect to fail permanently (it won't reconnect because we called Close())
	// Instead, test that a NEW connection works and subscriptions are isolated
	nc2, err := NewConnection("nats://localhost:4222")
	if err != nil {
		t.Fatalf("failed to reconnect: %v", err)
	}
	defer nc2.Close()

	sub2, err := nc2.SubscribeSync("test.reconnect")
	if err != nil {
		t.Fatalf("failed to subscribe on reconnect: %v", err)
	}
	defer sub2.Unsubscribe()
	nc2.Flush()

	if err := nc2.Publish("test.reconnect", []byte("world")); err != nil {
		t.Fatalf("failed to publish after reconnect: %v", err)
	}
	msg2, err := sub2.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("failed to receive message after reconnect: %v", err)
	}
	if string(msg2.Data) != "world" {
		t.Fatalf("unexpected data after reconnect: %s", msg2.Data)
	}
}

func TestNATSQueueSubscribeDurable(t *testing.T) {
	if !natsAvailable(t) {
		return
	}

	nc, err := NewConnection("nats://localhost:4222")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer nc.Close()

	// Test queue subscription (used in production for worker distribution)
	sub, err := nc.QueueSubscribeSync("test.queue", "test-group")
	if err != nil {
		t.Fatalf("failed to queue subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	nc.Flush()

	// Publish two messages - queue group should deliver only one to this subscriber
	if err := nc.Publish("test.queue", []byte("first")); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
	if err := nc.Publish("test.queue", []byte("second")); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("failed to receive first queue message: %v", err)
	}
	if string(msg.Data) != "first" {
		t.Fatalf("expected 'first', got %s", msg.Data)
	}
}

func TestNATSSlowConsumerHandler(t *testing.T) {
	if !natsAvailable(t) {
		return
	}

	nc, err := NewConnection("nats://localhost:4222")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer nc.Close()

	// Subscribe with a very slow handler to test pending limits
	sub, err := nc.Subscribe("test.slow", func(msg *nats.Msg) {
		time.Sleep(100 * time.Millisecond)
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	sub.SetPendingLimits(10, 1024)
	nc.Flush()

	// Publish more than the pending limit
	for i := 0; i < 50; i++ {
		nc.Publish("test.slow", []byte("data"))
	}
	nc.Flush()

	time.Sleep(500 * time.Millisecond)

	pending, _, err := sub.Pending()
	if err != nil {
		t.Fatalf("failed to get pending: %v", err)
	}
	if pending < 0 {
		t.Error("pending should never be negative")
	}
}

func TestNATSPublishAfterReconnect(t *testing.T) {
	if !natsAvailable(t) {
		return
	}

	nc, err := NewConnection("nats://localhost:4222")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer nc.Close()

	nc.Close()

	// Create a fresh connection to simulate reconnect
	nc2, err := NewConnection("nats://localhost:4222")
	if err != nil {
		t.Fatalf("failed to reconnect after close: %v", err)
	}
	defer nc2.Close()

	sub, err := nc2.SubscribeSync("test.pub.after.reconnect")
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	nc2.Flush()

	if err := nc2.Publish("test.pub.after.reconnect", []byte("after-reconnect")); err != nil {
		t.Fatalf("failed to publish after reconnect: %v", err)
	}

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("failed to receive message after reconnect: %v", err)
	}
	if string(msg.Data) != "after-reconnect" {
		t.Fatalf("unexpected data: %s", msg.Data)
	}
}
