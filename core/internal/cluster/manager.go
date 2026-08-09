package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/renos-project/renos/core/internal/db"
)

type NodeRole string

const (
	RoleLeader    NodeRole = "leader"
	RoleFollower  NodeRole = "follower"
	RoleCandidate NodeRole = "candidate"
)

type NodeStatus string

const (
	StatusJoining     NodeStatus = "joining"
	StatusActive      NodeStatus = "active"
	StatusDraining    NodeStatus = "draining"
	StatusOffline     NodeStatus = "offline"
	StatusMaintenance NodeStatus = "maintenance"
)

type Manager struct {
	nodeID       string
	hostname     string
	address      string
	port         int
	role         NodeRole
	status       NodeStatus
	nodes        *db.ClusterNodeRepository
	elections    *db.ClusterElectionRepository
	tasks        *db.ClusterTaskRepository
	state        *db.ClusterStateRepository
	metrics      *db.ClusterMetricsRepository
	mu           sync.RWMutex
	heartbeatCh  chan struct{}
	stopCh       chan struct{}
}

func NewManager(nodeID, hostname, address string, port int, nodes *db.ClusterNodeRepository, elections *db.ClusterElectionRepository, tasks *db.ClusterTaskRepository, state *db.ClusterStateRepository, metrics *db.ClusterMetricsRepository) *Manager {
	if nodeID == "" {
		nodeID = uuid.New().String()
	}
	return &Manager{
		nodeID:   nodeID,
		hostname: hostname,
		address:  address,
		port:     port,
		role:     RoleFollower,
		status:   StatusJoining,
		nodes:    nodes,
		elections: elections,
		tasks:    tasks,
		state:    state,
		metrics:  metrics,
		heartbeatCh: make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
	}
}

func (m *Manager) Start(ctx context.Context) error {
	metadata, _ := json.Marshal(map[string]interface{}{
		"version": "0.1.0",
		"go":      "1.25",
	})

	node := &db.ClusterNode{
		NodeID:   m.nodeID,
		Hostname: m.hostname,
		Address:  m.address,
		Port:     m.port,
		Role:     string(m.role),
		Status:   string(StatusActive),
		Metadata: metadata,
	}

	if err := m.nodes.Register(ctx, node); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	m.status = StatusActive
	slog.Info("cluster node registered", "node_id", m.nodeID, "host", m.hostname)

	go m.heartbeatLoop(ctx)
	go m.electionLoop(ctx)
	go m.taskLoop(ctx)
	go m.stateCleanupLoop(ctx)

	return nil
}

func (m *Manager) Stop() {
	close(m.stopCh)
	m.nodes.UpdateStatus(context.Background(), m.nodeID, string(StatusOffline))
}

func (m *Manager) NodeID() string { return m.nodeID }
func (m *Manager) Role() NodeRole { m.mu.RLock(); defer m.mu.RUnlock(); return m.role }
func (m *Manager) Status() NodeStatus { m.mu.RLock(); defer m.mu.RUnlock(); return m.status }

func (m *Manager) SetMaintenanceMode(ctx context.Context, enabled bool) error {
	status := StatusActive
	if enabled { status = StatusMaintenance }
	m.mu.Lock()
	m.status = status
	m.mu.Unlock()
	return m.nodes.UpdateStatus(ctx, m.nodeID, string(status))
}

func (m *Manager) StartDraining(ctx context.Context) error {
	m.mu.Lock()
	m.status = StatusDraining
	m.mu.Unlock()
	return m.nodes.UpdateStatus(ctx, m.nodeID, string(StatusDraining))
}

func (m *Manager) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			if err := m.nodes.Heartbeat(ctx, m.nodeID); err != nil {
				slog.Error("heartbeat failed", "error", err)
			}
			m.recordMetrics(ctx)
		}
	}
}

func (m *Manager) electionLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkLeader(ctx)
		}
	}
}

