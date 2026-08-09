import { describe, expect, it } from "vitest";
import {
  GRID_COLS,
  clamp,
  gridColumnStyle,
  gridRowStyle,
  moveWidget,
  nextFreePosition,
  resizeWidget,
} from "@/lib/dashboard-layout";
import type { MonitoringWidget } from "@/lib/types";

function widget(partial: Partial<MonitoringWidget> = {}): MonitoringWidget {
  return {
    id: "w1",
    dashboard_id: "d1",
    widget_type: "cpu",
    title: "CPU",
    config: {},
    position_x: 0,
    position_y: 0,
    width: 6,
    height: 3,
    created_at: "2026-01-01T00:00:00Z",
    ...partial,
  };
}

describe("dashboard-layout", () => {
  it("clamps values to range", () => {
    expect(clamp(5, 0, 10)).toBe(5);
    expect(clamp(-1, 0, 10)).toBe(0);
    expect(clamp(11, 0, 10)).toBe(10);
  });

  it("moves a widget and clamps to grid columns", () => {
    const w = widget({ position_x: 5, width: 4 });
    const moved = moveWidget(w, 3, 2);
    expect(moved.position_x).toBe(8);
    expect(moved.position_y).toBe(2);

    const pushed = moveWidget(widget({ position_x: 9, width: 4 }), 2, 0);
    expect(pushed.position_x).toBe(GRID_COLS - 4);
  });

  it("resizes a widget within min/max bounds", () => {
    const resized = resizeWidget(widget({ width: 6, height: 3 }), 3, 2);
    expect(resized.width).toBe(9);
    expect(resized.height).toBe(5);

    const maxed = resizeWidget(widget({ width: 11, height: 7 }), 2, 2);
    expect(maxed.width).toBe(GRID_COLS);
    expect(maxed.height).toBe(8);

    const minned = resizeWidget(widget(), -10, -10);
    expect(minned.width).toBe(2);
    expect(minned.height).toBe(2);
  });

  it("finds the next free position honoring occupancy", () => {
    const occupied = [widget({ id: "a", position_x: 0, position_y: 0, width: 6, height: 3 })];
    const pos = nextFreePosition(occupied, 4, 2);
    expect(pos.position_x).toBe(6);
    expect(pos.position_y).toBe(0);

    const wrap = nextFreePosition(
      [widget({ id: "a", position_x: 0, position_y: 0, width: GRID_COLS, height: 4 })],
      2,
      2,
    );
    expect(wrap.position_y).toBe(4);
    expect(wrap.position_x).toBe(0);
  });

  it("renders grid column/row style strings", () => {
    expect(gridColumnStyle({ position_x: 2, width: 4 })).toBe("3 / span 4");
    expect(gridRowStyle({ position_y: 1, height: 3 })).toBe("2 / span 3");
  });
});
