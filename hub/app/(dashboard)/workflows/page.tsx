"use client";

import { useState } from "react";
import { useAuth } from "@/hooks/use-auth";
import {
  useWorkflows,
  useWorkflowSteps,
  useWorkflowRuns,
  useAllWorkflowRuns,
  useWorkflowTemplates,
  useCreateWorkflow,
  useDeleteWorkflow,
  useToggleWorkflow,
  useExecuteWorkflow,
  useCreateWorkflowFromTemplate,
} from "@/hooks/use-workflows";
import type { Workflow, WorkflowStep, WorkflowRun } from "@/lib/types";
import { ErrorState } from "@/components/ui/error-state";
import { ScreenReaderOnly } from "@/components/ui/screen-reader-only";

type WorkflowTab = "workflows" | "runs" | "templates";

const statusColors: Record<string, string> = {
  completed: "var(--color-success)",
  running: "var(--color-info)",
  pending: "var(--color-warning)",
  failed: "var(--color-error)",
  cancelled: "var(--color-text-tertiary)",
  paused: "var(--color-warning)",
};

const triggerLabels: Record<string, string> = {
  manual: "Manual",
  schedule: "Schedule",
  event: "Event",
  webhook: "Webhook",
  condition: "Condition",
};

const stepTypeColors: Record<string, string> = {
  action: "var(--color-info)",
  condition: "var(--color-warning)",
  delay: "var(--color-text-tertiary)",
  parallel: "var(--color-success)",
  branch: "var(--color-error)",
};

export default function WorkflowsPage() {
  const { can } = useAuth();
  const [activeTab, setActiveTab] = useState<WorkflowTab>("workflows");
  const [selectedWorkflow, setSelectedWorkflow] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Automation Engine</h1>
        {can("operator") && (
          <button
            onClick={() => setShowCreateForm(true)}
            className="px-4 py-2 bg-[var(--color-accent)] text-[var(--color-text-on-accent)] rounded-lg text-body-small hover:opacity-90"
          >
            + New Workflow
          </button>
        )}
      </div>

      <div className="flex gap-2 border-b border-[var(--color-border)] pb-0">
        {(["workflows", "runs", "templates"] as WorkflowTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => { setActiveTab(tab); setSelectedWorkflow(null); }}
            className={`px-4 py-2 text-body-small rounded-t-lg border border-b-0 transition-colors ${
              activeTab === tab
                ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] border-[var(--color-border)] font-medium"
                : "bg-transparent text-[var(--color-text-tertiary)] border-transparent hover:text-[var(--color-text-secondary)]"
            }`}
          >
            {tab === "workflows" && "Workflows"}
            {tab === "runs" && "Recent Runs"}
            {tab === "templates" && "Templates"}
          </button>
        ))}
      </div>

      <div className="mt-6">
        {activeTab === "workflows" && !selectedWorkflow && (
          <WorkflowsListTab onShowDetail={setSelectedWorkflow} showCreate={showCreateForm} onCreateClose={() => setShowCreateForm(false)} />
        )}
        {activeTab === "workflows" && selectedWorkflow && (
          <WorkflowDetailTab id={selectedWorkflow} onBack={() => setSelectedWorkflow(null)} />
        )}
        {activeTab === "runs" && <RunsTab />}
        {activeTab === "templates" && <TemplatesTab />}
      </div>
    </div>
  );
}

