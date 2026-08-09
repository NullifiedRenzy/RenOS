"use client";

import { Languages } from "lucide-react";
import { locales, localeNames, type Locale } from "@/lib/i18n/translations";
import { useLocale } from "@/lib/i18n/use-locale";

export function LocaleToggle() {
  const { locale, setLocale, t } = useLocale();

  return (
    <div className="flex items-center gap-1.5" title={localeNames[locale]}>
      <Languages size={16} className="text-[var(--color-text-tertiary)]" aria-hidden="true" />
      <select
        aria-label={t("locale.label")}
        value={locale}
        onChange={(e) => setLocale(e.target.value as Locale)}
        className="h-8 rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] text-caption text-[var(--color-text-secondary)] px-1.5 focus:border-[var(--color-accent)] focus:outline-none"
      >
        {locales.map((l) => (
          <option key={l} value={l}>
            {localeNames[l]}
          </option>
        ))}
      </select>
    </div>
  );
}
