"use client";

import { cn } from "@/lib/utils";

type BadgeTone = "neutral" | "success" | "warning" | "error" | "info" | "accent";

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: BadgeTone;
  dot?: boolean;
}

const toneClasses: Record<BadgeTone, string> = {
  neutral: "bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)]",
  success: "bg-[var(--color-success-tint)] text-[var(--color-success)]",
  warning: "bg-[var(--color-warning-tint)] text-[var(--color-warning)]",
  error: "bg-[var(--color-error-tint)] text-[var(--color-error)]",
  info: "bg-[var(--color-info-tint)] text-[var(--color-info)]",
  accent: "bg-[var(--color-accent-tint)] text-[var(--color-accent)]",
};

const dotColors: Record<BadgeTone, string> = {
  neutral: "var(--color-text-tertiary)",
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  error: "var(--color-error)",
  info: "var(--color-info)",
  accent: "var(--color-accent)",
};

export function Badge({ tone = "neutral", dot, className, children, ...rest }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-[var(--radius-sm)] px-2 py-0.5 text-caption font-medium",
        toneClasses[tone],
        className,
      )}
      {...rest}
    >
      {dot && (
        <span
          className="inline-block w-1.5 h-1.5 rounded-full"
          style={{ backgroundColor: dotColors[tone] }}
          aria-hidden="true"
        />
      )}
      {children}
    </span>
  );
}
