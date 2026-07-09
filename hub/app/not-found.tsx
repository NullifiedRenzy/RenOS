import Link from "next/link";

export default function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)] px-4">
      <div className="text-center">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">
          404
        </h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-2">
          This page could not be found.
        </p>
        <Link
          href="/dashboard"
          className="btn-primary mt-6 inline-flex"
        >
          Go to Dashboard
        </Link>
      </div>
    </div>
  );
}
