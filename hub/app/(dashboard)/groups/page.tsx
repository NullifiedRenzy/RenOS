"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { api } from "@/lib/api-client";
import { EmptyState } from "@/components/ui/empty-state";

export default function GroupsPage() {
  const queryClient = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["groups"],
    queryFn: () => api.listGroups(),
    refetchInterval: 30_000,
  });

  const createMutation = useMutation({
    mutationFn: () => api.createGroup({ name, description }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["groups"] });
      setShowCreate(false);
      setName("");
      setDescription("");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteGroup(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["groups"] }),
  });

  const groups = data?.groups ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">Groups</h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {isLoading ? "Loading..." : `${groups.length} group${groups.length !== 1 ? "s" : ""}`}
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="btn-primary text-body-small px-3 py-1.5"
        >
          + New Group
        </button>
      </div>

      {showCreate && (
        <div className="card p-4 space-y-3">
          <input
            type="text"
            placeholder="Group name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input"
          />
          <textarea
            placeholder="Description (optional)"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="input min-h-[60px]"
          />
          <div className="flex gap-2">
            <button
              onClick={() => createMutation.mutate()}
              disabled={!name || createMutation.isPending}
              className="btn-primary text-body-small px-3 py-1.5"
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </button>
            <button
              onClick={() => setShowCreate(false)}
              className="btn-ghost text-body-small px-3 py-1.5"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {error && (
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load groups</p>
        </div>
      )}

      {!isLoading && !error && groups.length === 0 && (
        <EmptyState
          title="No groups created"
          description="Create a group to organize your devices and run bulk commands."
        />
      )}

      {groups.length > 0 && (
        <div className="space-y-2">
          {groups.map((group) => (
            <div key={group.id} className="card p-4 flex items-center justify-between">
              <Link href={`/groups/${group.id}`} className="flex-1 min-w-0">
                <p className="text-body font-medium text-[var(--color-text-primary)] truncate">
                  {group.name}
                </p>
                <p className="text-body-small text-[var(--color-text-tertiary)] truncate">
                  {group.description || "No description"} &middot; {group.device_count} device{group.device_count !== 1 ? "s" : ""}
                </p>
              </Link>
              <button
                onClick={() => {
                  if (confirm("Delete this group? Devices will not be affected.")) {
                    deleteMutation.mutate(group.id);
                  }
                }}
                className="btn-ghost text-body-small text-[var(--color-error)] px-2 py-1"
              >
                Delete
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
