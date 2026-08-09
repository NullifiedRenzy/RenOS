"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowUpRight, Boxes, ChevronRight, ClipboardList, Command as CommandIcon, CornerDownLeft,
  Cpu, KeySquare, Logs, MonitorCog, Package, Rocket, Search, Server,
  ShieldCheck, Users, Vault, Workflow, type LucideIcon,
} from "lucide-react";
import { api } from "@/lib/api-client";
import { fuzzyRank } from "@/lib/fuzzy";
import { getCommands, type Command } from "@/lib/commands";
import type { SearchResponse, SearchResult } from "@/lib/types";
import { useCommandCenterStore } from "@/stores/command-center-store";
import { useUiStore } from "@/stores/ui-store";
import { cn } from "@/lib/utils";

const CATEGORY_ICONS: Record<string, LucideIcon> = {
  devices: MonitorCog,
  jobs: ClipboardList,
  workflows: Workflow,
  plugins: Package,
  marketplace: Rocket,
  policies: ShieldCheck,
  secrets: Vault,
  users: Users,
  logs: Logs,
  containers: Boxes,
  vms: Cpu,
  "proxmox-vms": Server,
  clusters: Boxes,
};

const CATEGORY_LABELS: Record<string, string> = {
  devices: "Devices",
  jobs: "Jobs",
  workflows: "Workflows",
  plugins: "Plugins",
  marketplace: "Marketplace",
  policies: "Policies",
  secrets: "Secrets",
  users: "Users",
  logs: "Logs",
  containers: "Containers",
  vms: "Virtual Machines",
  "proxmox-vms": "Proxmox VMs",
  clusters: "Cluster Nodes",
};

type PickType = "devices" | "workflows";

interface PickContext {
  type: PickType;
  commandId?: string;
}

interface PickItem {
  id: string;
  title: string;
  subtitle: string;
}

type ListEntry =
  | { kind: "header"; label: string; icon?: LucideIcon }
  | { kind: "result"; result: SearchResult }
  | { kind: "command"; command: Command }
  | { kind: "pick"; item: PickItem };

function isTypingTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el) return false;
  const tag = el.tagName;
  return (
    tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" ||
    el.isContentEditable ||
    (el.getAttribute && el.getAttribute("role") === "combobox")
  );
}

