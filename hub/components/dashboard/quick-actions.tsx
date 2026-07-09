const actions = [
  { label: "Run Backup", icon: "💾" },
  { label: "Update All", icon: "🔄" },
  { label: "Deploy App", icon: "🚀" },
  { label: "Create VM", icon: "🖥" },
  { label: "Ask AI", icon: "✨" },
];

export function QuickActions() {
  return (
    <div className="card">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">
          Quick Actions
        </h2>
        <button className="btn-ghost text-body-small">See all →</button>
      </div>
      <div className="flex gap-3">
        {actions.map((action) => (
          <button
            key={action.label}
            className="btn-secondary flex-col gap-1 h-auto py-3 px-4 min-w-[100px]"
          >
            <span className="text-xl">{action.icon}</span>
            <span className="text-caption">{action.label}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
