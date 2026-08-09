"use client";

import { useVersion } from "@/hooks/use-version";
import { useDevices } from "@/hooks/use-devices";
import { LoadingState } from "@/components/ui/loading-state";
import { ErrorState } from "@/components/ui/error-state";
import { Server, Shield, Clock, Database } from "lucide-react";

export default function SettingsPage() {
  const { data: version, isLoading: versionLoading, error: versionError, refetch: refetchVersion } = useVersion();
  const { data: devicesData, isLoading: devicesLoading } = useDevices();
  const devices = devicesData?.devices ?? [];
  const onlineCount = devices.filter((d) => d.status === "online").length;

  return (
    <div className="space-y-6 max-w-4xl">
      <div>
        <h1 className="text-headline-2 text-[var(--color-text-primary)]">Settings</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          System information and configuration
        </p>
      </div>

      {/* System Info */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2">
          <Server className="h-5 w-5 text-[var(--color-accent)]" />
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">System</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <InfoRow
            label="Version"
            value={versionLoading ? "Loading..." : version?.version || "Unknown"}
          />
          <InfoRow
            label="Commit"
            value={versionLoading ? "Loading..." : version?.commit || "Unknown"}
            mono
          />
          <InfoRow
            label="Build Time"
            value={versionLoading ? "Loading..." : version?.build_time || "Unknown"}
          />
          <InfoRow
            label="Devices"
            value={devicesLoading ? "Loading..." : `${onlineCount} online / ${devices.length} total`}
          />
        </div>

        {versionError && (
          <ErrorState
            error={versionError as Error}
            onRetry={() => refetchVersion()}
            title="Could not load version info"
          />
        )}
      </div>

      {/* Security */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-[var(--color-success)]" />
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">Security</h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <InfoRow label="Authentication" value="JWT" />
          <InfoRow label="Password Hashing" value="bcrypt" />
          <InfoRow label="TLS Termination" value="Reverse Proxy" />
          <InfoRow label="Rate Limiting" value="Enabled" />
        </div>
      </div>

      {/* Maintenance */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2">
          <Database className="h-5 w-5 text-[var(--color-warning)]" />
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">Maintenance</h2>
        </div>

        <div className="space-y-3">
          <p className="text-body text-[var(--color-text-secondary)]">
            Database backups can be created using the CLI or API.
          </p>
          <div className="flex gap-3">
            <button className="px-4 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] text-[var(--color-text-primary)] text-body-small font-medium hover:bg-[var(--color-surface-hover)] transition-colors">
              Create Backup
            </button>
          </div>
        </div>
      </div>

      {/* About */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2">
          <Clock className="h-5 w-5 text-[var(--color-text-tertiary)]" />
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">About</h2>
        </div>

        <div className="space-y-2 text-body text-[var(--color-text-secondary)]">
          <p>
            <strong>RenOS</strong> is an AI-powered operating ecosystem that unifies
            desktops, servers, Raspberry Pis, NAS devices, Docker containers, and VMs
            into one platform.
          </p>
          <p className="text-caption text-[var(--color-text-tertiary)]">
            Licensed under MIT. Built with Go, Rust, and Next.js.
          </p>
        </div>
      </div>
    </div>
  );
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex justify-between items-center py-2 border-b border-[var(--color-border)]">
      <span className="text-body-small text-[var(--color-text-tertiary)]">{label}</span>
      <span className={`text-body-small text-[var(--color-text-primary)] ${mono ? "font-mono text-caption" : ""}`}>
        {value}
      </span>
    </div>
  );
}
