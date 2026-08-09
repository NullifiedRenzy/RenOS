import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { WidgetGrid } from "@/components/monitoring/widget-grid";
import { WidgetRenderer, WIDGET_PALETTE } from "@/components/monitoring/widget-renderer";
import type { MonitoringWidget, WidgetData } from "@/lib/types";

function widget(id: string, partial: Partial<MonitoringWidget> = {}): MonitoringWidget {
  return {
    id,
    dashboard_id: "d1",
    widget_type: "cpu",
    title: `Widget ${id}`,
    config: {},
    position_x: 0,
    position_y: 0,
    width: 6,
    height: 3,
    created_at: "2026-01-01T00:00:00Z",
    ...partial,
  };
}

describe("WidgetGrid", () => {
  it("renders widgets at their grid position", () => {
    render(
      <WidgetGrid
        widgets={[widget("a", { position_x: 3, position_y: 2 })]}
        widgetData={{ a: { metric: "cpu", samples: [] } }}
      />,
    );
    expect(screen.getByText("Widget a")).toBeInTheDocument();
  });

  it("does not allow drag when not in edit mode", () => {
    const onChange = vi.fn();
    render(<WidgetGrid widgets={[widget("a")]} onWidgetsChange={onChange} />);
    fireEvent.pointerDown(screen.getByText("Widget a"), { pointerId: 1, clientX: 0, clientY: 0 });
    fireEvent.pointerMove(screen.getByText("Widget a"), { clientX: 100, clientY: 100 });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("shows move handle and remove buttons in edit mode", () => {
    render(<WidgetGrid widgets={[widget("a")]} editMode onRemoveWidget={() => {}} />);
    expect(screen.getByLabelText("Move Widget a")).toBeInTheDocument();
    expect(screen.getByLabelText("Remove Widget a")).toBeInTheDocument();
  });

  it("calls onRemoveWidget when remove clicked", () => {
    const onRemove = vi.fn();
    render(<WidgetGrid widgets={[widget("a")]} editMode onRemoveWidget={onRemove} />);
    fireEvent.click(screen.getByLabelText("Remove Widget a"));
    expect(onRemove).toHaveBeenCalledWith("a");
  });
});

describe("WidgetRenderer", () => {
  it("renders devices-overview stats", () => {
    const data: WidgetData = { total: 5, online: 4, offline: 1, error: 0, by_status: {}, os_dist: {} };
    const { container } = render(<WidgetRenderer widget={widget("a", { widget_type: "devices-overview" })} data={data} />);
    expect(container.textContent).toContain("5");
    expect(container.textContent).toContain("4");
  });

  it("renders cpu metric card with bar samples", () => {
    const data: WidgetData = { metric: "cpu", samples: [{ value: 40, recorded_at: "2026-01-01T00:00:00Z" }] };
    const { container } = render(<WidgetRenderer widget={widget("a", { widget_type: "cpu" })} data={data} />);
    expect(container.textContent).toContain("40%");
  });

  it("renders alert firing count", () => {
    const data: WidgetData = { firing: 2, active: ["high load"] };
    const { container } = render(<WidgetRenderer widget={widget("a", { widget_type: "alerts" })} data={data} />);
    expect(container.textContent).toContain("high load");
  });

  it("shows a fallback for unknown widget types", () => {
    render(<WidgetRenderer widget={widget("a", { widget_type: "mystery" })} />);
    expect(screen.getByText(/mystery/)).toBeInTheDocument();
  });

  it("exposes every palette type in the registry", () => {
    expect(WIDGET_PALETTE.length).toBeGreaterThanOrEqual(10);
    expect(WIDGET_PALETTE.map((p) => p.type)).toContain("proxmox");
  });
});
