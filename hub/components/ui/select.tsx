"use client";

import { forwardRef, useId } from "react";
import { ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface SelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, "children"> {
  label?: string;
  error?: string;
  hint?: string;
  options: SelectOption[];
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { label, error, hint, options, className, id, ...rest },
  ref,
) {
  const autoId = useId();
  const selectId = id ?? autoId;
  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      {label && (
        <label htmlFor={selectId} className="text-body-small font-medium text-[var(--color-text-secondary)]">
          {label}
        </label>
      )}
      <div className="relative">
        <select
          ref={ref}
          id={selectId}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${selectId}-error` : hint ? `${selectId}-hint` : undefined}
          className={cn(
            "w-full h-10 px-3 pr-8 text-body appearance-none rounded-[var(--radius-sm)]",
            "bg-[var(--color-surface)] border border-[var(--color-border)] text-[var(--color-text-primary)]",
            "transition-colors duration-[var(--duration-fast)] ease-out",
            "focus:outline-none focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-focus-ring)]",
            error && "border-[var(--color-error)] focus:border-[var(--color-error)] focus:ring-[var(--color-error-tint)]",
            rest.disabled && "opacity-50 cursor-not-allowed",
          )}
          {...rest}
        >
          {options.map((o) => (
            <option key={o.value} value={o.value} disabled={o.disabled}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronDown
          size={16}
          className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-text-tertiary)]"
          aria-hidden="true"
        />
      </div>
      {error && (
        <p id={`${selectId}-error`} className="text-body-small text-[var(--color-error)]" role="alert">
          {error}
        </p>
      )}
      {!error && hint && (
        <p id={`${selectId}-hint`} className="text-body-small text-[var(--color-text-tertiary)]">
          {hint}
        </p>
      )}
    </div>
  );
});
