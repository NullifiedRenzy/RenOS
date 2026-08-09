"use client";

import { useUiStore } from "@/stores/ui-store";
import { useLocale } from "@/lib/i18n/use-locale";
import type { TranslationKey } from "@/lib/i18n/translations";

export function ThemeToggle() {
  const { theme, setTheme } = useUiStore();
  const { t } = useLocale();

  function cycle() {
    const next = theme === "light" ? "dark" : theme === "dark" ? "system" : "light";
    setTheme(next);
  }

  const icon = theme === "dark" ? "\u{1F319}" : theme === "light" ? "\u2600\uFE0F" : "\u{1F31B}";
  const labelKey: TranslationKey =
    theme === "dark" ? "theme.dark" : theme === "light" ? "theme.light" : "theme.system";
  const actionKey: TranslationKey =
    theme === "dark"
      ? "theme.switchToSystem"
      : theme === "light"
        ? "theme.switchToDark"
        : "theme.switchToLight";

  return (
    <button
      onClick={cycle}
      className="btn-ghost text-body-small gap-1"
      title={t("theme.themeLabel", { label: t(labelKey) })}
      aria-label={t(actionKey)}
    >
      <span className="text-base" aria-hidden="true">{icon}</span>
      <span className="hidden md:inline">{t(labelKey)}</span>
    </button>
  );
}
