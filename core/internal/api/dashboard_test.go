package api

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/renos-project/renos/core/internal/db"
)

var errBoom = errors.New("boom")

type fakeWidgetProvider struct {
	responses map[string]json.RawMessage
	errors    map[string]error
}

func (f *fakeWidgetProvider) Hydrate(_ context.Context, w *db.MonitoringWidget) (json.RawMessage, error) {
	if err := f.errors[w.ID]; err != nil {
		return nil, err
	}
	if d, ok := f.responses[w.ID]; ok {
		return d, nil
	}
	return json.RawMessage(`{}`), nil
}

func TestHydrateDashboardData(t *testing.T) {
	provider := &fakeWidgetProvider{
		responses: map[string]json.RawMessage{
			"w1": json.RawMessage(`{"total":3,"online":2}`),
			"w2": json.RawMessage(`{"firing":1}`),
		},
		errors: map[string]error{
			"w3": errBoom,
		},
	}
	widgets := []*db.MonitoringWidget{
		{ID: "w1", WidgetType: "devices-overview"},
		{ID: "w2", WidgetType: "alerts"},
		{ID: "w3", WidgetType: "cpu"}, // provider fails -> omitted
	}

	data := hydrateDashboardData(context.Background(), provider, widgets)

	if len(data) != 2 {
		t.Fatalf("len(data) = %d, want 2 (failed widget omitted)", len(data))
	}
	if _, ok := data["w3"]; ok {
		t.Error("failed widget w3 must be omitted from data")
	}
	if got := string(data["w1"]); got != `{"total":3,"online":2}` {
		t.Errorf("w1 data = %s", got)
	}
}

func TestHydrateDashboardDataNilProvider(t *testing.T) {
	data := hydrateDashboardData(context.Background(), nil, []*db.MonitoringWidget{
		{ID: "w1", WidgetType: "cpu"},
	})
	if len(data) != 0 {
		t.Errorf("nil provider must return empty data, got %v", data)
	}
}

func TestHydrateDashboardDataNilWidgets(t *testing.T) {
	provider := &fakeWidgetProvider{}
	data := hydrateDashboardData(context.Background(), provider, nil)
	if len(data) != 0 {
		t.Errorf("nil widgets must produce empty data, got %v", data)
	}
}
