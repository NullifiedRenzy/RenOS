package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	stateFilePath = "/var/lib/renos/upgrade-state.json"
	maxHistory    = 50
)

type UpgradeRecord struct {
	ID          string    `json:"id"`
	Component   string    `json:"component"`
	FromVersion string    `json:"from_version"`
	ToVersion   string    `json:"to_version"`
	Status      string    `json:"status"`
	BackupID    string    `json:"backup_id"`
	Timestamp   time.Time `json:"timestamp"`
}

type UpgradeState struct {
	mu               sync.RWMutex
	CurrentVersion   string                   `json:"current_version"`
	History          []UpgradeRecord          `json:"history"`
	LastBackupID     string                   `json:"last_backup_id"`
	ComponentVersion map[string]string        `json:"component_versions"`
	filePath         string
}

func NewUpgradeState() *UpgradeState {
	return &UpgradeState{
		CurrentVersion:   "0.0.0",
		ComponentVersion: make(map[string]string),
		filePath:         stateFilePath,
	}
}

func Load() (*UpgradeState, error) {
	s := NewUpgradeState()
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	if s.ComponentVersion == nil {
		s.ComponentVersion = make(map[string]string)
	}
	return s, nil
}

func (s *UpgradeState) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state directory %s: %w", dir, err)
	}

	tmpPath := s.filePath + ".tmp"
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp state file: %w", err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

func (s *UpgradeState) RecordUpgrade(component, from, to, status, backupID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := UpgradeRecord{
		ID:          fmt.Sprintf("upg-%d", time.Now().UnixNano()),
		Component:   component,
		FromVersion: from,
		ToVersion:   to,
		Status:      status,
		BackupID:    backupID,
		Timestamp:   time.Now(),
	}

	s.History = append(s.History, record)
	if len(s.History) > maxHistory {
		s.History = s.History[len(s.History)-maxHistory:]
	}

	s.CurrentVersion = to
	if component != "" && component != "all" {
		s.ComponentVersion[component] = to
	}

	return s.Save()
}

func (s *UpgradeState) ListHistory() []UpgradeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := len(s.History)
	if n > maxHistory {
		n = maxHistory
	}
	result := make([]UpgradeRecord, n)
	copy(result, s.History[len(s.History)-n:])

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	return result
}

func (s *UpgradeState) GetUpgrade(id string) *UpgradeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := len(s.History) - 1; i >= 0; i-- {
		if s.History[i].ID == id {
			return &s.History[i]
		}
	}
	return nil
}

func (s *UpgradeState) ComponentVersionOf(component string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.ComponentVersion[component]; ok {
		return v
	}
	return "0.0.0"
}
