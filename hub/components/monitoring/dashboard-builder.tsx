"use client";

import { useMemo, useState } from "react";
import { ChevronLeft, Pencil, Plus, Save, X } from "lucide-react";
import {
  useMonitoringDashboard,
  useUpdateMonitoringDashboard,
} from "@/hooks/use-monitoring";
import type { MonitoringWidget } from "@/lib/types";
import { nextFreePosition } from "@/lib/dashboard-layout";
import {
  WIDGET_PALETTE,
  PaletteIcon,
} from "@/components/monitoring/widget-renderer";
import { WidgetGrid } from "@/components/monitoring/widget-grid";
import { ErrorState } from "@/components/ui/error-state";
import { cn } from "@/lib/utils";

interface DashboardBuilderProps {
  dashboardId: string;
  onBack: () => void;
}

export function DashboardBuilder({
  dashboardId,
  onBack,
}: DashboardBuilderProps) {
  const {
    data: dashboard,
    isLoading,
    error,
  } = useMonitoringDashboard(dashboardId);
  const update = useUpdateMonitoringDashboard(dashboardId);
  const [editMode, setEditMode] = useState(false);
  const [draft, setDraft] = useState<MonitoringWidget[] | null>(null);
  const [savedFlash, setSavedFlash] = useState(false);

  const widgets = useMemo(
    () => draft ?? dashboard?.widgets ?? [],
    [dashboard?.widgets, draft],
  );
  const widgetData = dashboard?.widget_data ?? {};
  const canEdit = update.isPending === false;

  const missing = useMemo(
    () =>
      WIDGET_PALETTE.filter(
        (p) => !widgets.some((w) => w.widget_type === p.type),
      ),
    [widgets],
  );

  if (isLoading) {
    return (
      <div className="py-12 text-center text-body-small text-[var(--color-text-tertiary)]">
        Loading dashboard…
      </div>
    );
  }
  if (error || !dashboard) {
    return (
      <ErrorState
        error={error as Error}
        onRetry={onBack}
        title="Could not load dashboard"
      />
    );
  }

  const addWidget = (type: string) => {
    const entry = WIDGET_PALETTE.find((p) => p.type === type);
    if (!entry) return;
    const pos = nextFreePosition(widgets, entry.defaultW, entry.defaultH);
    setDraft([
      ...widgets,
      {
        id: `local-${Date.now()}`,
        dashboard_id: dashboardId,
        widget_type: entry.type,
        title: entry.label,
        config: {},
        position_x: pos.position_x,
        position_y: pos.position_y,
        width: entry.defaultW,
        height: entry.defaultH,
        created_at: new Date().toISOString(),
      },
    ]);
  };

  const removeWidget = (id: string) =>
    setDraft(widgets.filter((w) => w.id !== id));

  const save = () => {
    update.mutate(
      { widgets: widgets.map(({ id: _id, created_at: _c, ...w }) => w) },
      {
        onSuccess: () => {
          setDraft(null);
          setEditMode(false);
          setSavedFlash(true);
          setTimeout(() => setSavedFlash(false), 2000);
        },
      },
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <button
          onClick={onBack}
          className="btn-ghost flex items-center gap-1 text-caption"
          aria-label="Back to dashboards"
        >
          <ChevronLeft size={16} aria-hidden="true" />
          Dashboards
        </button>
        <h2 className="text-headline-2 text-[var(--color-text-primary)]">
          {dashboard.name}
        </h2>
        {dashboard.description && (
          <p className="text-caption text-[var(--color-text-tertiary)]">
            {dashboard.description}
          </p>
        )}
        <div className="ml-auto flex items-center gap-2">
          {savedFlash && (
            <span className="text-caption text-[var(--color-success)]">
              Saved
            </span>
          )}
          {editMode ? (
            <>
              <button
                onClick={() => setDraft(null)}
                className="btn-ghost flex items-center gap-1.5 text-caption"
              >
                <X size={14} aria-hidden="true" />
                Cancel
              </button>
              <button
                onClick={save}
                disabled={!canEdit}
                className="btn-primary flex items-center gap-1.5 text-caption"
              >
                <Save size={14} aria-hidden="true" />
                Save layout
              </button>
            </>
          ) : (
            <button
              onClick={() => setEditMode(true)}
              className="btn-ghost flex items-center gap-1.5 text-caption"
            >
              <Pencil size={14} aria-hidden="true" />
              Edit
            </button>
          )}
        </div>
      </div>

      {editMode && (
        <div className="rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)] p-3">
          <div className="flex items-center gap-1.5 mb-2 text-caption text-[var(--color-text-secondary)]">
            <Plus size={14} aria-hidden="true" />
            Add widget
          </div>
          <div className="flex flex-wrap gap-1.5">
            {missing.map((p) => (
              <button
                key={p.type}
                onClick={() => addWidget(p.type)}
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md border border-[var(--color-border)] bg-[var(--color-bg-primary)] text-caption text-[var(--color-text-secondary)] hover:border-[var(--color-accent)] hover:text-[var(--color-accent)] transition-colors"
                title={p.description}
              >
                <PaletteIcon
                  type={p.type}
                  className="text-[var(--color-text-tertiary)]"
                />
                {p.label}
              </button>
            ))}
          </div>
          {missing.length === 0 && (
            <p className="text-caption text-[var(--color-text-tertiary)]">
              All widget types are on the dashboard.
            </p>
          )}
        </div>
      )}

      {widgets.length === 0 ? (
        <div className="py-16 text-center">
          <p className="text-body text-[var(--color-text-primary)]">
            No widgets yet
          </p>
          <p className="text-caption text-[var(--color-text-tertiary)] mt-1">
            {editMode
              ? "Add widgets from the palette above."
              : "Click Edit to start building."}
          </p>
        </div>
      ) : (
        <WidgetGrid
          widgets={widgets}
          widgetData={widgetData}
          editMode={editMode}
          onWidgetsChange={editMode ? setDraft : undefined}
          onRemoveWidget={editMode ? removeWidget : undefined}
        />
      )}
    </div>
  );
}
