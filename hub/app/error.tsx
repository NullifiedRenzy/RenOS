"use client";

import { getErrorMessage } from "@/lib/error-messages";

export default function ErrorPage({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const copy = getErrorMessage(error);

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)] px-4">
      <div role="alert" className="text-center">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">
          {copy.title}
        </h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-2">
          {copy.message}
        </p>
        {copy.hint && (
          <p className="text-body-small text-[var(--color-text-tertiary)] mt-2">
            {copy.hint}
          </p>
        )}
        <button
          onClick={reset}
          className="btn-primary mt-6"
        >
          Try again
        </button>
      </div>
    </div>
  );
}
