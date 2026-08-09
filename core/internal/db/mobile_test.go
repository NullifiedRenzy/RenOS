package db

import (
	"context"
	"testing"
	"time"
)

func TestPushRegistrationFields(t *testing.T) {
	reg := &PushRegistration{
		ID:          "test-id",
		UserID:      "user-1",
		Platform:    "fcm",
		Token:       "token-abc",
		DeviceName:  "Pixel 8",
		DeviceModel: "Google Pixel 8",
		OSVersion:   "Android 14",
		AppVersion:  "1.0.0",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if reg.ID != "test-id" { t.Error("expected ID to be test-id") }
	if reg.UserID != "user-1" { t.Error("expected UserID to be user-1") }
	if reg.Platform != "fcm" { t.Error("expected Platform to be fcm") }
	if !reg.Enabled { t.Error("expected Enabled to be true") }
}

func TestMobileSessionFields(t *testing.T) {
	s := &MobileSession{
		ID:           "session-1",
		UserID:       "user-1",
		DeviceID:     "device-1",
		DeviceName:   "iPhone 15",
		BiometricEnabled: true,
		PushEnabled:  true,
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
	}

	if s.BiometricEnabled != true { t.Error("expected BiometricEnabled") }
	if s.PushEnabled != true { t.Error("expected PushEnabled") }
	if s.ExpiresAt.Before(time.Now()) { t.Error("expected ExpiresAt to be in the future") }
}

func TestOfflineSyncItemFields(t *testing.T) {
	item := &OfflineSyncItem{
		ID:         "sync-1",
		UserID:     "user-1",
		DeviceID:   "device-1",
		Action:     "command.execute",
		Payload:    []byte(`{"command":"ls"}`),
		Priority:   10,
		Status:     "pending",
		RetryCount: 0,
		MaxRetries: 3,
	}

	if item.Action != "command.execute" { t.Error("expected action") }
	if item.Priority != 10 { t.Error("expected priority 10") }
	if item.MaxRetries != 3 { t.Error("expected max retries 3") }
}

func TestDeviceApprovalFields(t *testing.T) {
	d := &DeviceApproval{
		ID:          "approval-1",
		UserID:      "user-1",
		DeviceID:    "device-1",
		DeviceName:  "MacBook Pro",
		Status:      "pending",
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	}

	if d.Status != "pending" { t.Error("expected pending status") }
}

func TestMobileSummaryFields(t *testing.T) {
	s := &MobileSummary{
		ID:          "summary-1",
		UserID:      "user-1",
		SummaryType: "overview",
		Data:        []byte(`{"total_devices":5}`),
		Version:     1,
	}

	if s.SummaryType != "overview" { t.Error("expected overview type") }
	if s.Version != 1 { t.Error("expected version 1") }
}

func TestClusterNodeFields(t *testing.T) {
	n := &ClusterNode{
		ID:       "node-1",
		NodeID:   "node-alpha",
		Hostname: "server-1",
		Address:  "192.168.1.10",
		Port:     8443,
		Role:     "leader",
		Status:   "active",
	}

	if n.Role != "leader" { t.Error("expected leader role") }
	if n.Port != 8443 { t.Error("expected port 8443") }
}

func TestClusterTaskFields(t *testing.T) {
	task := &ClusterTask{
		ID:         "task-1",
		TaskType:   "deploy.run",
		Priority:   5,
		Status:     "pending",
		MaxRetries: 3,
	}

	if task.TaskType != "deploy.run" { t.Error("expected task type") }
	if task.Priority != 5 { t.Error("expected priority 5") }
}

func TestNewRepositories(t *testing.T) {
	_ = NewPushRegistrationRepository(nil)
	_ = NewMobileSessionRepository(nil)
	_ = NewOfflineSyncRepository(nil)
	_ = NewDeviceApprovalRepository(nil)
	_ = NewMobileSummaryRepository(nil)
	_ = NewClusterNodeRepository(nil)
	_ = NewClusterElectionRepository(nil)
	_ = NewClusterTaskRepository(nil)
	_ = NewClusterStateRepository(nil)
	_ = NewClusterMetricsRepository(nil)
	_ = NewUpdateHistoryRepository(nil)
	_ = NewUpdateBackupRepository(nil)
}

func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Error("expected DeadlineExceeded")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("context should have been cancelled")
	}
}
