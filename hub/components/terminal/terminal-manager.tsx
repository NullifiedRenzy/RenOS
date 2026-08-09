"use client";

import { useState, useCallback } from "react";
import { Terminal } from "./terminal";

interface TerminalTab {
  id: string;
  label: string;
  sessionId: string | null;
  isConnected: boolean;
}

interface TerminalManagerProps {
  deviceId: string;
  token: string;
}

let tabCounter = 0;

export function TerminalManager({ deviceId, token }: TerminalManagerProps) {
  const [tabs, setTabs] = useState<TerminalTab[]>(() => {
    tabCounter++;
    return [
      {
        id: `tab-${tabCounter}`,
        label: `Terminal ${tabCounter}`,
        sessionId: null,
        isConnected: false,
      },
    ];
  });
  const [activeTabId, setActiveTabId] = useState(tabs[0]?.id ?? "");

  const addTab = useCallback(() => {
    tabCounter++;
    const newTab: TerminalTab = {
      id: `tab-${tabCounter}`,
      label: `Terminal ${tabCounter}`,
      sessionId: null,
      isConnected: false,
    };
    setTabs((prev) => [...prev, newTab]);
    setActiveTabId(newTab.id);
  }, []);

  const closeTab = useCallback(
    (tabId: string) => {
      setTabs((prev) => {
        const filtered = prev.filter((t) => t.id !== tabId);
        if (filtered.length === 0) {
          // Always keep at least one tab
          tabCounter++;
          const newTab: TerminalTab = {
            id: `tab-${tabCounter}`,
            label: `Terminal ${tabCounter}`,
            sessionId: null,
            isConnected: false,
          };
          setActiveTabId(newTab.id);
          return [newTab];
        }
        if (activeTabId === tabId) {
          setActiveTabId(filtered[filtered.length - 1].id);
        }
        return filtered;
      });
    },
    [activeTabId]
  );

  const updateTab = useCallback((tabId: string, updates: Partial<TerminalTab>) => {
    setTabs((prev) =>
      prev.map((t) => (t.id === tabId ? { ...t, ...updates } : t))
    );
  }, []);

  const activeTab = tabs.find((t) => t.id === activeTabId) ?? tabs[0];

  return (
    <div className="flex flex-col h-full">
      {/* Tab bar */}
      <div
        className="flex items-center gap-0 overflow-x-auto"
        style={{
          backgroundColor: "var(--color-surface)",
          borderBottom: "1px solid var(--color-border)",
        }}
      >
        {tabs.map((tab) => (
          <div
            key={tab.id}
            className="flex items-center gap-1 px-3 py-1.5 text-xs cursor-pointer border-r"
            style={{
              backgroundColor: tab.id === activeTabId ? "var(--color-background)" : "transparent",
              color:
                tab.id === activeTabId
                  ? "var(--color-text-primary)"
                  : "var(--color-text-secondary)",
              borderColor: "var(--color-border)",
            }}
            onClick={() => setActiveTabId(tab.id)}
          >
            <span
              className="inline-block w-1.5 h-1.5 rounded-full"
              style={{
                backgroundColor: tab.isConnected ? "var(--color-success)" : "var(--color-text-tertiary)",
              }}
            />
            <span className="max-w-[100px] truncate">{tab.label}</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                closeTab(tab.id);
              }}
              className="ml-1 hover:opacity-70"
              style={{ color: "var(--color-text-tertiary)" }}
            >
              ×
            </button>
          </div>
        ))}
        <button
          onClick={addTab}
          className="px-2 py-1.5 text-xs"
          style={{ color: "var(--color-text-tertiary)" }}
          title="New terminal"
        >
          +
        </button>
      </div>

      {/* Terminal content */}
      <div className="flex-1 relative">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            className="absolute inset-0"
            style={{ display: tab.id === activeTabId ? "block" : "none" }}
          >
            <Terminal
              deviceId={deviceId}
              token={token}
              isActive={tab.id === activeTabId}
              onSessionId={(sid) => updateTab(tab.id, { sessionId: sid })}
              onDisconnect={() => updateTab(tab.id, { isConnected: false })}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
