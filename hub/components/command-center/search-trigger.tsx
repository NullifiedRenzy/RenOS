"use client";

import { Search } from "lucide-react";
import { useCommandCenterStore } from "@/stores/command-center-store";
import { useLocale } from "@/lib/i18n/use-locale";

export function SearchTrigger() {
  const open = useCommandCenterStore((s) => s.open);
  const { t } = useLocale();

  return (
    <button
      onClick={() => open("search")}
      className="flex items-center gap-2 h-8 px-3 rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] text-caption text-[var(--color-text-tertiary)] hover:border-[var(--color-accent)] hover:text-[var(--color-text-primary)] transition-colors w-56"
      aria-label={t("search.ariaLabel")}
    >
      <Search size={14} aria-hidden="true" />
      <span className="flex-1 text-left">{t("search.placeholder")}</span>
      <kbd className="kbd hidden sm:inline-flex" aria-hidden="true">Ctrl K</kbd>
    </button>
  );
}