func (m *Manager) checkLeader(ctx context.Context) {
	leader, err := m.nodes.FindLeader(ctx)
	if err != nil || leader == nil {
		m.startElection(ctx)
		return
	}

	// Check if leader is alive
	if time.Since(leader.LastHeartbeat) > 15*time.Second {
		slog.Warn("leader heartbeat stale, starting election", "leader", leader.NodeID)
		m.startElection(ctx)
		return
	}

	if leader.NodeID == m.nodeID {
		m.mu.Lock()
		m.role = RoleLeader
		m.mu.Unlock()
	} else {
		m.mu.Lock()
		m.role = RoleFollower
		m.mu.Unlock()
	}
}

func (m *Manager) startElection(ctx context.Context) {
	term, _ := m.elections.CurrentTerm(ctx)
	term++

	m.mu.Lock()
	m.role = RoleCandidate
	m.mu.Unlock()

	election := &db.ClusterElection{
		CandidateID: m.nodeID,
		Term:        term,
	}
	if err := m.elections.Create(ctx, election); err != nil {
		slog.Error("failed to create election", "error", err)
		return
	}

	// Simple majority vote (single-node cluster = auto-win)
	allNodes, _ := m.nodes.ListAll(ctx)
	votesFor := 1
	votesAgainst := 0
	for _, node := range allNodes {
		if node.NodeID != m.nodeID && node.Status == string(StatusActive) {
			votesFor++
		}
	}

	majority := (len(allNodes) / 2) + 1
	if votesFor >= majority {
		m.elections.UpdateResult(ctx, election.ID, "won", votesFor, votesAgainst)
		m.nodes.UpdateRole(ctx, m.nodeID, string(RoleLeader))
		m.mu.Lock()
		m.role = RoleLeader
		m.mu.Unlock()
		slog.Info("won election", "term", term, "votes", votesFor)
	} else {
		m.elections.UpdateResult(ctx, election.ID, "lost", votesFor, votesAgainst)
		m.mu.Lock()
		m.role = RoleFollower
		m.mu.Unlock()
	}
}

func (m *Manager) taskLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			if m.Role() != RoleLeader { continue }
			pending, _ := m.tasks.ListPending(ctx, 10)
			for _, task := range pending {
				m.assignTask(ctx, task)
			}
			m.tasks.Rebalance(ctx)
		}
	}
}

func (m *Manager) assignTask(ctx context.Context, task *db.ClusterTask) {
	allNodes, _ := m.nodes.ListAll(ctx)
	for _, node := range allNodes {
		if node.Status == string(StatusActive) {
			if err := m.tasks.Claim(ctx, task.ID, node.NodeID); err == nil {
				slog.Info("task assigned", "task_id", task.ID, "node", node.NodeID)
				return
			}
		}
	}
	slog.Warn("no available nodes for task", "task_id", task.ID)
}

func (m *Manager) stateCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.state.CleanupExpired(ctx)
		}
	}
}

func (m *Manager) recordMetrics(ctx context.Context) {
	metric := &db.ClusterMetric{
		NodeID:      m.nodeID,
		CPUUsage:    0,
		MemoryUsage: 0,
		DiskUsage:   0,
	}
	m.metrics.Record(ctx, metric)
}

func (m *Manager) GetClusterSummary(ctx context.Context) (map[string]interface{}, error) {
	nodes, err := m.nodes.ListAll(ctx)
	if err != nil { return nil, err }

	leader, _ := m.nodes.FindLeader(ctx)
	leaderID := ""
	if leader != nil { leaderID = leader.NodeID }

	term, _ := m.elections.CurrentTerm(ctx)
	pending, _ := m.tasks.ListPending(ctx, 1000)
	totalTasks := len(pending)

	summary := map[string]interface{}{
		"node_count":    len(nodes),
		"leader":        leaderID,
		"term":          term,
		"pending_tasks": totalTasks,
		"nodes":         nodes,
	}
	return summary, nil
}
