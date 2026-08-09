"use client";

import { useDialogEscape } from "@/hooks/use-dialog";

interface ActionOutputModalProps {
  title: string;
  output: string;
  error: string;
  exitCode: number | null;
  onClose: () => void;
  size?: "md" | "lg";
}

export function ActionOutputModal({ title, output, error, exitCode, onClose, size = "md" }: ActionOutputModalProps) {
  const isError = exitCode !== 0;
  const dialogRef = useDialogEscape(onClose, true);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        className={`bg-[var(--color-surface)] rounded-md shadow-lg ${
          size === "lg" ? "max-w-3xl" : "max-w-2xl"
        } w-full mx-4 max-h-[80vh] flex flex-col outline-none`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
          <h3 className="text-body font-medium text-[var(--color-text-primary)]">{title}</h3>
          <button className="btn-ghost p-1 text-body-small" onClick={onClose}>
            &#10005;
          </button>
        </div>
        <pre className="p-4 overflow-auto text-body-small font-mono text-[var(--color-text-primary)] whitespace-pre-wrap flex-1">
          {isError && error ? error : output || "(empty output)"}
        </pre>
        {exitCode !== null && (
          <div className="px-4 pb-3 text-caption text-[var(--color-text-tertiary)]">
            Exit code: {exitCode}
          </div>
        )}
      </div>
    </div>
  );
}
