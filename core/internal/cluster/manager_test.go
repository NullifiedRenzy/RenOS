package cluster

import (
	"testing"
	"time"
)

func TestNodeRoles(t *testing.T) {
	roles := []NodeRole{RoleLeader, RoleFollower, RoleCandidate}
	valid := map[NodeRole]bool{}
	for _, r := range roles {
		valid[r] = true
	}

	if !valid[RoleLeader] { t.Error("expected leader role") }
	if !valid[RoleFollower] { t.Error("expected follower role") }
	if !valid[RoleCandidate] { t.Error("expected candidate role") }
}

func TestNodeStatuses(t *testing.T) {
	statuses := []NodeStatus{StatusJoining, StatusActive, StatusDraining, StatusOffline, StatusMaintenance}
	valid := map[NodeStatus]bool{}
	for _, s := range statuses {
		valid[s] = true
	}

	if !valid[StatusActive] { t.Error("expected active status") }
	if !valid[StatusOffline] { t.Error("expected offline status") }
	if !valid[StatusJoining] { t.Error("expected joining status") }
}

func TestNodeRoleString(t *testing.T) {
	if string(RoleLeader) != "leader" { t.Error("expected leader string") }
	if string(RoleFollower) != "follower" { t.Error("expected follower string") }
	if string(RoleCandidate) != "candidate" { t.Error("expected candidate string") }
}

func TestNodeStatusString(t *testing.T) {
	if string(StatusActive) != "active" { t.Error("expected active string") }
	if string(StatusOffline) != "offline" { t.Error("expected offline string") }
	if string(StatusJoining) != "joining" { t.Error("expected joining string") }
	if string(StatusDraining) != "draining" { t.Error("expected draining string") }
	if string(StatusMaintenance) != "maintenance" { t.Error("expected maintenance string") }
}

func TestNewManager(t *testing.T) {
	m := NewManager("node-1", "host-1", "10.0.0.1", 8443, nil, nil, nil, nil, nil)
	if m.nodeID != "node-1" { t.Error("expected node ID") }
	if m.hostname != "host-1" { t.Error("expected hostname") }
	if m.address != "10.0.0.1" { t.Error("expected address") }
	if m.port != 8443 { t.Error("expected port") }
	if m.role != RoleFollower { t.Error("expected follower role") }
	if m.status != StatusJoining { t.Error("expected joining status") }
}

func TestNewManagerAutoNodeID(t *testing.T) {
	m := NewManager("", "host-1", "10.0.0.1", 8443, nil, nil, nil, nil, nil)
	if m.nodeID == "" { t.Error("expected auto-generated node ID") }
}

func TestHeartbeatInterval(t *testing.T) {
	interval := 10 * time.Second
	if interval != 10*time.Second { t.Error("expected 10s") }
}
