"use client";

import dynamic from "next/dynamic";
import { useParams } from "next/navigation";
import Link from "next/link";
import { useDevice } from "@/hooks/use-devices";
import { useTerminalSessions } from "@/hooks/use-terminal";
import { useAuth } from "@/hooks/use-auth";

const TerminalManager = dynamic(
  () => import("@/components/terminal/terminal-manager").then((m) => m.TerminalManager),
  { ssr: false, loading: () => <div className="h-full w-full skeleton" /> },
);

export default function TerminalPage() {
  const params = useParams();
  const deviceId = params.id as string;
  const { data: device } = useDevice(deviceId);
  const { data: sessionsData } = useTerminalSessions(deviceId);
  const { token } = useAuth();

  if (!device || !token) {
    return (
      <div className="p-6" style={{ color: "var(--color-text-secondary)" }}>
        Loading...
      </div>
    );
  }

  const sessions = sessionsData?.sessions ?? [];
  const activeSessions = sessions.filter((s) => s.status === "active");

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <Link
            href={`/devices/${deviceId}`}
            className="text-sm px-2 py-1 rounded"
            style={{ color: "var(--color-text-secondary)" }}
          >
            ← Back
          </Link>
          <h1 className="text-lg font-semibold" style={{ color: "var(--color-text-primary)" }}>
            Terminal — {device.device.hostname}
          </h1>
          {activeSessions.length > 0 && (
            <span className="text-xs px-2 py-0.5 rounded" style={{ backgroundColor: "var(--color-success)", color: "white" }}>
              {activeSessions.length} active
            </span>
          )}
        </div>
      </div>

      {/* Terminal */}
      <div
        className="flex-1 rounded-lg overflow-hidden"
        style={{
          backgroundColor: "#1a1b26",
          border: "1px solid var(--color-border)",
        }}
      >
        <TerminalManager deviceId={deviceId} token={token} />
      </div>
    </div>
  );
}
