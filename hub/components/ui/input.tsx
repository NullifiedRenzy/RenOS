"use client";

import { forwardRef, useId } from "react";
import { cn } from "@/lib/utils";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, error, hint, className, id, ...rest },
  ref,
) {
  const autoId = useId();
  const inputId = id ?? autoId;
  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      {label && (
        <label htmlFor={inputId} className="text-body-small font-medium text-[var(--color-text-secondary)]">
          {label}
        </label>
      )}
      <input
        ref={ref}
        id={inputId}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${inputId}-error` : hint ? `${inputId}-hint` : undefined}
        className={cn(
          "w-full h-10 px-3 text-body rounded-[var(--radius-sm)]",
          "bg-[var(--color-surface)] border border-[var(--color-border)]",
          "placeholder:text-[var(--color-text-tertiary)]",
          "transition-colors duration-[var(--duration-fast)] ease-out",
          "focus:outline-none focus:border-[var(--color-accent)] focus:ring-2 focus:ring-[var(--color-focus-ring)]",
          error && "border-[var(--color-error)] focus:border-[var(--color-error)] focus:ring-[var(--color-error-tint)]",
          rest.disabled && "opacity-50 cursor-not-allowed",
        )}
        {...rest}
      />
      {error && (
        <p id={`${inputId}-error`} className="text-body-small text-[var(--color-error)]" role="alert">
          {error}
        </p>
      )}
      {!error && hint && (
        <p id={`${inputId}-hint`} className="text-body-small text-[var(--color-text-tertiary)]">
          {hint}
        </p>
      )}
    </div>
  );
});
