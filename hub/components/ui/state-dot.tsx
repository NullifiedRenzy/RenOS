const STATE_COLORS: Record<string, string> = {
  running: "var(--color-success)",
  exited: "var(--color-text-tertiary)",
  paused: "var(--color-warning)",
  restarting: "var(--color-info)",
  removing: "var(--color-warning)",
  dead: "var(--color-error)",
  created: "var(--color-info)",
};

export function stateDotColor(state: string): string {
  return STATE_COLORS[state.toLowerCase()] ?? "var(--color-text-tertiary)";
}

export function StateDot({ state }: { state: string }) {
  return (
    <span
      className="inline-block w-2 h-2 rounded-full flex-shrink-0"
      style={{ backgroundColor: stateDotColor(state) }}
    />
  );
}
