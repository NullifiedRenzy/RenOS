"use client";

import React from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { getErrorMessage } from "@/lib/error-messages";

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends React.Component<
  { children: React.ReactNode; fallback?: React.ReactNode },
  ErrorBoundaryState
> {
  constructor(props: { children: React.ReactNode; fallback?: React.ReactNode }) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("ErrorBoundary caught:", error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback;
      const copy = getErrorMessage(this.state.error ?? undefined);
      return (
        <div role="alert" className="flex flex-col items-center justify-center p-12 text-center">
          <AlertTriangle className="h-12 w-12 text-[var(--color-error)] mb-4" aria-hidden="true" />
          <h2 className="text-headline-3 text-[var(--color-text-primary)] mb-2">
            {copy.title}
          </h2>
          <p className="text-body text-[var(--color-text-secondary)] mb-4 max-w-md">
            {copy.message}
          </p>
          {copy.hint && (
            <p className="text-body-small text-[var(--color-text-tertiary)] max-w-md mb-4">
              {copy.hint}
            </p>
          )}
          <button
            onClick={() => {
              this.setState({ hasError: false, error: null });
              window.location.reload();
            }}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[var(--color-accent)] text-white text-body-small font-medium hover:opacity-90 transition-opacity"
          >
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            Reload page
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
