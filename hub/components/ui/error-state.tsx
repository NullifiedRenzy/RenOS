"use client";

import { AlertTriangle, RefreshCw } from "lucide-react";
import { ApiRequestError } from "@/lib/api-client";
import { getErrorMessage } from "@/lib/error-messages";

interface ErrorStateProps {
  error: Error | ApiRequestError;
  onRetry?: () => void;
  title?: string;
  hint?: string;
}

export function ErrorState({ error, onRetry, title, hint }: ErrorStateProps) {
  const copy = getErrorMessage(error);
  const isApiError = error instanceof ApiRequestError;
  const code = isApiError ? error.code : undefined;

  return (
    <div role="alert" className="flex flex-col items-center justify-center p-8 text-center">
      <AlertTriangle className="h-10 w-10 text-[var(--color-error)] mb-3" aria-hidden="true" />
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-1">
        {title || copy.title}
      </h3>
      <p className="text-body text-[var(--color-text-secondary)] mb-1 max-w-sm">
        {copy.message}
      </p>
      {hint || copy.hint ? (
        <p className="text-body-small text-[var(--color-text-tertiary)] max-w-sm mb-3">
          {hint || copy.hint}
        </p>
      ) : null}
      {code && (
        <p className="text-caption text-[var(--color-text-tertiary)] mb-3 font-mono">
          Error code: {code}
        </p>
      )}
      {onRetry && (
        <button
          onClick={onRetry}
          className="flex items-center gap-2 px-4 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] text-body-small font-medium hover:bg-[var(--color-surface-hover)] transition-colors"
        >
          <RefreshCw className="h-4 w-4" aria-hidden="true" />
          Try again
        </button>
      )}
    </div>
  );
}
