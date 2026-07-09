export function RecentActivity() {
  const activities = [
    { time: "10:32", event: "desktop-01 updated docker containers", type: "success" },
    { time: "09:15", event: "nas-01 backup completed", type: "success" },
    { time: "08:00", event: "pi-04 went offline", type: "error" },
    { time: "07:45", event: "Workflow 'Deploy' completed", type: "success" },
    { time: "06:30", event: "server-01 health check passed", type: "success" },
  ];

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-4">
        Recent Activity
      </h2>
      <div className="space-y-3">
        {activities.map((a, i) => (
          <div key={i} className="flex items-start gap-3">
            <span className="text-caption text-[var(--color-text-tertiary)] w-12 flex-shrink-0">
              {a.time}
            </span>
            <span
              className={`inline-block w-1.5 h-1.5 rounded-full mt-1.5 flex-shrink-0 ${
                a.type === "success"
                  ? "bg-[var(--color-success)]"
                  : "bg-[var(--color-error)]"
              }`}
            />
            <span className="text-body-small text-[var(--color-text-secondary)]">
              {a.event}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
