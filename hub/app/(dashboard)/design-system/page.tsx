"use client";

import { useState } from "react";
import { Plus, Save, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Card, CardHeader, CardBody, CardFooter } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Table } from "@/components/ui/table";
import { Dialog } from "@/components/ui/dialog";
import { Tabs } from "@/components/ui/tabs";
import { Skeleton, SkeletonCircle } from "@/components/ui/skeleton";

const colorTokens = [
  { token: "--color-bg", light: "#FFFFFF", dark: "#1C1C1E" },
  { token: "--color-bg-primary", light: "#FFFFFF", dark: "#1C1C1E" },
  { token: "--color-bg-secondary", light: "#F2F2F7", dark: "#2C2C2E" },
  { token: "--color-surface", light: "#F2F2F7", dark: "#2C2C2E" },
  { token: "--color-surface-elevated", light: "#FFFFFF", dark: "#3A3A3C" },
  { token: "--color-border", light: "#E5E5EA", dark: "#3A3A3C" },
  { token: "--color-text-primary", light: "#0A0A0B", dark: "#FFFFFF" },
  { token: "--color-text-secondary", light: "#636366", dark: "#E5E5EA" },
  { token: "--color-text-tertiary", light: "#8E8E93", dark: "#8E8E93" },
  { token: "--color-accent", light: "#007AFF", dark: "#0A84FF" },
  { token: "--color-success", light: "#34C759", dark: "#32D74B" },
  { token: "--color-warning", light: "#FF9500", dark: "#FFD60A" },
  { token: "--color-error", light: "#FF3B30", dark: "#FF453A" },
  { token: "--color-info", light: "#5AC8FA", dark: "#64D2FF" },
];

const spacingTokens = ["--space-1", "--space-2", "--space-3", "--space-4", "--space-5", "--space-6", "--space-8", "--space-10", "--space-12", "--space-16"];

const motionTokens = [
  { token: "--duration-fast", value: "100ms", use: "Hover, press, focus transitions" },
  { token: "--duration-med", value: "200ms", use: "Dropdowns, toggles, color shifts" },
  { token: "--duration-slow", value: "400ms", use: "Modals, page transitions" },
];

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="scroll-mt-6">
      <h2 className="text-headline-3 text-[var(--color-text-primary)] mb-4">{title}</h2>
      {children}
    </section>
  );
}

const sampleRows = [
  { name: "webserver-01", status: "online", region: "us-east-1" },
  { name: "db-02", status: "degraded", region: "eu-west-1" },
  { name: "cache-01", status: "offline", region: "ap-south-1" },
];