function WorkflowsListTab({ onShowDetail, showCreate, onCreateClose }: { onShowDetail: (id: string) => void; showCreate: boolean; onCreateClose: () => void }) {
  const { data: workflows, isLoading, error } = useWorkflows();
  const deleteWorkflow = useDeleteWorkflow();
  const toggleWorkflow = useToggleWorkflow();
  const executeWorkflow = useExecuteWorkflow();
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  if (isLoading) return <Loading message="Loading workflows..." />;

  const list = workflows ?? [];

  return (
    <div className="space-y-4">
      {showCreate && <CreateWorkflowForm onClose={onCreateClose} />}

      {error && !isLoading && <ErrorState error={error} title="Failed to load workflows" />}

      {list.length === 0 ? (
        <EmptyState message="No workflows yet. Create one to get started." />
      ) : (
        <div className="space-y-3">
          {list.map((wf: Workflow) => (
            <div key={wf.id} className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4 hover:border-[var(--color-accent)] transition-colors">
              <div className="flex items-center justify-between">
                <button
                  type="button"
                  onClick={() => onShowDetail(wf.id)}
                  className="flex-1 text-left cursor-pointer"
                >
                  <div className="flex items-center gap-3">
                    <span className="w-3 h-3 rounded-full" aria-hidden="true" style={{ backgroundColor: wf.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }} />
                    <ScreenReaderOnly>{wf.enabled ? "Enabled" : "Disabled"}</ScreenReaderOnly>
                    <span className="text-body text-[var(--color-text-primary)] font-medium">{wf.name}</span>
                    <span className="text-caption text-[var(--color-text-tertiary)] px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)]">
                      {triggerLabels[wf.trigger_type] || wf.trigger_type}
                    </span>
                    <span className="text-caption text-[var(--color-text-tertiary)]">v{wf.version}</span>
                  </div>
                  {wf.description && (
                    <p className="text-body-small text-[var(--color-text-tertiary)] mt-1 ml-6">{wf.description}</p>
                  )}
                </button>
                <div className="flex items-center gap-2 ml-4">
                  {wf.enabled && (
                    <button
                      onClick={() => executeWorkflow.mutate({ id: wf.id })}
                      disabled={executeWorkflow.isPending}
                      className="text-caption text-[var(--color-accent)] hover:underline disabled:opacity-50"
                    >
                      Run
                    </button>
                  )}
                  <button
                    onClick={() => toggleWorkflow.mutate({ id: wf.id, enabled: !wf.enabled })}
                    className="text-caption text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"
                  >
                    {wf.enabled ? "Disable" : "Enable"}
                  </button>
                  {confirmDelete === wf.id ? (
                    <div className="flex items-center gap-2">
                      <button onClick={() => deleteWorkflow.mutate(wf.id, { onSuccess: () => setConfirmDelete(null) })} className="text-caption text-[var(--color-error)] hover:underline">Confirm</button>
                      <button onClick={() => setConfirmDelete(null)} className="text-caption text-[var(--color-text-tertiary)] hover:underline">Cancel</button>
                    </div>
                  ) : (
                    <button onClick={() => setConfirmDelete(wf.id)} className="text-caption text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]">Delete</button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function WorkflowDetailTab({ id, onBack }: { id: string; onBack: () => void }) {
  const { data: steps, error: stepsError } = useWorkflowSteps(id);
  const { data: runs, error: runsError } = useWorkflowRuns(id);

  return (
    <div className="space-y-6">
      <button onClick={onBack} className="text-body-small text-[var(--color-accent)] hover:underline">← Back to workflows</button>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
          <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">Steps</h3>
          {stepsError ? (
            <ErrorState error={stepsError} title="Failed to load steps" />
          ) : !steps || steps.length === 0 ? (
            <p className="text-body-small text-[var(--color-text-tertiary)]">No steps defined</p>
          ) : (
            <div className="space-y-2">
              {steps.map((step: WorkflowStep) => (
                <div key={step.id} className="flex items-center gap-3 p-3 bg-[var(--color-bg-tertiary)] rounded-lg">
                  <span className="text-caption text-[var(--color-text-tertiary)] w-6">#{step.step_order + 1}</span>
                  <span className="w-2 h-2 rounded-full" style={{ backgroundColor: stepTypeColors[step.step_type] || "var(--color-text-tertiary)" }} />
                  <span className="text-body-small text-[var(--color-text-primary)]">{step.name}</span>
                  <span className="text-caption text-[var(--color-text-tertiary)] ml-auto">{step.step_type}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
          <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">Recent Runs</h3>
          {runsError ? (
            <ErrorState error={runsError} title="Failed to load runs" />
          ) : !runs || runs.length === 0 ? (
            <p className="text-body-small text-[var(--color-text-tertiary)]">No runs yet</p>
          ) : (
            <div className="space-y-2">
              {runs.slice(0, 10).map((run: WorkflowRun) => (
                <div key={run.id} className="flex items-center justify-between p-3 bg-[var(--color-bg-tertiary)] rounded-lg">
                  <div className="flex items-center gap-3">
                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: statusColors[run.status] || "var(--color-text-tertiary)" }} />
                    <span className="text-body-small text-[var(--color-text-primary)]">{run.id.slice(0, 8)}</span>
                  </div>
                  <span className="text-caption capitalize" style={{ color: statusColors[run.status] || "var(--color-text-tertiary)" }}>{run.status}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function RunsTab() {
  const { data: runs, isLoading, error } = useAllWorkflowRuns();

  if (isLoading) return <Loading message="Loading runs..." />;

  const list = runs ?? [];
  if (error) return <ErrorState error={error} title="Failed to load runs" />;
  if (list.length === 0) return <EmptyState message="No workflow runs yet" />;

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
      <table className="w-full">
        <thead>
          <tr className="border-b border-[var(--color-border)]">
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Workflow</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Trigger</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Status</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Started</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Completed</th>
          </tr>
        </thead>
        <tbody>
          {list.map((run: WorkflowRun) => (
            <tr key={run.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
              <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{run.workflow_id.slice(0, 8)}</td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{triggerLabels[run.trigger_type] || run.trigger_type}</td>
              <td className="px-4 py-3">
                <span className="text-caption px-2 py-1 rounded capitalize" style={{ backgroundColor: (statusColors[run.status] || "var(--color-text-tertiary)") + "20", color: statusColors[run.status] || "var(--color-text-tertiary)" }}>
                  {run.status}
                </span>
              </td>
              <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{run.started_at ? new Date(run.started_at).toLocaleString() : "—"}</td>
              <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{run.completed_at ? new Date(run.completed_at).toLocaleString() : "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TemplatesTab() {
  const { data: templates, isLoading, error } = useWorkflowTemplates();
  const createFromTemplate = useCreateWorkflowFromTemplate();

  if (isLoading) return <Loading message="Loading templates..." />;

  const list = templates ?? [];
  if (error) return <ErrorState error={error} title="Failed to load templates" />;
  if (list.length === 0) return <EmptyState message="No templates available" />;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {list.map((t) => (
        <div key={t.id} className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-5">
          <div className="flex items-center gap-3 mb-3">
            <span className="text-2xl">{t.icon || "⚙"}</span>
            <div>
              <div className="text-body text-[var(--color-text-primary)] font-medium">{t.name}</div>
              <div className="text-caption text-[var(--color-text-tertiary)]">{t.category}</div>
            </div>
          </div>
          {t.description && (
            <p className="text-body-small text-[var(--color-text-secondary)] mb-4 line-clamp-2">{t.description}</p>
          )}
          <button
            onClick={() => createFromTemplate.mutate(t.id)}
            disabled={createFromTemplate.isPending}
            className="w-full px-3 py-2 bg-[var(--color-accent)] text-[var(--color-text-on-accent)] rounded-lg text-body-small hover:opacity-90 disabled:opacity-50"
          >
            Use Template
          </button>
        </div>
      ))}
    </div>
  );
}

function CreateWorkflowForm({ onClose }: { onClose: () => void }) {
  const createWorkflow = useCreateWorkflow();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [triggerType, setTriggerType] = useState("manual");

  const handleSubmit = () => {
    if (!name) return;
    createWorkflow.mutate(
      { name, description, trigger_type: triggerType, enabled: true },
      { onSuccess: onClose }
    );
  };

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">New Workflow</h3>
      <div className="space-y-4">
        <input
          type="text"
          placeholder="Workflow name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]"
        />
        <textarea
          placeholder="Description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          className="w-full px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]"
        />
        <select
          value={triggerType}
          onChange={(e) => setTriggerType(e.target.value)}
          className="w-full px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]"
        >
          <option value="manual">Manual</option>
          <option value="schedule">Schedule</option>
          <option value="event">Event</option>
          <option value="webhook">Webhook</option>
          <option value="condition">Condition</option>
        </select>
        <div className="flex gap-2 justify-end">
          <button onClick={onClose} className="px-4 py-2 text-body-small text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]">Cancel</button>
          <button onClick={handleSubmit} disabled={!name || createWorkflow.isPending} className="px-4 py-2 bg-[var(--color-accent)] text-[var(--color-text-on-accent)] rounded-lg text-body-small hover:opacity-90 disabled:opacity-50">
            Create
          </button>
        </div>
      </div>
    </div>
  );
}

function Loading({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}

function EmptyState({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}
