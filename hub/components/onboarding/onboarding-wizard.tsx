"use client";

import { useMemo, useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Sparkles,
  Brain,
  BellRing,
  Laptop,
  Puzzle,
  Archive,
  RefreshCw,
  PartyPopper,
  Check,
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
  Loader2,
  Rocket,
  HeartPulse,
  ExternalLink,
  AlertCircle,
} from "lucide-react";
import { useOnboarding } from "@/hooks/use-onboarding";
import { useDemo } from "@/hooks/use-demo";
import type { OnboardingStatus, DemoSeedResult } from "@/lib/types";
import { cn } from "@/lib/utils";

const providers = [
  { id: "ollama", name: "Ollama", description: "Run models locally on your own hardware." },
  { id: "openai", name: "OpenAI", description: "GPT models via the OpenAI API." },
  { id: "anthropic", name: "Anthropic", description: "Claude models via the Anthropic API." },
  { id: "google", name: "Google", description: "Gemini models via the Google AI API." },
  { id: "none", name: "None", description: "Skip RenAI for now and configure later." },
] as const;

interface WizardStepDef {
  id: string;
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  short: string;
}

const steps: WizardStepDef[] = [
  { id: "welcome", icon: Sparkles, title: "Welcome to RenOS", short: "Welcome" },
  { id: "ai", icon: Brain, title: "Connect your AI provider", short: "AI provider" },
  { id: "notifications", icon: BellRing, title: "Set up notifications", short: "Notifications" },
  { id: "devices", icon: Laptop, title: "Enroll your first device", short: "First device" },
  { id: "plugins", icon: Puzzle, title: "Extend with plugins", short: "Plugins" },
  { id: "backups", icon: Archive, title: "Enable backups", short: "Backups" },
  { id: "updates", icon: RefreshCw, title: "Choose an update policy", short: "Updates" },
  { id: "done", icon: PartyPopper, title: "You're all set", short: "Done" },
];

const DONE_INDEX = steps.length - 1;

function isStepDone(stepId: string, status?: OnboardingStatus): boolean {
  if (!status) return false;
  switch (stepId) {
    case "ai":
      return status.ai_provider_configured;
    case "notifications":
      return status.notifications_configured;
    case "devices":
      return status.device_enrolled;
    case "plugins":
      return status.plugins_installed > 0;
    case "backups":
      return status.backups_enabled;
    default:
      return false;
  }
}

interface ConfettiPiece {
  left: number;
  delay: number;
  duration: number;
  color: string;
  size: number;
}

const CONFETTI_COLORS = [
  "var(--color-accent)",
  "var(--color-success)",
  "var(--color-warning)",
  "var(--color-error)",
  "var(--color-info)",
];

function useConfetti(count: number): ConfettiPiece[] {
  return useMemo(() => {
    return Array.from({ length: count }, (_, i) => ({
      left: (i / count) * 100 + Math.random() * 4,
      delay: Math.random() * 0.8,
      duration: 2.5 + Math.random() * 2,
      color: CONFETTI_COLORS[i % CONFETTI_COLORS.length],
      size: 6 + Math.random() * 6,
    }));
  }, [count]);
}

function Confetti() {
  const pieces = useConfetti(40);
  return (
    <div aria-hidden className="pointer-events-none fixed inset-0 overflow-hidden">
      {pieces.map((p, i) => (
        <span
          key={i}
          className="absolute top-0 rounded-sm"
          style={{
            left: `${p.left}%`,
            width: p.size,
            height: p.size * 1.6,
            backgroundColor: p.color,
            animation: `confetti-fall ${p.duration}s linear ${p.delay}s infinite`,
          }}
        />
      ))}
    </div>
  );
}

function StepShell({
  icon: Icon,
  title,
  description,
  children,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  children?: React.ReactNode;
}) {
  return (
    <div>
      <div className="flex items-center gap-3 mb-6">
        <div className="h-12 w-12 rounded-md bg-[var(--color-accent-tint)] text-[var(--color-accent)] flex items-center justify-center">
          <Icon className="h-6 w-6" />
        </div>
        <div>
          <h2 className="text-headline-2 text-[var(--color-text-primary)]">{title}</h2>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">{description}</p>
        </div>
      </div>
      <div className="space-y-4">{children}</div>
    </div>
  );
}

function InfoLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-1.5 text-body-small font-medium text-[var(--color-accent)] hover:text-[var(--color-accent-hover)]"
    >
      {children}
      <ExternalLink className="h-3.5 w-3.5" />
    </Link>
  );
}

export function OnboardingWizard() {
  const router = useRouter();
  const { status, markComplete, completing } = useOnboarding();
  const { demoStatus, seedDemo } = useDemo();

  const [current, setCurrent] = useState(0);
  const [completingAll, setCompletingAll] = useState(false);
  const [completeError, setCompleteError] = useState<string | null>(null);

  const [provider, setProvider] = useState<string | null>(null);
  const [loadDemo, setLoadDemo] = useState(false);
  const [demoResult, setDemoResult] = useState<DemoSeedResult | null>(null);
  const [demoError, setDemoError] = useState<string | null>(null);

  const demoLoaded = demoStatus.data?.demo_loaded ?? false;

  const goNext = useCallback(() => {
    setCurrent((c) => Math.min(c + 1, DONE_INDEX));
    setCompleteError(null);
  }, []);

  const goBack = useCallback(() => {
    setCurrent((c) => Math.max(c - 1, 0));
    setCompleteError(null);
  }, []);

  const handleSkipAll = useCallback(async () => {
    setCompletingAll(true);
    setCompleteError(null);
    try {
      await markComplete();
    } catch (err) {
      setCompleteError(err instanceof Error ? err.message : "Failed to finish onboarding.");
      setCompletingAll(false);
    }
  }, [markComplete]);

  useEffect(() => {
    if (current === DONE_INDEX) {
      markComplete().catch((err) => {
        setCompleteError(err instanceof Error ? err.message : "Failed to finish onboarding.");
      });
    }
  }, [current, markComplete]);

  const handleDemoToggle = async (checked: boolean) => {
    setLoadDemo(checked);
    setDemoError(null);
    if (!checked) return;
    try {
      const result = await seedDemo.mutateAsync();
      setDemoResult(result);
    } catch (err) {
      setDemoError(err instanceof Error ? err.message : "Failed to load demo data.");
    }
  };

  const demoCount = demoResult
    ? demoResult.devices.length +
      demoResult.workflows.length +
      demoResult.alert_rules.length +
      demoResult.dashboards.length +
      demoResult.notifications.length +
      demoResult.marketplace.length +
      demoResult.installed_plugins.length +
      demoResult.jobs.length
    : 0;

  const isDone = current === DONE_INDEX;

  return (
    <div className="min-h-screen bg-[var(--color-bg)] lg:flex">
      <Confetti />

      {/* Mobile progress strip */}
      <div className="lg:hidden border-b border-[var(--color-border)] px-6 py-4">
        <div className="flex items-center gap-2 overflow-x-auto">
          {steps.map((step, i) => {
            const done = i < current || isStepDone(step.id, status);
            const active = i === current;
            return (
              <div key={step.id} className="flex items-center gap-2 flex-shrink-0">
                <div
                  className={cn(
                    "h-7 w-7 rounded-full flex items-center justify-center text-caption transition-colors",
                    done && "bg-[var(--color-accent)] text-white",
                    !done && active && "bg-[var(--color-accent-tint)] text-[var(--color-accent)] border border-[var(--color-accent)]",
                    !done && !active && "bg-[var(--color-surface)] text-[var(--color-text-tertiary)] border border-[var(--color-border)]",
                  )}
                >
                  {done ? <Check className="h-3.5 w-3.5" /> : i + 1}
                </div>
                {i < steps.length - 1 && <div className={cn("h-px w-4", i < current ? "bg-[var(--color-accent)]" : "bg-[var(--color-border)]")} />}
              </div>
            );
          })}
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col px-6 py-8 lg:px-16 lg:py-12 max-w-3xl">
        <div className="flex items-center justify-between">
          <span className="text-headline-4 text-[var(--color-text-primary)]">RenOS</span>
          <button
            onClick={handleSkipAll}
            disabled={completingAll || isDone}
            className="btn-ghost text-caption text-[var(--color-text-tertiary)]"
          >
            {completingAll ? "Finishing..." : "Skip all"}
          </button>
        </div>

        <div className="flex-1 flex flex-col justify-center py-8">
          <div key={current} className="animate-fade-in-up">
            {current === 0 && (
              <StepShell
                icon={Sparkles}
                title="Welcome to RenOS"
                description="Your devices. One intelligent system. This short setup will get you ready in a few minutes."
              >
                <div className="card p-6">
                  <p className="text-body text-[var(--color-text-secondary)]">
                    RenOS unifies your servers and edge devices into a single control plane, with
                    AI-assisted operations, automation workflows, monitoring, and alerts.
                  </p>
                </div>

                {!demoLoaded && (
                  <div className="card p-5 flex items-start justify-between gap-4">
                    <div>
                      <p className="text-body-small font-medium text-[var(--color-text-primary)]">
                        Load demo data
                      </p>
                      <p className="text-body-small text-[var(--color-text-secondary)] mt-1">
                        Demo data gives you a pre-populated system to explore.
                      </p>
                    </div>
                    <button
                      role="switch"
                      aria-checked={loadDemo}
                      aria-label="Load demo data"
                      onClick={() => handleDemoToggle(!loadDemo)}
                      disabled={seedDemo.isPending}
                      className={cn(
                        "relative h-6 w-11 rounded-full transition-colors flex-shrink-0",
                        loadDemo ? "bg-[var(--color-accent)]" : "bg-[var(--color-surface)] border border-[var(--color-border)]",
                      )}
                    >
                      <span
                        className={cn(
                          "absolute top-0.5 h-5 w-5 rounded-full transition-transform",
                          loadDemo ? "translate-x-[22px] bg-white" : "translate-x-0.5 bg-[var(--color-text-tertiary)]",
                        )}
                      />
                    </button>
                  </div>
                )}

                {demoLoaded && !demoResult && (
                  <div className="card p-5 flex items-center gap-3">
                    <CheckCircle2 className="h-5 w-5 text-[var(--color-success)] flex-shrink-0" />
                    <p className="text-body-small text-[var(--color-text-secondary)]">
                      Demo data is already loaded.
                    </p>
                  </div>
                )}

                {seedDemo.isPending && (
                  <div className="flex items-center gap-2 text-body-small text-[var(--color-text-secondary)]">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Seeding demo data...
                  </div>
                )}

                {demoError && (
                  <div className="flex items-center gap-2 p-3 rounded-sm bg-[rgba(255,59,48,0.08)] text-body-small text-[var(--color-error)]">
                    <AlertCircle className="h-4 w-4 flex-shrink-0" />
                    {demoError}
                  </div>
                )}

                {demoResult && (
                  <div className="card p-5 flex items-center gap-3">
                    <CheckCircle2 className="h-5 w-5 text-[var(--color-success)] flex-shrink-0" />
                    <p className="text-body-small text-[var(--color-text-secondary)]">
                      Demo data loaded — {demoResult.devices.length} devices, {demoResult.workflows.length} workflow
                      {demoResult.workflows.length !== 1 ? "s" : ""}, {demoResult.alert_rules.length} alert rules,{" "}
                      {demoCount} total entities.
                    </p>
                  </div>
                )}
              </StepShell>
            )}

            {current === 1 && (
              <StepShell
                icon={Brain}
                title="Connect your AI provider"
                description="RenAI powers AI-assisted operations, chat, and automation across your fleet."
              >
                <p className="text-body text-[var(--color-text-secondary)]">
                  Choose which provider you plan to use. You can finish configuration at any time —
                  RenAI will be enabled as soon as a provider is configured on the server.
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {providers.map((p) => (
                    <button
                      key={p.id}
                      onClick={() => setProvider(p.id)}
                      className={cn(
                        "card text-left transition-colors",
                        provider === p.id
                          ? "border-[var(--color-accent)] bg-[var(--color-accent-tint)]"
                          : "hover:border-[var(--color-border)]",
                      )}
                    >
                      <p className="text-body-small font-medium text-[var(--color-text-primary)]">{p.name}</p>
                      <p className="text-caption text-[var(--color-text-secondary)] mt-1">{p.description}</p>
                    </button>
                  ))}
                </div>
                <div className="card p-4 flex flex-wrap items-center gap-x-6 gap-y-2">
                  <p className="text-body-small text-[var(--color-text-secondary)]">Configure it now:</p>
                  <InfoLink href="/ai">Open AI Assistant</InfoLink>
                  <InfoLink href="/settings">Server settings</InfoLink>
                </div>
              </StepShell>
            )}

            {current === 2 && (
              <StepShell
                icon={BellRing}
                title="Set up notifications"
                description="Get alerts about device health, job failures, and critical events."
              >
                <p className="text-body text-[var(--color-text-secondary)]">
                  Connect an email (SMTP) or messaging provider so RenOS can reach you when something
                  needs attention. Alerts, workflows, and automation all rely on notifications.
                </p>
                <div className="card p-4 flex flex-wrap items-center gap-x-6 gap-y-2">
                  <p className="text-body-small text-[var(--color-text-secondary)]">Configure providers:</p>
                  <InfoLink href="/notifications">Open notifications</InfoLink>
                </div>
              </StepShell>
            )}

            {current === 3 && (
              <StepShell
                icon={Laptop}
                title="Enroll your first device"
                description="Install the RenOS agent on a server or edge device to bring it under your control."
              >
                <ul className="space-y-3">
                  {[
                    "Install the RenOS agent on your device",
                    "Run the enrollment command from the devices page",
                    "The agent authenticates with a device secret and appears online",
                  ].map((item, i) => (
                    <li key={item} className="flex items-start gap-3 text-body text-[var(--color-text-secondary)]">
                      <span className="h-5 w-5 rounded-full bg-[var(--color-surface)] border border-[var(--color-border)] text-caption text-[var(--color-text-tertiary)] flex items-center justify-center flex-shrink-0 mt-0.5">
                        {i + 1}
                      </span>
                      {item}
                    </li>
                  ))}
                </ul>
                <div className="card p-4 flex flex-wrap items-center gap-x-6 gap-y-2">
                  <p className="text-body-small text-[var(--color-text-secondary)]">Enroll a device:</p>
                  <InfoLink href="/devices">Go to devices</InfoLink>
                </div>
              </StepShell>
            )}

            {current === 4 && (
              <StepShell
                icon={Puzzle}
                title="Extend with plugins"
                description="Install plugins to add capabilities like proxies, monitoring, and integrations."
              >
                <p className="text-body text-[var(--color-text-secondary)]">
                  The RenOS marketplace has verified plugins you can install with one click. Plugins can
                  add workers, routes, dashboards, and notification channels.
                </p>
                <div className="card p-4 flex flex-wrap items-center gap-x-6 gap-y-2">
                  <p className="text-body-small text-[var(--color-text-secondary)]">Browse plugins:</p>
                  <InfoLink href="/marketplace">Open marketplace</InfoLink>
                </div>
              </StepShell>
            )}

            {current === 5 && (
              <StepShell
                icon={Archive}
                title="Enable backups"
                description="Protect your data with automated, verifiable backups."
              >
                <p className="text-body text-[var(--color-text-secondary)]">
                  RenOS can take encrypted backups of its database and files on a schedule. Backups are
                  verified regularly so you always know they restore cleanly.
                </p>
                <div className="card p-4 flex flex-wrap items-center gap-x-6 gap-y-2">
                  <p className="text-body-small text-[var(--color-text-secondary)]">Manage backups:</p>
                  <InfoLink href="/settings">Open settings</InfoLink>
                </div>
              </StepShell>
            )}

            {current === 6 && (
              <StepShell
                icon={RefreshCw}
                title="Choose an update policy"
                description="Decide how RenOS should roll out new versions."
              >
                <p className="text-body text-[var(--color-text-secondary)]">
                  Updates are planned and reversible. You can pick a channel and control when rollouts
                  happen from the updates page.
                </p>
                <div className="card p-4 flex flex-wrap items-center gap-x-6 gap-y-2">
                  <p className="text-body-small text-[var(--color-text-secondary)]">Manage updates:</p>
                  <InfoLink href="/updates">Open updates</InfoLink>
                </div>
              </StepShell>
            )}

            {isDone && (
              <div className="text-center">
                <div className="mx-auto h-20 w-20 rounded-full bg-[var(--color-success)] bg-opacity-10 flex items-center justify-center mb-6">
                  <PartyPopper className="h-10 w-10 text-[var(--color-success)]" />
                </div>
                <h2 className="text-headline-1 text-[var(--color-text-primary)] mb-3">You&apos;re all set!</h2>
                <p className="text-body text-[var(--color-text-secondary)] max-w-md mx-auto mb-2">
                  Your RenOS instance is ready. Head to the dashboard to see your fleet, or check the
                  health dashboard to review every subsystem.
                </p>

                {completeError && (
                  <div className="mx-auto max-w-sm flex flex-col items-center gap-2 p-3 rounded-sm bg-[rgba(255,59,48,0.08)] text-body-small text-[var(--color-error)] mb-4">
                    <span className="flex items-center gap-2">
                      <AlertCircle className="h-4 w-4 flex-shrink-0" />
                      {completeError}
                    </span>
                    <button onClick={handleSkipAll} disabled={completingAll} className="btn-secondary text-body-small px-3 h-7">
                      {completingAll ? "Retrying..." : "Retry"}
                    </button>
                  </div>
                )}

                <div className="flex flex-wrap items-center justify-center gap-3 mt-6">
                  <button
                    onClick={() => router.push("/dashboard")}
                    className="btn-primary"
                    disabled={completing}
                  >
                    <Rocket className="h-4 w-4" />
                    Go to dashboard
                  </button>
                  <button onClick={() => router.push("/health")} className="btn-secondary">
                    <HeartPulse className="h-4 w-4" />
                    View health
                  </button>
                </div>

                {completing && (
                  <p className="text-caption text-[var(--color-text-tertiary)] mt-4 flex items-center justify-center gap-2">
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    Finalizing setup...
                  </p>
                )}
              </div>
            )}
          </div>
        </div>

        {!isDone && (
          <div className="flex items-center justify-between gap-4 pt-4 border-t border-[var(--color-border)]">
            <button onClick={goBack} disabled={current === 0} className="btn-ghost">
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>
            <div className="flex items-center gap-3">
              <button onClick={goNext} className="btn-ghost">
                Skip for now
              </button>
              <button onClick={goNext} className="btn-primary">
                {current === 0 ? "Get started" : "Continue"}
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Desktop progress rail */}
      <aside className="hidden lg:flex lg:flex-col lg:w-72 lg:border-l lg:border-[var(--color-border)] lg:px-8 lg:py-12 flex-shrink-0">
        <p className="text-caption uppercase tracking-wider text-[var(--color-text-tertiary)] px-2 pb-4">
          Setup progress
        </p>
        <nav className="space-y-1">
          {steps.map((step, i) => {
            const done = i < current || isStepDone(step.id, status);
            const active = i === current;
            const Icon = step.icon;
            return (
              <button
                key={step.id}
                onClick={() => {
                  if (i < current) setCurrent(i);
                }}
                disabled={!done && i !== current}
                className={cn(
                  "w-full flex items-center gap-3 px-2 py-2 rounded-sm text-body text-left transition-colors",
                  active
                    ? "bg-[var(--color-accent-tint)] text-[var(--color-accent)] font-medium"
                    : "text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)]",
                )}
              >
                <span
                  className={cn(
                    "h-7 w-7 rounded-full flex items-center justify-center flex-shrink-0",
                    done
                      ? "bg-[var(--color-accent)] text-white"
                      : active
                        ? "bg-[var(--color-accent-tint)] border border-[var(--color-accent)] text-[var(--color-accent)]"
                        : "bg-[var(--color-surface)] border border-[var(--color-border)] text-[var(--color-text-tertiary)]",
                  )}
                >
                  {done ? <Check className="h-3.5 w-3.5" /> : <Icon className="h-3.5 w-3.5" />}
                </span>
                <span className={cn(done && !active && "line-through opacity-60")}>{step.short}</span>
              </button>
            );
          })}
        </nav>
        <div className="mt-8 p-4 rounded-sm bg-[var(--color-surface)] border border-[var(--color-border)]">
          <p className="text-caption text-[var(--color-text-tertiary)] leading-5">
            You can skip any step and finish setup later from the dashboard.
          </p>
        </div>
      </aside>
    </div>
  );
}
