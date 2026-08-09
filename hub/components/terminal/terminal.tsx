"use client";

import { useEffect, useRef, useCallback, useState } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import "@xterm/xterm/css/xterm.css";

interface TerminalProps {
  deviceId: string;
  token: string;
  shell?: string;
  onSessionId?: (sessionId: string) => void;
  onDisconnect?: () => void;
  isActive: boolean;
}

interface TerminalMessage {
  type: string;
  data: Record<string, unknown>;
}

export function Terminal({
  deviceId,
  token,
  shell,
  onSessionId,
  onDisconnect,
  isActive,
}: TerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const searchAddonRef = useRef<SearchAddon | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [connected, setConnected] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);

  const connect = useCallback(() => {
    if (!terminalRef.current || wsRef.current) return;

    const xterm = new XTerm({
      cursorBlink: true,
      cursorStyle: "bar",
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', Menlo, monospace",
      theme: {
        background: "#1a1b26",
        foreground: "#c0caf5",
        cursor: "#c0caf5",
        cursorAccent: "#1a1b26",
        selectionBackground: "#33467c",
        black: "#15161e",
        red: "#f7768e",
        green: "#9ece6a",
        yellow: "#e0af68",
        blue: "#7aa2f7",
        magenta: "#bb9af7",
        cyan: "#7dcfff",
        white: "#a9b1d6",
        brightBlack: "#414868",
        brightRed: "#f7768e",
        brightGreen: "#9ece6a",
        brightYellow: "#e0af68",
        brightBlue: "#7aa2f7",
        brightMagenta: "#bb9af7",
        brightCyan: "#7dcfff",
        brightWhite: "#c0caf5",
      },
      allowProposedApi: true,
      scrollback: 10000,
      convertEol: true,
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    const searchAddon = new SearchAddon();

    xterm.loadAddon(fitAddon);
    xterm.loadAddon(webLinksAddon);
    xterm.loadAddon(searchAddon);

    xterm.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = xterm;
    fitAddonRef.current = fitAddon;
    searchAddonRef.current = searchAddon;

    // Connect WebSocket
    const wsUrl = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/api/v1/devices/${deviceId}/terminal/ws`;
    const params = new URLSearchParams({ token });
    if (shell) params.set("shell", shell);
    params.set("cols", String(xterm.cols));
    params.set("rows", String(xterm.rows));

    const ws = new WebSocket(`${wsUrl}?${params.toString()}`);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      xterm.focus();
    };

    ws.onmessage = (event) => {
      try {
        const msg: TerminalMessage = JSON.parse(event.data);
        switch (msg.type) {
          case "session":
            const sid = msg.data.session_id as string;
            setSessionId(sid);
            onSessionId?.(sid);
            break;
          case "output":
            const outputData = msg.data.data as string;
            const bytes = Uint8Array.from(atob(outputData), (c) => c.charCodeAt(0));
            xterm.write(bytes);
            break;
          case "error":
            xterm.writeln(`\r\n\x1b[31mError: ${msg.data.error}\x1b[0m`);
            break;
        }
      } catch {
        // ignore parse errors
      }
    };

    ws.onclose = () => {
      setConnected(false);
      xterm.writeln("\r\n\x1b[33mConnection closed. Reconnecting...\x1b[0m");
      onDisconnect?.();

      // Auto-reconnect after 3 seconds
      if (!reconnectTimeoutRef.current) {
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectTimeoutRef.current = null;
          cleanup();
          connect();
        }, 3000);
      }
    };

    ws.onerror = () => {
      setConnected(false);
    };

    // Terminal input → WebSocket
    const inputDisposable = xterm.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        const encoded = btoa(data);
        ws.send(
          JSON.stringify({
            type: "input",
            data: { data: encoded },
          })
        );
      }
    });

    // Terminal resize → WebSocket
    const resizeDisposable = xterm.onResize(({ cols, rows }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            type: "resize",
            data: { cols, rows },
          })
        );
      }
    });

    // Window resize → FitAddon
    const handleWindowResize = () => {
      fitAddon.fit();
    };
    window.addEventListener("resize", handleWindowResize);

    // Cleanup function stored for later
    const cleanup = () => {
      inputDisposable.dispose();
      resizeDisposable.dispose();
      window.removeEventListener("resize", handleWindowResize);
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
        wsRef.current = null;
      }
      setConnected(false);
    };

    // Store cleanup on the ref for external access
    (terminalRef.current as unknown as Record<string, unknown>).__cleanup = cleanup;
  }, [deviceId, token, shell, onSessionId, onDisconnect]);

  // Initialize on mount
  useEffect(() => {
    if (isActive) {
      connect();
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      const cleanup = (terminalRef.current as unknown as Record<string, unknown>)
        .__cleanup as (() => void) | undefined;
      cleanup?.();
      xtermRef.current?.dispose();
      xtermRef.current = null;
    };
  }, [isActive, connect]);

  // Handle fit when becoming active
  useEffect(() => {
    if (isActive && fitAddonRef.current) {
      setTimeout(() => fitAddonRef.current?.fit(), 100);
    }
  }, [isActive]);

  return (
    <div className="relative w-full h-full">
      <div ref={terminalRef} className="w-full h-full" />
      {/* Status bar */}
      <div
        className="absolute bottom-0 left-0 right-0 flex items-center justify-between px-2 py-0.5 text-[10px]"
        style={{
          backgroundColor: "rgba(26, 27, 38, 0.9)",
          color: connected ? "var(--color-success)" : "var(--color-error)",
        }}
      >
        <span>
          {connected ? "Connected" : "Disconnected"}
          {sessionId && ` · ${sessionId.slice(0, 8)}`}
        </span>
        <span>{xtermRef.current?.cols ?? 80}x{xtermRef.current?.rows ?? 24}</span>
      </div>
    </div>
  );
}

// Search toolbar component
export function TerminalSearch({ searchAddonRef }: { searchAddonRef: React.RefObject<SearchAddon | null> }) {
  const [query, setQuery] = useState("");

  const handleSearch = (direction: "next" | "prev") => {
    const addon = searchAddonRef.current;
    if (!addon) return;

    if (direction === "next") {
      addon.findNext(query);
    } else {
      addon.findPrevious(query);
    }
  };

  return (
    <div className="flex items-center gap-1 px-2 py-1" style={{ backgroundColor: "var(--color-surface)" }}>
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && handleSearch("next")}
        placeholder="Search..."
        className="px-2 py-0.5 text-xs rounded border"
        style={{
          backgroundColor: "var(--color-background)",
          borderColor: "var(--color-border)",
          color: "var(--color-text-primary)",
        }}
      />
      <button onClick={() => handleSearch("prev")} className="text-xs px-1" style={{ color: "var(--color-text-secondary)" }}>↑</button>
      <button onClick={() => handleSearch("next")} className="text-xs px-1" style={{ color: "var(--color-text-secondary)" }}>↓</button>
    </div>
  );
}
