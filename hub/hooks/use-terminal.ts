"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { TerminalSession, TerminalRecording } from "@/lib/types";

export function useTerminalSessions(deviceId: string) {
  return useQuery<{ sessions: TerminalSession[] }>({
    queryKey: ["terminal-sessions", deviceId],
    queryFn: () => api.listTerminalSessions(deviceId, 20),
    refetchInterval: 10_000,
  });
}

export function useMyTerminalSessions() {
  return useQuery<{ sessions: TerminalSession[] }>({
    queryKey: ["my-terminal-sessions"],
    queryFn: () => api.listMyTerminalSessions(50),
    refetchInterval: 30_000,
  });
}

export function useTerminalRecording(deviceId: string, sessionId: string | null) {
  return useQuery<{ recordings: TerminalRecording[] }>({
    queryKey: ["terminal-recording", deviceId, sessionId],
    queryFn: () => api.getTerminalRecording(deviceId, sessionId!),
    enabled: !!sessionId,
  });
}
