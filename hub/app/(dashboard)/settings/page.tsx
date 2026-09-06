"use client";

import { useEffect, useState } from "react";
import { useVersion } from "@/hooks/use-version";
import { useDevices } from "@/hooks/use-devices";
import { useAuth } from "@/hooks/use-auth";
import { api } from "@/lib/api-client";
import { ErrorState } from "@/components/ui/error-state";
import { Server, Shield, Clock, Database, KeyRound } from "lucide-react";

export default function SettingsPage() {
  const {
    data: version,
    isLoading: versionLoading,
    error: versionError,
    refetch: refetchVersion,
  } = useVersion();
  const { data: devicesData, isLoading: devicesLoading } = useDevices();
  const { user } = useAuth();
  const [mfaEnabled, setMfaEnabled] = useState(Boolean(user?.totp_enabled));
  const [mfaSetup, setMfaSetup] = useState<{
    secret: string;
    otpauth_url: string;
  } | null>(null);
  const [mfaCode, setMfaCode] = useState("");
  const [mfaPassword, setMfaPassword] = useState("");
  const [mfaBusy, setMfaBusy] = useState(false);
  const [mfaMessage, setMfaMessage] = useState<string | null>(null);
  const [mfaError, setMfaError] = useState<string | null>(null);
  const devices = devicesData?.devices ?? [];
  const onlineCount = devices.filter((d) => d.status === "online").length;

  useEffect(() => {
    setMfaEnabled(Boolean(user?.totp_enabled));
  }, [user?.totp_enabled]);

  async function startMFASetup() {
    setMfaBusy(true);
    setMfaError(null);
    setMfaMessage(null);
    try {
      const setup = await api.setupMFA();
      setMfaSetup(setup);
      setMfaCode("");
    } catch (err) {
      setMfaError(err instanceof Error ? err.message : "Failed to set up MFA");
    } finally {
      setMfaBusy(false);
    }
  }

  async function enableMFA() {
    setMfaBusy(true);
    setMfaError(null);
    setMfaMessage(null);
    try {
      await api.enableMFA(mfaCode);
      setMfaEnabled(true);
      setMfaSetup(null);
      setMfaCode("");
      setMfaMessage("MFA enabled.");
    } catch (err) {
      setMfaError(err instanceof Error ? err.message : "Failed to enable MFA");
    } finally {
      setMfaBusy(false);
    }
  }

  async function disableMFA() {
    setMfaBusy(true);
    setMfaError(null);
    setMfaMessage(null);
    try {
      await api.disableMFA(mfaPassword, mfaCode);
      setMfaEnabled(false);
      setMfaPassword("");
      setMfaCode("");
      setMfaSetup(null);
      setMfaMessage("MFA disabled.");
    } catch (err) {
      setMfaError(err instanceof Error ? err.message : "Failed to disable MFA");
    } finally {
      setMfaBusy(false);
    }
  }

  return (
    <div className="space-y-6 max-w-4xl">
      <div>
        <h1 className="text-headline-2 text-[var(--color-text-primary)]">
          Settings
        </h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          System information and configuration
        </p>
      </div>

      {/* System Info */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2">
          <Server className="h-5 w-5 text-[var(--color-accent)]" />
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">
            System
          </h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <InfoRow
            label="Version"
            value={
              versionLoading ? "Loading..." : version?.version || "Unknown"
            }
          />
          <InfoRow
            label="Commit"
            value={versionLoading ? "Loading..." : version?.commit || "Unknown"}
            mono
          />
          <InfoRow
            label="Build Time"
            value={
              versionLoading ? "Loading..." : version?.build_time || "Unknown"
            }
          />
          <InfoRow
            label="Devices"
            value={
              devicesLoading
                ? "Loading..."
                : `${onlineCount} online / ${devices.length} total`
            }
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
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">
            Security
          </h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <InfoRow label="Authentication" value="JWT" />
          <InfoRow label="Password Hashing" value="bcrypt" />
          <InfoRow label="TLS Termination" value="Reverse Proxy" />
          <InfoRow label="Rate Limiting" value="Enabled" />
          <InfoRow label="MFA" value={mfaEnabled ? "Enabled" : "Disabled"} />
        </div>

        <div className="border-t border-[var(--color-border)] pt-4 space-y-4">
          <div className="flex items-center gap-2">
            <KeyRound className="h-4 w-4 text-[var(--color-accent)]" />
            <h3 className="text-body-small font-medium text-[var(--color-text-primary)]">
              Multi-factor authentication
            </h3>
          </div>

          {mfaMessage && (
            <div className="rounded-sm bg-[rgba(52,199,89,0.08)] px-3 py-2 text-body-small text-[var(--color-success)]">
              {mfaMessage}
            </div>
          )}
          {mfaError && (
            <div className="rounded-sm bg-[rgba(255,59,48,0.08)] px-3 py-2 text-body-small text-[var(--color-error)]">
              {mfaError}
            </div>
          )}

          {!mfaEnabled && !mfaSetup && (
            <button
              onClick={startMFASetup}
              className="btn-primary"
              disabled={mfaBusy}
            >
              {mfaBusy ? "Preparing..." : "Set up MFA"}
            </button>
          )}

          {!mfaEnabled && mfaSetup && (
            <div className="space-y-3">
              <div className="grid grid-cols-1 gap-2">
                <InfoRow label="Secret" value={mfaSetup.secret} mono />
                <InfoRow label="URI" value={mfaSetup.otpauth_url} mono />
              </div>
              <div className="flex flex-col sm:flex-row gap-2">
                <input
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9 ]*"
                  placeholder="Authentication code"
                  value={mfaCode}
                  onChange={(e) => setMfaCode(e.target.value)}
                  className="input sm:max-w-56"
                  autoComplete="one-time-code"
                />
                <button
                  onClick={enableMFA}
                  className="btn-primary"
                  disabled={mfaBusy || mfaCode.trim().length < 6}
                >
                  Enable MFA
                </button>
              </div>
            </div>
          )}

          {mfaEnabled && (
            <div className="flex flex-col sm:flex-row gap-2">
              <input
                type="password"
                placeholder="Password"
                value={mfaPassword}
                onChange={(e) => setMfaPassword(e.target.value)}
                className="input sm:max-w-56"
                autoComplete="current-password"
              />
              <input
                type="text"
                inputMode="numeric"
                pattern="[0-9 ]*"
                placeholder="Authentication code"
                value={mfaCode}
                onChange={(e) => setMfaCode(e.target.value)}
                className="input sm:max-w-56"
                autoComplete="one-time-code"
              />
              <button
                onClick={disableMFA}
                className="px-4 py-2 rounded-lg border border-[var(--color-error)] text-[var(--color-error)] text-body-small font-medium hover:bg-[rgba(255,59,48,0.08)] transition-colors"
                disabled={mfaBusy || !mfaPassword || mfaCode.trim().length < 6}
              >
                Disable MFA
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Maintenance */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2">
          <Database className="h-5 w-5 text-[var(--color-warning)]" />
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">
            Maintenance
          </h2>
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
          <h2 className="text-headline-3 text-[var(--color-text-primary)]">
            About
          </h2>
        </div>

        <div className="space-y-2 text-body text-[var(--color-text-secondary)]">
          <p>
            <strong>RenOS</strong> is an AI-powered operating ecosystem that
            unifies desktops, servers, Raspberry Pis, NAS devices, Docker
            containers, and VMs into one platform.
          </p>
          <p className="text-caption text-[var(--color-text-tertiary)]">
            Licensed under MIT. Built with Go, Rust, and Next.js.
          </p>
        </div>
      </div>
    </div>
  );
}

function InfoRow({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="flex justify-between items-center py-2 border-b border-[var(--color-border)]">
      <span className="text-body-small text-[var(--color-text-tertiary)]">
        {label}
      </span>
      <span
        className={`text-body-small text-[var(--color-text-primary)] ${mono ? "font-mono text-caption" : ""}`}
      >
        {value}
      </span>
    </div>
  );
}
