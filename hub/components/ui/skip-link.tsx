"use client";

import { useLocale } from "@/lib/i18n/use-locale";

export function SkipLink() {
  const { t } = useLocale();
  return (
    <a
      href="#main-content"
      className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100] focus:px-4 focus:py-2 focus:rounded-sm focus:bg-[var(--color-accent)] focus:text-white focus:text-body-small focus:font-medium"
    >
      {t("skip.toMainContent")}
    </a>
  );
}
