"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/hooks/use-auth";
import { api, ApiRequestError } from "@/lib/api-client";

const scheduleTypes = [
  { value: "command", label: "Command" },
  { value: "container_action", label: "Container Action" },
  { value: "service_action", label: "Service Action" },
  { value: "update", label: "Update" },
];

const targetTypes = [
  { value: "device", label: "Device" },
  { value: "group", label: "Group" },
  { value: "multi", label: "Multi" },
];

export default function NewSchedulePage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { can } = useAuth();

  const [name, setName] = useState("");
  const [type, setType] = useState("command");
  const [targetType, setTargetType] = useState("device");
  const [targetIds, setTargetIds] = useState("");
  const [cronExpression, setCronExpression] = useState("");
  const [payloadJson, setPayloadJson] = useState("{}");
  const [error, setError] = useState("");

  const createMutation = useMutation({
    mutationFn: (data: Parameters<typeof api.createSchedule>[0]) => api.createSchedule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
      router.push("/schedules");
    },
    onError: (err: Error) => {
      setError(err instanceof ApiRequestError ? err.message : "Failed to create schedule");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!name.trim()) { setError("Name is required"); return; }
    if (!cronExpression.trim()) { setError("Cron expression is required"); return; }
    if (!targetIds.trim()) { setError("At least one target ID is required"); return; }

    let parsedPayload: Record<string, unknown>;
    try {
      parsedPayload = JSON.parse(payloadJson);
    } catch {
      setError("Invalid JSON in payload");
      return;
    }

    const ids = targetIds.split(",").map((s) => s.trim()).filter(Boolean);

    createMutation.mutate({
      name: name.trim(),
      type,
      payload: parsedPayload,
      target_type: targetType,
      target_ids: ids,
      cron_expression: cronExpression.trim(),
    });
  };

  if (!can("write")) {
    return (
      <div className="space-y-6 max-w-2xl">
        <div>
          <Link href="/schedules" className="btn-ghost text-body-small">&larr; All Schedules</Link>
        </div>
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Insufficient permissions. You need operator or admin role to create schedules.</p>
        </div>
      </div>
    );
  }

  const isHighFreq = (() => {
    // Warn if cron runs more than once per minute
    try {
      const parts = cronExpression.trim().split(/\s+/);
      if (parts.length >= 2) {
        const minute = parts[0];
        if (minute === "*" || minute === "*/1" || minute === "*/2" || minute === "*/3" || minute === "*/4" || minute === "*/5") return true;
        if (/^\d+$/.test(minute) || minute.includes(",")) return false;
        if (minute.startsWith("*/") && parseInt(minute.slice(2)) <= 5) return true;
      }
    } catch { }
    return false;
  })();

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <Link href="/schedules" className="btn-ghost text-body-small">&larr; All Schedules</Link>
      </div>

      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">New Schedule</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          Create a recurring automation
        </p>
      </div>

      <form onSubmit={handleSubmit} className="card p-6 space-y-5">
        {error && (
          <div className="text-body-small text-[var(--color-error)] bg-[var(--color-error)]/10 p-3 rounded-sm">
            {error}
          </div>
        )}

        <div className="space-y-1">
          <label className="text-body-small font-medium text-[var(--color-text-primary)]">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input w-full"
            placeholder="e.g. Nightly system update"
            maxLength={255}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1">
            <label className="text-body-small font-medium text-[var(--color-text-primary)]">Action Type</label>
            <select
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="input w-full"
            >
              {scheduleTypes.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <label className="text-body-small font-medium text-[var(--color-text-primary)]">Target Type</label>
            <select
              value={targetType}
              onChange={(e) => setTargetType(e.target.value)}
              className="input w-full"
            >
              {targetTypes.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </div>
        </div>

        <div className="space-y-1">
          <label className="text-body-small font-medium text-[var(--color-text-primary)]">
            Target IDs
          </label>
          <input
            type="text"
            value={targetIds}
            onChange={(e) => setTargetIds(e.target.value)}
            className="input w-full"
            placeholder="Comma-separated device or group IDs"
          />
          <p className="text-caption text-[var(--color-text-tertiary)]">
            For group target type, use the group ID
          </p>
        </div>

        <div className="space-y-1">
          <label className="text-body-small font-medium text-[var(--color-text-primary)]">
            Cron Expression
          </label>
          <input
            type="text"
            value={cronExpression}
            onChange={(e) => setCronExpression(e.target.value)}
            className="input w-full font-mono"
            placeholder="e.g. 0 3 * * * (daily at 3am)"
          />
          {isHighFreq && (
            <p className="text-caption text-[var(--color-warning)] mt-1">
              Warning: This cron expression runs frequently (&le;5 min). Use with caution.
            </p>
          )}
        </div>

        <div className="space-y-1">
          <label className="text-body-small font-medium text-[var(--color-text-primary)]">
            Payload (JSON)
          </label>
          <textarea
            value={payloadJson}
            onChange={(e) => setPayloadJson(e.target.value)}
            className="input w-full font-mono min-h-[120px]"
            placeholder='{"command": "apt update && apt upgrade -y"}'
          />
          <p className="text-caption text-[var(--color-text-tertiary)]">
            Depends on action type: command text, container_id + action, unit_name + action, etc.
          </p>
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="btn-primary text-body-small px-4 py-2"
          >
            {createMutation.isPending ? "Creating..." : "Create Schedule"}
          </button>
          <Link href="/schedules" className="btn-ghost text-body-small px-4 py-2">
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}
