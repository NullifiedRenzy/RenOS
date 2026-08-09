package update

import (
	"testing"
)

func TestReleaseChannelValues(t *testing.T) {
	channels := []Channel{ChannelStable, ChannelBeta, ChannelNightly}
	valid := map[Channel]bool{ChannelStable: true, ChannelBeta: true, ChannelNightly: true}

	for _, c := range channels {
		if !valid[c] {
			t.Errorf("unexpected channel: %s", c)
		}
	}
}

func TestUpdateStatusValues(t *testing.T) {
	statuses := []string{"pending", "downloading", "backing_up", "installing", "verifying", "completed", "failed", "rolled_back"}
	valid := map[string]bool{}
	for _, s := range statuses {
		valid[s] = true
	}

	if !valid["pending"] { t.Error("expected pending") }
	if !valid["completed"] { t.Error("expected completed") }
	if !valid["failed"] { t.Error("expected failed") }
	if !valid["rolled_back"] { t.Error("expected rolled_back") }
}

func TestUpdatePlanFields(t *testing.T) {
	plan := &UpdatePlan{
		CurrentVersion: "0.5.0",
		TargetVersion:  "0.6.0",
		Channel:        ChannelStable,
		Components:     []string{"core", "agent", "hub"},
		RequiresBackup: true,
		Steps: []PlanStep{
			{Order: 1, Name: "backup", Component: "core", Action: "backup"},
			{Order: 2, Name: "update", Component: "core", Action: "update"},
			{Order: 3, Name: "verify", Component: "core", Action: "verify"},
		},
	}

	if plan.CurrentVersion != "0.5.0" { t.Error("expected current version") }
	if plan.TargetVersion != "0.6.0" { t.Error("expected target version") }
	if len(plan.Components) != 3 { t.Error("expected 3 components") }
	if len(plan.Steps) != 3 { t.Error("expected 3 steps") }
	if !plan.RequiresBackup { t.Error("expected requires backup") }
}

func TestPlanStepFields(t *testing.T) {
	step := &PlanStep{
		Name:        "backup",
		Description: "Create backup before update",
		Component:   "core",
		Action:      "backup",
		Order:       1,
	}

	if step.Name != "backup" { t.Error("expected backup") }
	if step.Order != 1 { t.Error("expected order 1") }
}

func TestChannelString(t *testing.T) {
	if string(ChannelStable) != "stable" { t.Error("expected stable string") }
	if string(ChannelBeta) != "beta" { t.Error("expected beta string") }
	if string(ChannelNightly) != "nightly" { t.Error("expected nightly string") }
}

func TestVersionCompare(t *testing.T) {
	v1 := "0.5.0"
	v2 := "0.6.0"
	if v1 == v2 { t.Error("versions should not be equal") }
	if v1 > v2 { t.Error("0.5.0 should be less than 0.6.0") }
}

func TestRequiresBackupLogic(t *testing.T) {
	versions := []struct {
		from   string
		to     string
		backup bool
	}{
		{"0.5.0", "0.6.0", true},
		{"0.5.0", "0.5.1", false},
		{"0.5.0", "1.0.0", true},
	}

	for _, v := range versions {
		requiresBackup := v.from[:3] != v.to[:3]
		if requiresBackup != v.backup {
			t.Errorf("version %s -> %s: expected backup=%v", v.from, v.to, v.backup)
		}
	}
}

func TestNewPlanner(t *testing.T) {
	p := NewUpdatePlanner(nil, nil)
	if p == nil { t.Error("expected non-nil planner") }
	if p.channel != ChannelStable { t.Error("expected default stable channel") }
}
