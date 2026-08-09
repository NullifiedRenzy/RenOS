"use client";

export interface EmptyStateAction {
  label: string;
  onClick: () => void;
}

interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: string;
  action?: EmptyStateAction | React.ReactNode;
}

export function EmptyState({ title, description, action, icon }: EmptyStateProps) {
  return (
    <div
      className={`flex flex-col items-center justify-center text-center ${
        icon ? "py-16 px-4" : "p-12"
      }`}
    >
      {icon ? (
        <span className="text-5xl mb-4">{icon}</span>
      ) : (
        <div className="h-16 w-16 rounded-full bg-[var(--color-surface)] border border-[var(--color-border)] flex items-center justify-center mb-4">
          <svg
            className="h-8 w-8 text-[var(--color-text-tertiary)]"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={1.5}
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
            />
          </svg>
        </div>
      )}
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-1">{title}</h3>
      {description && (
        <p
          className={`text-body text-[var(--color-text-secondary)] ${
            icon ? "mt-2 max-w-md" : "max-w-sm mb-4"
          }`}
        >
          {description}
        </p>
      )}
      {typeof action === "object" && action !== null && "label" in action ? (
        <button onClick={action.onClick} className="btn-primary mt-6">
          {action.label}
        </button>
      ) : (
        action
      )}
    </div>
  );
}
