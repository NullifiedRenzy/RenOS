"use client";

export default function ErrorPage({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)] px-4">
      <div className="text-center">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">
          Something went wrong
        </h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-2">
          {error.message || "An unexpected error occurred."}
        </p>
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
