"use client";

import { useRequestCommand } from "@/hooks/use-commands";

const SAFE_COMMANDS = [
  { label: "whoami", command: "whoami", description: "Current user" },
  { label: "hostname", command: "hostname", description: "Machine hostname" },
  { label: "uptime", command: "uptime", description: "System uptime" },
  { label: "df -h", command: "df -h", description: "Disk usage" },
  { label: "free -h", command: "free -h", description: "Memory usage" },
];

interface SafeCommandsProps {
  deviceId: string;
}

export function SafeCommands({ deviceId }: SafeCommandsProps) {
  const request = useRequestCommand(deviceId);

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Safe Commands</h2>
      <p className="text-body-small text-[var(--color-text-tertiary)] mb-4">
        Run a read-only command on this device. Results appear below.
      </p>
      <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
        {SAFE_COMMANDS.map(({ label, command, description }) => (
          <button
            key={command}
            className="btn-secondary text-left flex-col items-start gap-0 h-auto py-2 px-3"
            disabled={request.isPending}
            onClick={() => request.mutate(command)}
          >
            <span className="text-body-small font-medium text-[var(--color-text-primary)]">{label}</span>
            <span className="text-caption text-[var(--color-text-tertiary)]">{description}</span>
          </button>
        ))}
      </div>
      {request.isPending && (
        <p className="text-body-small text-[var(--color-text-tertiary)] mt-3">
          Requesting command...
        </p>
      )}
      {request.isError && (
        <p className="text-body-small text-[var(--color-error)] mt-3">
          {(request.error as Error).message}
        </p>
      )}
    </div>
  );
}
