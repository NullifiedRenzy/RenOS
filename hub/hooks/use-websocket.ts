"use client";

import { useEffect, useRef, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/auth-store";

const RECONNECT_BASE = 1000;
const RECONNECT_MAX = 30000;

export function useWebSocket() {
  const queryClient = useQueryClient();
  const token = useAuthStore((s) => s.token);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const wsRef = useRef<WebSocket | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);

  const connect = useCallback(() => {
    if (!token) return;

    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${proto}//${window.location.host}/api/v1/ws?token=${encodeURIComponent(token)}`;

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      attemptRef.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (["device.health", "device.online", "device.offline"].includes(msg.type)) {
          queryClient.invalidateQueries({ queryKey: ["devices"] });
          if (msg.device_id) {
            queryClient.invalidateQueries({ queryKey: ["device", msg.device_id] });
          }
        }
        if (["execution.created", "execution.output", "execution.completed", "execution.cancelled"].includes(msg.type)) {
          if (msg.device_id) {
            queryClient.invalidateQueries({ queryKey: ["executions", msg.device_id] });
            if (msg.data?.execution_id) {
              queryClient.invalidateQueries({ queryKey: ["execution-output", msg.device_id, msg.data.execution_id] });
            }
          }
        }
      } catch {
        /* ignore malformed messages */
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (isAuthenticated) {
        const delay = Math.min(RECONNECT_BASE * 2 ** attemptRef.current, RECONNECT_MAX);
        attemptRef.current++;
        timerRef.current = setTimeout(connect, delay);
      }
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [token, isAuthenticated, queryClient]);

  useEffect(() => {
    if (isAuthenticated && token) {
      connect();
    }
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [isAuthenticated, token, connect]);
}
