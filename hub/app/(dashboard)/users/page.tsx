"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/hooks/use-auth";
import { api, LIST_LIMIT } from "@/lib/api-client";
import { EmptyState } from "@/components/ui/empty-state";

export default function UsersPage() {
  const { can, user: currentUser } = useAuth();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["users"],
    queryFn: () => api.listUsers(LIST_LIMIT),
  });

  const [changingRole, setChangingRole] = useState<{ id: string; role: string } | null>(null);

  const roleMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string }) => api.updateUserRole(id, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      setChangingRole(null);
    },
  });

  const users = data?.users ?? [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Users</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          {isLoading ? "Loading..." : `${users.length} user${users.length !== 1 ? "s" : ""}`}
        </p>
      </div>

      {error && (
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load users</p>
        </div>
      )}

      {!isLoading && !error && users.length === 0 && (
        <EmptyState title="No users" description="No users found." />
      )}

      {users.length > 0 && (
        <div className="card overflow-hidden">
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-[var(--color-border)]">
                <th scope="col" className="text-caption text-[var(--color-text-tertiary)] px-4 py-2 font-medium">Name</th>
                <th scope="col" className="text-caption text-[var(--color-text-tertiary)] px-4 py-2 font-medium">Email</th>
                <th scope="col" className="text-caption text-[var(--color-text-tertiary)] px-4 py-2 font-medium">Role</th>
                <th scope="col" className="text-caption text-[var(--color-text-tertiary)] px-4 py-2 font-medium">Created</th>
                {can("manage_roles") && <th scope="col" className="text-caption text-[var(--color-text-tertiary)] px-4 py-2 font-medium">Actions</th>}
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-surface)] transition-colors">
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">
                    {u.display_name}
                    {u.id === currentUser?.id && (
                      <span className="text-caption text-[var(--color-text-tertiary)] ml-2">(you)</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{u.email}</td>
                  <td className="px-4 py-3">
                    <span className={`text-caption font-medium px-2 py-0.5 rounded-sm ${
                      u.role === "admin" ? "text-[var(--color-accent)] bg-[var(--color-accent-tint)]" :
                      u.role === "operator" ? "text-[var(--color-success)] bg-[var(--color-success)]/10" :
                      "text-[var(--color-text-tertiary)] bg-[var(--color-surface)]"
                    }`}>
                      {u.role}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">
                    {new Date(u.created_at).toLocaleDateString()}
                  </td>
                  {can("manage_roles") && (
                    <td className="px-4 py-3">
                      {u.id === currentUser?.id ? (
                        <span className="text-caption text-[var(--color-text-tertiary)]">Cannot self-assign</span>
                      ) : changingRole?.id === u.id ? (
                        <div className="flex items-center gap-2">
                          <select
                            value={changingRole.role}
                            onChange={(e) => setChangingRole({ id: u.id, role: e.target.value })}
                            className="input text-caption py-0.5 px-1.5"
                          >
                            <option value="admin">admin</option>
                            <option value="operator">operator</option>
                            <option value="viewer">viewer</option>
                          </select>
                          <button
                            onClick={() => roleMutation.mutate({ id: u.id, role: changingRole.role })}
                            disabled={roleMutation.isPending}
                            className="btn-ghost text-caption px-2 py-0.5 text-[var(--color-success)]"
                          >
                            {roleMutation.isPending ? "..." : "Save"}
                          </button>
                          <button
                            onClick={() => setChangingRole(null)}
                            className="btn-ghost text-caption px-2 py-0.5"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setChangingRole({ id: u.id, role: u.role })}
                          className="btn-ghost text-caption px-2 py-0.5"
                        >
                          Change Role
                        </button>
                      )}
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