export default function DesignSystemPage() {
  const [tab, setTab] = useState("overview");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const [showSkeleton, setShowSkeleton] = useState(true);

  return (
    <div className="space-y-10 max-w-5xl">
      <header>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Design System</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-2 max-w-2xl">
          Semantic design tokens and the core UI component library. Everything is
          dark-mode aware, keyboard accessible, and consumed via
          <code className="mx-1 px-1.5 py-0.5 rounded-[var(--radius-sm)] bg-[var(--color-surface)] font-mono text-body-small">
            styles/tokens.css
          </code>
          and
          <code className="mx-1 px-1.5 py-0.5 rounded-[var(--radius-sm)] bg-[var(--color-surface)] font-mono text-body-small">
            components/ui
          </code>.
        </p>
      </header>

      <div className="sticky top-14 z-10 flex gap-1.5 overflow-x-auto pb-2 -mx-1 px-1 bg-[var(--color-bg)]">
        {["tokens", "buttons", "inputs", "cards", "badges", "table", "tabs", "dialog", "skeleton"].map((id) => (
          <a
            key={id}
            href={`#${id}`}
            className="shrink-0 px-3 py-1.5 rounded-[var(--radius-sm)] text-body-small text-[var(--color-text-secondary)] border border-[var(--color-border)] hover:border-[var(--color-accent)] hover:text-[var(--color-accent)] transition-colors duration-[var(--duration-fast)]"
          >
            {id}
          </a>
        ))}
      </div>

      <Section id="tokens" title="Design Tokens">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {colorTokens.map((c) => (
            <div key={c.token} className="flex items-center gap-3 rounded-[var(--radius-md)] border border-[var(--color-border)] p-3">
              <span className="w-8 h-8 rounded-[var(--radius-sm)] border border-[var(--color-border)] shrink-0" style={{ backgroundColor: `var(${c.token})` }} />
              <div className="min-w-0">
                <code className="text-body-small font-mono text-[var(--color-text-primary)]">{c.token}</code>
                <p className="text-caption text-[var(--color-text-tertiary)]">L {c.light} · D {c.dark}</p>
              </div>
            </div>
          ))}
        </div>

        <h3 className="text-headline-4 text-[var(--color-text-primary)] mt-8 mb-3">Spacing (4px base)</h3>
        <div className="space-y-1.5">
          {spacingTokens.map((t) => (
            <div key={t} className="flex items-center gap-3">
              <code className="w-24 text-body-small font-mono text-[var(--color-text-tertiary)]">{t}</code>
              <div className="h-4 rounded-[var(--radius-sm)] bg-[var(--color-accent)]" style={{ width: `var(${t})` }} />
            </div>
          ))}
        </div>

        <h3 className="text-headline-4 text-[var(--color-text-primary)] mt-8 mb-3">Motion</h3>
        <Table
          columns={[
            { key: "token", header: "Token", cell: (r) => <code className="font-mono text-body-small">{r.token}</code> },
            { key: "value", header: "Value", cell: (r) => r.value },
            { key: "use", header: "Use for", cell: (r) => r.use },
          ]}
          rows={motionTokens}
          rowKey={(r) => r.token}
        />
      </Section>

      <Section id="buttons" title="Buttons">
        <div className="flex flex-wrap items-center gap-3">
          <Button>Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Destructive</Button>
          <Button isLoading>Loading</Button>
          <Button disabled>Disabled</Button>
          <Button size="sm">Small</Button>
          <Button size="lg" leftIcon={<Plus size={16} aria-hidden="true" />}>With icon</Button>
        </div>
      </Section>

      <Section id="inputs" title="Inputs">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-3xl">
          <Input label="Hostname" placeholder="webserver-01" value={inputValue} onChange={(e) => setInputValue(e.target.value)} />
          <Input label="API token" error="Token is required" placeholder="sk-…" defaultValue="sk-invalid" />
          <Input label="Port" hint="Default 22" placeholder="22" type="number" />
          <Select
            label="Region"
            options={[
              { value: "us-east-1", label: "US East (N. Virginia)" },
              { value: "eu-west-1", label: "EU West (Ireland)" },
              { value: "ap-south-1", label: "Asia Pacific (Mumbai)" },
            ]}
          />
        </div>
      </Section>

      <Section id="cards" title="Cards">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-4xl">
          <Card>
            <CardHeader title="Default card" description="Surface-elevated with shadow-sm." />
            <CardBody>Body content sits on the elevated surface with secondary text by default.</CardBody>
            <CardFooter>
              <Button size="sm" variant="secondary">Action</Button>
            </CardFooter>
          </Card>
          <Card interactive>
            <CardHeader title="Interactive card" description="Hover lifts shadow and accents the border." />
            <CardBody>Use for clickable tiles and navigation cards.</CardBody>
          </Card>
        </div>
      </Section>

      <Section id="badges" title="Badges">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="neutral">neutral</Badge>
          <Badge tone="success" dot>online</Badge>
          <Badge tone="warning" dot>degraded</Badge>
          <Badge tone="error" dot>critical</Badge>
          <Badge tone="info" dot>info</Badge>
          <Badge tone="accent" dot>accent</Badge>
        </div>
      </Section>

      <Section id="table" title="Table">
        <div className="max-w-3xl">
          <Table
            columns={[
              { key: "name", header: "Name", cell: (r) => <span className="text-[var(--color-text-primary)] font-medium">{r.name}</span> },
              {
                key: "status",
                header: "Status",
                cell: (r) => <Badge tone={r.status === "online" ? "success" : r.status === "degraded" ? "warning" : "error"} dot>{r.status}</Badge>,
              },
              { key: "region", header: "Region", cell: (r) => r.region },
              {
                key: "actions",
                header: "",
                headerClassName: "text-right",
                className: "text-right",
                cell: () => (
                  <div className="inline-flex gap-1">
                    <Button size="sm" variant="ghost" aria-label="Save"><Save size={14} aria-hidden="true" /></Button>
                    <Button size="sm" variant="destructive" aria-label="Delete"><Trash2 size={14} aria-hidden="true" /></Button>
                  </div>
                ),
              },
            ]}
            rows={sampleRows}
            rowKey={(r) => r.name}
          />
        </div>
      </Section>

      <Section id="tabs" title="Tabs">
        <Tabs
          tabs={[
            { id: "overview", label: "Overview" },
            { id: "alerts", label: "Alerts", badge: 3 },
            { id: "logs", label: "Logs" },
          ]}
          active={tab}
          onChange={setTab}
        />
        <div className="mt-4 text-body text-[var(--color-text-secondary)]">Active tab: {tab}. Arrow keys navigate.</div>
      </Section>

      <Section id="dialog" title="Dialog">
        <Button onClick={() => setDialogOpen(true)}>Open dialog</Button>
        <Dialog
          open={dialogOpen}
          onClose={() => setDialogOpen(false)}
          title="Confirm action"
          description="Dialogs trap focus, close on Esc, and restore focus to the trigger."
          footer={
            <>
              <Button variant="ghost" onClick={() => setDialogOpen(false)}>Cancel</Button>
              <Button onClick={() => setDialogOpen(false)}>Confirm</Button>
            </>
          }
        >
          <p className="text-body text-[var(--color-text-secondary)]">
            Press Tab to cycle focus within the dialog. Press Esc to close.
          </p>
        </Dialog>
      </Section>

      <Section id="skeleton" title="Skeleton">
        <div className="flex flex-wrap items-start gap-6">
          <div className="flex items-center gap-3">
            <SkeletonCircle size={48} />
            <div className="w-40">
              <Skeleton lines={2} />
            </div>
          </div>
          <div className="w-64">
            <Skeleton lines={4} />
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm" onClick={() => setShowSkeleton((s) => !s)}>
              {showSkeleton ? "Hide" : "Show"} loading
            </Button>
          </div>
        </div>
        {showSkeleton && <div className="mt-4 w-96"><Skeleton lines={3} /></div>}
      </Section>
    </div>
  );
}
