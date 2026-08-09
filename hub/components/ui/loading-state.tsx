"use client";

import { Loader2 } from "lucide-react";

interface LoadingStateProps {
  message?: string;
  fullPage?: boolean;
}

export function LoadingState({ message, fullPage }: LoadingStateProps) {
  if (fullPage) {
    return (
      <div role="status" aria-live="polite" className="flex flex-col items-center justify-center min-h-[60vh]">
        <Loader2 className="h-8 w-8 text-[var(--color-accent)] animate-spin mb-4" aria-hidden="true" />
        <p className="text-body text-[var(--color-text-secondary)]">
          {message || "Loading..."}
        </p>
      </div>
    );
  }

  return (
    <div role="status" aria-live="polite" className="flex items-center justify-center p-8">
      <Loader2 className="h-5 w-5 text-[var(--color-accent)] animate-spin mr-3" aria-hidden="true" />
      <p className="text-body-small text-[var(--color-text-secondary)]">
        {message || "Loading..."}
      </p>
    </div>
  );
}

interface LoadingCardProps {
  count?: number;
}

export function LoadingCards({ count = 4 }: LoadingCardProps) {
  return (
    <div aria-hidden="true" className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="skeleton h-24 rounded-lg" />
      ))}
    </div>
  );
}
