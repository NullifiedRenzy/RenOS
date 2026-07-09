interface EmptyStateProps {
  icon?: string;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
}

export function EmptyState({ icon = "\u{1F4BB}", title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4">
      <span className="text-5xl mb-4">{icon}</span>
      <h3 className="text-headline-3 text-[var(--color-text-primary)] text-center">{title}</h3>
      {description && (
        <p className="text-body text-[var(--color-text-secondary)] mt-2 text-center max-w-md">
          {description}
        </p>
      )}
      {action && (
        <button onClick={action.onClick} className="btn-primary mt-6">
          {action.label}
        </button>
      )}
    </div>
  );
}
