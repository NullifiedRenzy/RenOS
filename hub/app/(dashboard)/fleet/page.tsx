"use client";

import { useState } from "react";
import { useAuth } from "@/hooks/use-auth";
import {
  useFleetDashboard,
  useDeviceInventoryList,
  useComplianceResults,
  useComplianceSummary,
  useComplianceChecks,
  useBulkOperations,
  useCancelBulkOperation,
  useTagKeys,
} from "@/hooks/use-fleet";
import type { DeviceInventory, ComplianceCheck, ComplianceResult, BulkOperation } from "@/lib/types";
import { formatUptime } from "@/lib/format";
import { LoadingState } from "@/components/ui/loading-state";
import { ErrorState } from "@/components/ui/error-state";
import { EmptyState } from "@/components/ui/empty-state";

type FleetTab = "dashboard" | "inventory" | "compliance" | "bulk" | "tags";

const statusColors: Record<string, string> = {
  pass: "var(--color-success)",
  fail: "var(--color-error)",
  skip: "var(--color-text-tertiary)",
  pending: "var(--color-warning)",
};

const severityColors: Record<string, string> = {
  info: "var(--color-info)",
  warning: "var(--color-warning)",
  critical: "var(--color-error)",
};

export default function FleetPage() {
  const { can } = useAuth();
  const [activeTab, setActiveTab] = useState<FleetTab>("dashboard");

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Fleet Management</h1>
      </div>

      <div className="flex gap-2 border-b border-[var(--color-border)] pb-0">
        {(["dashboard", "inventory", "compliance", "bulk", "tags"] as FleetTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-body-small rounded-t-lg border border-b-0 transition-colors ${
              activeTab === tab
                ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] border-[var(--color-border)] font-medium"
                : "bg-transparent text-[var(--color-text-tertiary)] border-transparent hover:text-[var(--color-text-secondary)]"
            }`}
          >
            {tab === "dashboard" && "Overview"}
            {tab === "inventory" && "Inventory"}
            {tab === "compliance" && "Compliance"}
            {tab === "bulk" && "Bulk Operations"}
            {tab === "tags" && "Tags"}
          </button>
        ))}
      </div>

      <div className="mt-6">
        {activeTab === "dashboard" && <DashboardTab />}
        {activeTab === "inventory" && <InventoryTab />}
        {activeTab === "compliance" && <ComplianceTab />}
        {activeTab === "bulk" && <BulkTab />}
        {activeTab === "tags" && <TagsTab />}
      </div>
    </div>
  );
}

function DashboardTab() {
  const { data: dash, isLoading, error } = useFleetDashboard();
  const { data: complianceSummary } = useComplianceSummary();

  if (isLoading) return <LoadingState message="Loading fleet dashboard..." />;
  if (error) return <ErrorState error={error} title="Failed to load fleet dashboard" />;
  if (!dash) return <EmptyState title="No fleet data available" />;

  const statCards = [
    { label: "Total Devices", value: dash.total_devices, color: "var(--color-info)" },
    { label: "Online", value: dash.online_devices, color: "var(--color-success)" },
    { label: "Offline", value: dash.offline_devices, color: "var(--color-error)" },
    { label: "Groups", value: dash.total_groups, color: "var(--color-info)" },
    { label: "Tags", value: dash.total_tags, color: "var(--color-info)" },
    { label: "Inventory", value: dash.inventory_count, color: "var(--color-info)" },
  ];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        {statCards.map((stat) => (
          <div key={stat.label} className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4">
            <div className="text-caption text-[var(--color-text-tertiary)]">{stat.label}</div>
            <div className="text-headline-2 mt-1" style={{ color: stat.color }}>{stat.value}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
          <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">OS Distribution</h3>
          {Object.keys(dash.os_distribution).length === 0 ? (
            <p className="text-body-small text-[var(--color-text-tertiary)]">No data available</p>
          ) : (
            <div className="space-y-2">
              {(Object.entries(dash.os_distribution) as [string, number][]).sort(([, a], [, b]) => b - a).map(([os, count]) => (
                <div key={os} className="flex items-center justify-between">
                  <span className="text-body-small text-[var(--color-text-secondary)]">{os}</span>
                  <div className="flex items-center gap-2">
                    <div className="h-2 bg-[var(--color-info)] rounded-full" style={{ width: `${(count / dash.total_devices) * 100}px` }} />
                    <span className="text-caption text-[var(--color-text-tertiary)]">{count}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
          <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">Compliance Status</h3>
          {complianceSummary ? (
            <div className="space-y-3">
              {(Object.entries(complianceSummary) as [string, number][]).map(([status, count]) => (
                <div key={status} className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-3 h-3 rounded-full" style={{ backgroundColor: statusColors[status] || "var(--color-text-tertiary)" }} />
                    <span className="text-body-small text-[var(--color-text-secondary)] capitalize">{status}</span>
                  </div>
                  <span className="text-body-small font-medium text-[var(--color-text-primary)]">{count}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-body-small text-[var(--color-text-tertiary)]">No compliance data</p>
          )}
        </div>
      </div>
    </div>
  );
}

function InventoryTab() {
  const { data: inventory, isLoading, error } = useDeviceInventoryList();

  if (isLoading) return <LoadingState message="Loading inventory..." />;
  if (error) return <ErrorState error={error} title="Failed to load inventory" />;

  const list = inventory ?? [];
  if (list.length === 0) return <EmptyState title="No device inventory collected yet" />;

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
      <table className="w-full">
        <thead>
          <tr className="border-b border-[var(--color-border)]">
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Device</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">CPU</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">RAM</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Storage</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">GPU</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">OS</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Uptime</th>
          </tr>
        </thead>
        <tbody>
          {list.map((inv: DeviceInventory) => (
            <tr key={inv.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
              <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{inv.device_id.slice(0, 8)}</td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{inv.cpu_model || "N/A"}</td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">
                {inv.ram_total_mb > 0 ? `${(inv.ram_total_mb / 1024).toFixed(1)} GB` : "N/A"}
              </td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">
                {inv.storage_total_gb > 0 ? `${inv.storage_total_gb.toFixed(1)} GB` : "N/A"}
              </td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{inv.gpu_model || "None"}</td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{inv.os_family || "N/A"}</td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">
                {inv.uptime_seconds > 0 ? formatUptime(inv.uptime_seconds, { alwaysHours: true }) : "N/A"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ComplianceTab() {
  const { data: checks } = useComplianceChecks();
  const { data: results } = useComplianceResults();
  const { data: summary } = useComplianceSummary();

  return (
    <div className="space-y-6">
      {summary && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {Object.entries(summary).map(([status, count]) => (
            <div key={status} className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4">
              <div className="text-caption text-[var(--color-text-tertiary)] capitalize">{status}</div>
              <div className="text-headline-2 mt-1" style={{ color: statusColors[status] || "var(--color-text-primary)" }}>{count}</div>
            </div>
          ))}
        </div>
      )}

      {checks && checks.length > 0 && (
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
          <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">Compliance Checks</h3>
          <div className="space-y-2">
            {checks.map((check: ComplianceCheck) => (
              <div key={check.id} className="flex items-center justify-between p-3 bg-[var(--color-bg-tertiary)] rounded-lg">
                <div>
                  <div className="text-body-small text-[var(--color-text-primary)]">{check.name}</div>
                  <div className="text-caption text-[var(--color-text-tertiary)]">{check.description}</div>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-caption px-2 py-1 rounded" style={{ backgroundColor: severityColors[check.severity] + "20", color: severityColors[check.severity] }}>
                    {check.severity}
                  </span>
                  <span className="text-caption text-[var(--color-text-tertiary)]">{check.check_type}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {results && results.length > 0 && (
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-[var(--color-border)]">
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Check</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Device</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Status</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Message</th>
              </tr>
            </thead>
            <tbody>
              {results.map((res: ComplianceResult) => (
                <tr key={res.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{res.check_id.slice(0, 8)}</td>
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{res.device_id.slice(0, 8)}</td>
                  <td className="px-4 py-3">
                    <span className="text-caption px-2 py-1 rounded capitalize" style={{ backgroundColor: (statusColors[res.status] || "var(--color-text-tertiary)") + "20", color: statusColors[res.status] || "var(--color-text-tertiary)" }}>
                      {res.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{res.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function BulkTab() {
  const { data: ops, isLoading } = useBulkOperations();
  const cancelOp = useCancelBulkOperation();

  return (
    <div className="space-y-4">
      {isLoading ? (
        <LoadingState message="Loading operations..." />
      ) : !ops || ops.length === 0 ? (
        <EmptyState title="No bulk operations yet" />
      ) : (
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-[var(--color-border)]">
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Action</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Targets</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Progress</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Status</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Actions</th>
              </tr>
            </thead>
            <tbody>
              {ops.map((op: BulkOperation) => (
                <tr key={op.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{op.action}</td>
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{op.total_count}</td>
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">
                    {op.completed_count}/{op.total_count}
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-caption px-2 py-1 rounded capitalize" style={{ backgroundColor: statusColors[op.status] ? statusColors[op.status] + "20" : "var(--color-bg-tertiary)", color: statusColors[op.status] || "var(--color-text-tertiary)" }}>
                      {op.status}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {op.status === "running" && (
                      <button onClick={() => cancelOp.mutate(op.id)} className="text-caption text-[var(--color-error)] hover:underline">
                        Cancel
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function TagsTab() {
  const { data: tagKeys, isLoading } = useTagKeys();

  return (
    <div className="space-y-4">
      {isLoading ? (
        <LoadingState message="Loading tags..." />
      ) : !tagKeys || tagKeys.length === 0 ? (
        <EmptyState title="No tags defined yet" />
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
          {tagKeys.map((key: string) => (
            <div key={key} className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4">
              <div className="text-body-small text-[var(--color-text-primary)] font-medium">{key}</div>
              <div className="text-caption text-[var(--color-text-tertiary)] mt-1">Tag key</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
