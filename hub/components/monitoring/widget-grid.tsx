"use client";

import { useCallback, useRef, useState } from "react";
import { GripHorizontal, Scaling, Trash2 } from "lucide-react";
import type { MonitoringWidget, WidgetData } from "@/lib/types";
import { GRID_COLS, gridColumnStyle, gridRowStyle, moveWidget, resizeWidget } from "@/lib/dashboard-layout";
import { WidgetRenderer } from "@/components/monitoring/widget-renderer";
import { cn } from "@/lib/utils";

interface WidgetGridProps {
  widgets: MonitoringWidget[];
  widgetData?: Record<string, WidgetData>;
  editMode?: boolean;
  onWidgetsChange?: (widgets: MonitoringWidget[]) => void;
  onRemoveWidget?: (id: string) => void;
}

export function WidgetGrid({ widgets, widgetData, editMode, onWidgetsChange, onRemoveWidget }: WidgetGridProps) {
  const [drag, setDrag] = useState<{ id: string; type: "move" | "resize"; startX: number; startY: number; orig: MonitoringWidget } | null>(null);
  const cellRef = useRef<HTMLDivElement>(null);

  const getGridCell = useCallback((clientX: number, clientY: number) => {
    const el = cellRef.current;
    if (!el) return { dx: 0, dy: 0 };
    const rect = el.getBoundingClientRect();
    const x = Math.floor(((clientX - rect.left) / rect.width) * GRID_COLS);
    const y = Math.floor((clientY - rect.top) / 36);
    return { dx: x, dy: y };
  }, []);

  const onPointerDown = (e: React.PointerEvent, widget: MonitoringWidget, type: "move" | "resize") => {
    if (!editMode) return;
    e.preventDefault();
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    setDrag({ id: widget.id, type, startX: e.clientX, startY: e.clientY, orig: widget });
  };

  const onPointerMove = (e: React.PointerEvent) => {
    if (!drag) return;
    const { dx, dy } = getGridCell(e.clientX, e.clientY);
    onWidgetsChange?.(
      widgets.map((w) =>
        w.id === drag.id
          ? drag.type === "move"
            ? moveWidget({ ...drag.orig, position_x: drag.orig.position_x, position_y: drag.orig.position_y }, dx, dy)
            : resizeWidget({ ...drag.orig, width: drag.orig.width, height: drag.orig.height }, dx, dy)
          : w,
      ),
    );
  };

  const onPointerUp = () => setDrag(null);

  return (
    <div
      ref={cellRef}
      className={cn("grid gap-3", editMode ? "min-h-[300px]" : "min-h-[120px]")}
      style={{ gridTemplateColumns: `repeat(${GRID_COLS}, minmax(0, 1fr))`, gridAutoRows: "36px" }}
    >
      {widgets.map((w) => {
        const isDragging = drag?.id === w.id;
        return (
          <div
            key={w.id}
            onPointerMove={onPointerMove}
            onPointerUp={onPointerUp}
            onPointerCancel={onPointerUp}
            className={cn(
              "relative flex flex-col rounded-lg border border-[var(--color-border)] bg-[var(--color-bg-secondary)]",
              editMode && "group",
              isDragging && "ring-2 ring-[var(--color-accent)]",
            )}
            style={{ gridColumn: gridColumnStyle(w), gridRow: gridRowStyle(w), touchAction: "none" }}
          >
            <div
              className={cn(
                "flex items-center gap-2 px-3 h-8 border-b border-[var(--color-border)] rounded-t-lg",
                editMode ? "cursor-move select-none" : "",
              )}
              onPointerDown={editMode ? (e) => onPointerDown(e, w, "move") : undefined}
              role={editMode ? "button" : undefined}
              aria-label={editMode ? `Move ${w.title}` : undefined}
            >
              {editMode && <GripHorizontal size={14} className="text-[var(--color-text-tertiary)] shrink-0" aria-hidden="true" />}
              <span className="text-caption font-medium text-[var(--color-text-primary)] truncate">{w.title}</span>
              {editMode && (
                <button
                  onClick={() => onRemoveWidget?.(w.id)}
                  className="ml-auto p-1 rounded text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]"
                  aria-label={`Remove ${w.title}`}
                >
                  <Trash2 size={13} />
                </button>
              )}
            </div>
            <div className="flex-1 overflow-hidden p-3">
              <WidgetRenderer widget={w} data={widgetData?.[w.id]} />
            </div>
            {editMode && (
              <button
                className="absolute bottom-1 right-1 w-4 h-4 rounded-sm bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)] hidden items-center justify-center group-hover:flex"
                onPointerDown={(e) => onPointerDown(e, w, "resize")}
                aria-label={`Resize ${w.title}`}
              >
                <Scaling size={11} />
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}
