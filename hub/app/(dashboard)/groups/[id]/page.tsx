"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { EmptyState } from "@/components/ui/empty-state";
import { BulkActionProgress } from "@/components/groups/BulkActionProgress";
import { useGroup } from "@/hooks/use-groups";

export default function GroupDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useGroup(id);

  const { data: devicesData } = useQuery({
    queryKey: ["devices"],
    queryFn: () => api.listDevices(),
  });

  const group = data?.group ?? null;
  const deviceIds = data?.device_ids ?? [];
  const devices = devicesData?.devices ?? [];

  const addMutation = useMutation({
    mutationFn: (deviceId: string) => api.addDeviceToGroup(id, deviceId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["group", id] }),
  });

  const removeMutation = useMutation({
    mutationFn: (deviceId: string) => api.removeDeviceFromGroup(id, deviceId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["group", id] }),
  });

  const [command, setCommand] = useState("");
  const [activeJobIds, setActiveJobIds] = useState<string[]>([]);

  const runCommandMutation = useMutation({
    mutationFn: (cmd: string) => api.runGroupCommand(id, cmd),
    onSuccess: (data) => {
      setCommand("");
      setActiveJobIds((prev) => [...prev, data.job.id]);
    },
  });

  const [availableDeviceId, setAvailableDeviceId] = useState("");

  const devicesInGroup = devices.filter((d) => deviceIds.includes(d.id));
  const devicesNotInGroup = devices.filter((d) => !deviceIds.includes(d.id));

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-8 w-48" />
        <div className="skeleton h-32" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <button onClick={() => router.back()} className="btn-ghost text-body-small">&larr; Back</button>
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load group</p>
        </div>
      </div>
    );
  }

  if (!group) {
    return (
      <EmptyState
        title="Group not found"
        description="The group you are looking for does not exist."
        action={{ label: "Back to Groups", onClick: () => router.push("/groups") }}
      />
    );
  }

  return (
    <div className="space-y-6 max-w-4xl">
      <div>
        <Link href="/groups" className="btn-ghost text-body-small">&larr; All Groups</Link>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">{group.name}</h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {group.description || "No description"} &middot; {deviceIds.length} device{deviceIds.length !== 1 ? "s" : ""}
          </p>
        </div>
      </div>

      {activeJobIds.length > 0 && (
        <div className="space-y-2">
          {activeJobIds.map((jobId) => (
            <BulkActionProgress
              key={jobId}
              jobId={jobId}
            />
          ))}
        </div>
      )}

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Run Command</h2>
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="Enter command..."
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            className="input flex-1"
            onKeyDown={(e) => {
              if (e.key === "Enter" && command && deviceIds.length > 0) {
                runCommandMutation.mutate(command);
              }
            }}
          />
          <button
            onClick={() => runCommandMutation.mutate(command)}
            disabled={!command || deviceIds.length === 0 || runCommandMutation.isPending}
            className="btn-primary text-body-small px-3 py-1.5"
          >
            {runCommandMutation.isPending ? "Running..." : `Run on ${deviceIds.length} device${deviceIds.length !== 1 ? "s" : ""}`}
          </button>
        </div>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Devices in Group</h2>

        {devicesInGroup.length === 0 && (
          <p className="text-body text-[var(--color-text-tertiary)] py-2">No devices in this group</p>
        )}

        {devicesInGroup.length > 0 && (
          <div className="space-y-2">
            {devicesInGroup.map((d) => (
              <div key={d.id} className="flex items-center justify-between py-1">
                <div className="flex items-center gap-2 min-w-0">
                  <span
                    className="inline-block w-2 h-2 rounded-full flex-shrink-0"
                    style={{
                      backgroundColor: d.status === "online" ? "var(--color-success)" : "var(--color-text-tertiary)",
                    }}
                  />
                  <Link href={`/devices/${d.id}`} className="text-body text-[var(--color-accent)] truncate hover:underline">
                    {d.hostname}
                  </Link>
                  <span className="text-caption text-[var(--color-text-tertiary)]">{d.status}</span>
                </div>
                <button
                  onClick={() => removeMutation.mutate(d.id)}
                  className="btn-ghost text-caption text-[var(--color-error)] px-2 py-0.5"
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}

        {devicesNotInGroup.length > 0 && (
          <div className="flex gap-2 pt-2 border-t border-[var(--color-border)]">
            <select
              value={availableDeviceId}
              onChange={(e) => setAvailableDeviceId(e.target.value)}
              className="input flex-1"
            >
              <option value="">Select a device to add...</option>
              {devicesNotInGroup.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.hostname} ({d.status})
                </option>
              ))}
            </select>
            <button
              onClick={() => {
                if (availableDeviceId) {
                  addMutation.mutate(availableDeviceId);
                  setAvailableDeviceId("");
                }
              }}
              disabled={!availableDeviceId}
              className="btn-primary text-body-small px-3 py-1.5"
            >
              Add
            </button>
          </div>
        )}
      </div>

    </div>
  );
}
