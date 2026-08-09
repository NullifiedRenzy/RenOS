"use client";

import { cn } from "@/lib/utils";

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  interactive?: boolean;
}

export function Card({ interactive, className, ...rest }: CardProps) {
  return (
    <div
      className={cn(
        "rounded-[var(--radius-md)] border border-[var(--color-border)] bg-[var(--color-surface-elevated)]",
        "p-[var(--space-4)] shadow-[var(--shadow-sm)]",
        interactive &&
          "cursor-pointer transition-shadow duration-[var(--duration-fast)] ease-out hover:shadow-[var(--shadow-md)] hover:border-[var(--color-accent)]",
        className,
      )}
      {...rest}
    />
  );
}

export function CardHeader({ title, description, action, className }: { title: React.ReactNode; description?: React.ReactNode; action?: React.ReactNode; className?: string }) {
  return (
    <div className={cn("flex items-start justify-between gap-4 mb-[var(--space-3)]", className)}>
      <div>
        <h3 className="text-headline-4 text-[var(--color-text-primary)]">{title}</h3>
        {description && <p className="text-body-small text-[var(--color-text-tertiary)] mt-0.5">{description}</p>}
      </div>
      {action}
    </div>
  );
}

export function CardBody({ className, ...rest }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("text-body text-[var(--color-text-secondary)]", className)} {...rest} />;
}

export function CardFooter({ className, ...rest }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex items-center gap-2 mt-[var(--space-4)] pt-[var(--space-3)] border-t border-[var(--color-border)]", className)} {...rest} />;
}
