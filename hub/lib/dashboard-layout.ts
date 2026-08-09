import type { MonitoringWidget } from "@/lib/types";

export const GRID_COLS = 12;
export const MIN_W = 2;
export const MIN_H = 2;

export function clamp(v: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, v));
}

export function moveWidget(w: MonitoringWidget, dx: number, dy: number): MonitoringWidget {
  return {
    ...w,
    position_x: clamp(w.position_x + dx, 0, GRID_COLS - w.width),
    position_y: Math.max(0, w.position_y + dy),
  };
}

export function resizeWidget(
  w: MonitoringWidget,
  dw: number,
  dh: number,
): MonitoringWidget {
  const width = clamp(w.width + dw, MIN_W, GRID_COLS);
  const height = clamp(w.height + dh, MIN_H, 8);
  return {
    ...w,
    width,
    height,
    position_x: clamp(w.position_x, 0, GRID_COLS - width),
  };
}

// nextFreePosition scans the grid row-by-row for a free slot for a widget of the
// given size, honoring already-occupied cells.
export function nextFreePosition(
  widgets: MonitoringWidget[],
  width: number,
  height: number,
): { position_x: number; position_y: number } {
  const occupied = new Set<string>();
  for (const w of widgets) {
    for (let y = w.position_y; y < w.position_y + w.height; y++) {
      for (let x = w.position_x; x < w.position_x + w.width; x++) {
        occupied.add(`${x},${y}`);
      }
    }
  }
  const w = clamp(width, MIN_W, GRID_COLS);
  const h = clamp(height, MIN_H, 8);
  for (let y = 0; y < 64; y++) {
    for (let x = 0; x <= GRID_COLS - w; x++) {
      let free = true;
      for (let yy = y; yy < y + h; yy++) {
        for (let xx = x; xx < x + w; xx++) {
          if (occupied.has(`${xx},${yy}`)) {
            free = false;
            break;
          }
        }
        if (!free) break;
      }
      if (free) return { position_x: x, position_y: y };
    }
  }
  return { position_x: 0, position_y: 64 };
}

export function gridColumnStyle(w: { position_x: number; width: number }): string {
  return `${w.position_x + 1} / span ${w.width}`;
}

export function gridRowStyle(w: { position_y: number; height: number }): string {
  return `${w.position_y + 1} / span ${w.height}`;
}