export function CommandCenter() {
  const router = useRouter();
  const { isOpen, mode, open, close, setMode } = useCommandCenterStore();
  const setTheme = useUiStore((s) => s.setTheme);
  const toggleSidebar = useUiStore((s) => s.toggleSidebar);

  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(0);
  const [picking, setPicking] = useState<PickContext | null>(null);
  const [picks, setPicks] = useState<PickItem[]>([]);
  const [pickQuery, setPickQuery] = useState("");
  const [notice, setNotice] = useState<string | null>(null);

  const inputRef = useRef<HTMLInputElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const lastFocused = useRef<HTMLElement | null>(null);

  const commandPrefix = query.trimStart().startsWith(">");
  const effectiveMode: "search" | "command" = commandPrefix ? "command" : mode;
  const searchQuery = commandPrefix ? query.trimStart().slice(1).trimStart() : query.trim();

  // Reset state on open/close.
  useEffect(() => {
    if (isOpen) {
      lastFocused.current = document.activeElement as HTMLElement | null;
      setQuery("");
      setPicking(null);
      setPickQuery("");
      setNotice(null);
      document.body.style.overflow = "hidden";
      const t = window.setTimeout(() => inputRef.current?.focus(), 10);
      return () => window.clearTimeout(t);
    }
    document.body.style.overflow = "";
    lastFocused.current?.focus?.();
    return undefined;
  }, [isOpen]);

  // Global shortcut: Ctrl/Cmd+K toggles, "/" opens search.
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        open();
        return;
      }
      if (e.key === "/" && !isOpen && !isTypingTarget(e.target)) {
        e.preventDefault();
        open("search");
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [isOpen, open]);

  // Debounced backend search via react-query.
  const searchEnabled = isOpen && effectiveMode === "search" && searchQuery.length > 0;
  const { data: searchData, isFetching: searching } = useQuery({
    queryKey: ["global-search", searchQuery],
    queryFn: () => api.search(searchQuery, 10),
    enabled: searchEnabled,
    placeholderData: (prev) => prev,
    staleTime: 30_000,
  });

  const commands = useMemo(() => getCommands(), []);
  const filteredCommands = useMemo(() => {
    if (effectiveMode !== "command") return [];
    return fuzzyRank(commands, commandPrefix ? searchQuery : query, (c) => `${c.title} ${c.keywords}`, 14);
  }, [commands, effectiveMode, commandPrefix, searchQuery, query]);

  // Load pick items when a command needs a device/workflow.
  useEffect(() => {
    if (!isOpen || !picking) {
      setPicks([]);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        if (picking.type === "devices") {
          const res = await api.listDevices();
          if (cancelled) return;
          setPicks(
            (res.devices ?? []).map((d) => ({
              id: d.id,
              title: d.hostname,
              subtitle: `${d.os || "unknown"} · ${d.status || "unknown"}`,
            })),
          );
        } else {
          const res = await api.getWorkflows();
          if (cancelled) return;
          setPicks(
            (res.data ?? []).map((w) => ({
              id: w.id,
              title: w.name,
              subtitle: w.description || "Workflow",
            })),
          );
        }
      } catch {
        setPicks([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isOpen, picking]);

  const filteredPicks = useMemo(() => {
    if (!pickQuery.trim()) return picks.slice(0, 12);
    return fuzzyRank(picks, pickQuery, (p) => `${p.title} ${p.subtitle}`, 12);
  }, [picks, pickQuery]);

  const entries = useMemo<ListEntry[]>(() => {
    if (picking) {
      return [
        { kind: "header", label: picking.type === "devices" ? "Select a device" : "Select a workflow" },
        ...filteredPicks.map((item) => ({ kind: "pick" as const, item })),
      ];
    }
    if (effectiveMode === "command") {
      const groups = new Map<string, Command[]>();
      for (const c of filteredCommands) {
        if (!groups.has(c.group)) groups.set(c.group, []);
        groups.get(c.group)!.push(c);
      }
      const out: ListEntry[] = [];
      for (const [label, cmds] of groups) {
        out.push({ kind: "header", label });
        for (const command of cmds) out.push({ kind: "command", command });
      }
      return out;
    }
    const data: SearchResponse | undefined = searchData;
    const out: ListEntry[] = [];
    if (data && data.total > 0) {
      for (const [cat, results] of Object.entries(data.results)) {
        if (results.length === 0) continue;
        out.push({ kind: "header", label: CATEGORY_LABELS[cat] ?? cat, icon: CATEGORY_ICONS[cat] });
        for (const result of results) out.push({ kind: "result", result });
      }
    }
    return out;
  }, [picking, filteredPicks, effectiveMode, filteredCommands, searchData]);

  const selectable = useMemo(
    () => entries.filter((e) => e.kind !== "header"),
    [entries],
  );

  useEffect(() => {
    setSelected(0);
  }, [effectiveMode, query, picking, pickQuery]);

  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-index="${selected}"]`);
    (el as HTMLElement | null)?.scrollIntoView({ block: "nearest" });
  }, [selected]);

  const runCommand = useCallback(
    (command: Command, arg?: { id: string; title: string }) => {
      if (command.pick && !arg) {
        setPicking({ type: command.pick, commandId: command.id });
        setPickQuery("");
        return;
      }
      close();
      command.run({
        navigate: (href) => router.push(href),
        execute: (id, a) => {
          switch (id) {
            case "run-workflow":
              if (a) {
                api.executeWorkflow(a.id).catch(() => undefined);
                router.push("/workflows");
              }
              break;
            case "open-terminal":
              if (a) router.push(`/devices/${a.id}/terminal`);
              break;
            case "restart-device":
              if (a) router.push(`/devices/${a.id}`);
              break;
            case "toggle-theme":
              setTheme(document.documentElement.classList.contains("dark") ? "light" : "dark");
              break;
            case "light-mode":
              setTheme("light");
              break;
            case "toggle-sidebar":
              toggleSidebar();
              break;
            case "help-shortcuts":
              setNotice("Keyboard: Ctrl+K search · > commands · ↑/↓ navigate · Enter run · Esc close");
              window.setTimeout(() => setNotice(null), 4000);
              break;
          }
        },
      });
    },
    [close, router, setTheme, toggleSidebar],
  );

  const runEntry = useCallback(
    (entry: ListEntry) => {
      if (entry.kind === "result") {
        close();
        router.push(entry.result.href || "/dashboard");
      } else if (entry.kind === "command") {
        runCommand(entry.command);
      } else if (entry.kind === "pick") {
        const item = entry.item;
        if (picking?.type === "devices") {
          if (picking.commandId === "open-terminal") {
            close();
            router.push(`/devices/${item.id}/terminal`);
          } else if (picking.commandId === "restart-device") {
            close();
            router.push(`/devices/${item.id}`);
          } else {
            close();
            router.push(`/devices/${item.id}`);
          }
        } else if (picking?.type === "workflows") {
          close();
          api.executeWorkflow(item.id).catch(() => undefined);
          router.push("/workflows");
        }
      }
    },
    [close, router, picking],
  );

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        if (picking) {
          setPicking(null);
        } else {
          close();
        }
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelected((s) => Math.min(s + 1, Math.max(selectable.length - 1, 0)));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelected((s) => Math.max(s - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        const entry = selectable[selected];
        if (entry) runEntry(entry);
      } else if (e.key === "Tab") {
        // Simple focus trap.
        const focusables = panelRef.current?.querySelectorAll<HTMLElement>(
          'button, [href], input, [tabindex]:not([tabindex="-1"])',
        );
        if (!focusables || focusables.length === 0) return;
        const first = focusables[0];
        const last = focusables[focusables.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    },
    [selectable, selected, runEntry, close, picking],
  );

  if (!isOpen) return null;

  const inputValue = commandPrefix ? query : query;

  return (
    <div className="fixed inset-0 z-[100] flex items-start justify-center pt-[12vh] px-4">
      <button
        aria-label="Close search"
        onClick={close}
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        tabIndex={-1}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label="Command center"
        onKeyDown={onKeyDown}
        className="relative w-full max-w-xl rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] shadow-2xl overflow-hidden focus:outline-none"
        style={{ animation: "fade-in-up 120ms ease-out" }}
      >
        <div className="flex items-center gap-3 px-4 border-b border-[var(--color-border)]">
          <span className="text-[var(--color-text-tertiary)]" aria-hidden="true">
            {commandPrefix ? ">" : <Search size={18} />}
          </span>
          <input
            ref={inputRef}
            value={inputValue}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={
              picking
                ? picking.type === "devices"
                  ? "Type to filter devices…"
                  : "Type to filter workflows…"
                : effectiveMode === "command"
                  ? "Type a command…"
                  : "Search devices, workflows, plugins, policies, logs…"
            }
            aria-label={picking ? "Filter list" : effectiveMode === "command" ? "Command" : "Search the platform"}
            className="flex-1 bg-transparent border-none outline-none h-12 text-body text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)]"
          />
          <div className="flex items-center gap-1" role="tablist" aria-label="Command center mode">
            <button
              onClick={() => { setMode("search"); setQuery((q) => q.replace(/^>\s*/, "")); }}
              aria-label="Search mode"
              aria-selected={effectiveMode === "search"}
              role="tab"
              className={cn(
                "p-1.5 rounded-md transition-colors",
                effectiveMode === "search"
                  ? "bg-[var(--color-accent)]/10 text-[var(--color-accent)]"
                  : "text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]",
              )}
            >
              <Search size={15} />
            </button>
            <button
              onClick={() => { setMode("command"); if (!commandPrefix) setQuery("> "); }}
              aria-label="Command palette"
              aria-selected={effectiveMode === "command"}
              role="tab"
              className={cn(
                "p-1.5 rounded-md transition-colors",
                effectiveMode === "command"
                  ? "bg-[var(--color-accent)]/10 text-[var(--color-accent)]"
                  : "text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]",
              )}
            >
              <CommandIcon size={15} />
            </button>
          </div>
        </div>

        <div ref={listRef} className="max-h-[46vh] overflow-y-auto py-2">
          {entries.length === 0 && (
            <div className="px-4 py-10 text-center">
              {effectiveMode === "search" ? (
                <>
                  {searching ? (
                    <div className="flex items-center justify-center gap-2 text-caption text-[var(--color-text-tertiary)]">
                      <span className="skeleton h-3 w-3 rounded-full animate-pulse" />
                      Searching…
                    </div>
                  ) : searchQuery ? (
                    <p className="text-caption text-[var(--color-text-tertiary)]">
                      No results for “{searchQuery}”. Try a device, workflow, or policy name.
                    </p>
                  ) : (
                    <>
                      <Search size={28} className="mx-auto mb-3 text-[var(--color-text-tertiary)]" />
                      <p className="text-caption text-[var(--color-text-tertiary)]">
                        Search devices, workflows, plugins, policies, secrets, logs and more.
                      </p>
                      <p className="mt-2 text-caption text-[var(--color-text-tertiary)] opacity-70">
                        Type <kbd className="kbd">Ctrl</kbd> <kbd className="kbd">K</kbd> to open · type <kbd className="kbd">&gt;</kbd> for commands
                      </p>
                    </>
                  )}
                </>
              ) : (
                <p className="text-caption text-[var(--color-text-tertiary)]">
                  No matching commands. Try “Open Devices” or “Run Workflow”.
                </p>
              )}
            </div>
          )}

          {entries.map((entry, i) => {
            if (entry.kind === "header") {
              return (
                <div key={`h-${i}`} className="px-4 pt-3 pb-1.5 flex items-center gap-2">
                  {entry.icon && <entry.icon size={12} className="text-[var(--color-text-tertiary)]" />}
                  <span className="text-caption uppercase tracking-wider text-[var(--color-text-tertiary)]">
                    {entry.label}
                  </span>
                </div>
              );
            }
            const idx = selectable.indexOf(entry);
            const isActive = idx === selected;
            if (entry.kind === "result") {
              const Icon = CATEGORY_ICONS[entry.result.type] ?? Search;
              return (
                <button
                  key={`r-${entry.result.type}-${entry.result.id}`}
                  data-index={idx}
                  onMouseEnter={() => setSelected(idx)}
                  onClick={() => runEntry(entry)}
                  className={cn(
                    "w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors",
                    isActive ? "bg-[var(--color-accent)]/10" : "hover:bg-[var(--color-surface-hover)]",
                  )}
                >
                  <Icon size={16} className="text-[var(--color-text-tertiary)] flex-shrink-0" />
                  <span className="min-w-0 flex-1">
                    <span className="block text-body truncate text-[var(--color-text-primary)]">{entry.result.title}</span>
                    <span className="block text-caption truncate text-[var(--color-text-tertiary)]">{entry.result.subtitle}</span>
                  </span>
                  <ArrowUpRight size={14} className="text-[var(--color-text-tertiary)] flex-shrink-0" />
                </button>
              );
            }
            if (entry.kind === "command") {
              const Icon = entry.command.icon;
              return (
                <button
                  key={`c-${entry.command.id}`}
                  data-index={idx}
                  onMouseEnter={() => setSelected(idx)}
                  onClick={() => runEntry(entry)}
                  className={cn(
                    "w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors",
                    isActive ? "bg-[var(--color-accent)]/10" : "hover:bg-[var(--color-surface-hover)]",
                  )}
                >
                  <Icon size={16} className="text-[var(--color-text-tertiary)] flex-shrink-0" />
                  <span className="flex-1 text-body text-[var(--color-text-primary)]">{entry.command.title}</span>
                  {entry.command.shortcut && (
                    <span className="text-caption text-[var(--color-text-tertiary)] whitespace-nowrap">
                      {entry.command.shortcut}
                    </span>
                  )}
                  {entry.command.pick && <ChevronRight size={14} className="text-[var(--color-text-tertiary)]" />}
                </button>
              );
            }
            return (
              <button
                key={`p-${entry.item.id}`}
                data-index={idx}
                onMouseEnter={() => setSelected(idx)}
                onClick={() => runEntry(entry)}
                className={cn(
                  "w-full text-left px-4 py-2.5 flex items-center gap-3 transition-colors",
                  isActive ? "bg-[var(--color-accent)]/10" : "hover:bg-[var(--color-surface-hover)]",
                )}
              >
                {picking?.type === "devices" ? (
                  <MonitorCog size={16} className="text-[var(--color-text-tertiary)] flex-shrink-0" />
                ) : (
                  <Workflow size={16} className="text-[var(--color-text-tertiary)] flex-shrink-0" />
                )}
                <span className="min-w-0 flex-1">
                  <span className="block text-body truncate text-[var(--color-text-primary)]">{entry.item.title}</span>
                  <span className="block text-caption truncate text-[var(--color-text-tertiary)]">{entry.item.subtitle}</span>
                </span>
              </button>
            );
          })}
        </div>

        <div className="flex items-center justify-between px-4 py-2 border-t border-[var(--color-border)] bg-[var(--color-surface)]">
          {notice ? (
            <span className="text-caption text-[var(--color-text-tertiary)]">{notice}</span>
          ) : (
            <div className="flex items-center gap-3 text-caption text-[var(--color-text-tertiary)]">
              <span className="flex items-center gap-1">
                <kbd className="kbd">↑</kbd><kbd className="kbd">↓</kbd> navigate
              </span>
              <span className="flex items-center gap-1">
                <kbd className="kbd"><CornerDownLeft size={11} /></kbd> select
              </span>
              <span className="flex items-center gap-1">
                <kbd className="kbd">esc</kbd> close
              </span>
            </div>
          )}
          {picking && (
            <input
              value={pickQuery}
              onChange={(e) => setPickQuery(e.target.value)}
              placeholder="Filter…"
              aria-label="Filter"
              className="w-40 px-2 py-1 text-caption rounded-md border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent)]"
            />
          )}
        </div>
      </div>
    </div>
  );
}
