import { defaultLocale, translations, type Locale } from "@/lib/i18n/translations";

let currentLocale: string = defaultLocale;

export function getLocale(): string {
  return currentLocale;
}

export function setLocale(locale: string): void {
  if (locale in translations) currentLocale = locale as Locale;
}

const dtCache = new Map<string, Intl.DateTimeFormat>();
const nfCache = new Map<string, Intl.NumberFormat>();

function dateTimeFormat(options: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
  const key = `${currentLocale}:${JSON.stringify(options)}`;
  let fmt = dtCache.get(key);
  if (!fmt) {
    fmt = new Intl.DateTimeFormat(currentLocale, options);
    dtCache.set(key, fmt);
  }
  return fmt;
}

function numberFormat(options: Intl.NumberFormatOptions = {}): Intl.NumberFormat {
  const key = `${currentLocale}:${JSON.stringify(options)}`;
  let fmt = nfCache.get(key);
  if (!fmt) {
    fmt = new Intl.NumberFormat(currentLocale, options);
    nfCache.set(key, fmt);
  }
  return fmt;
}

export function formatDateTime(iso: string | number | Date): string {
  return dateTimeFormat({
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso));
}

export function formatDate(iso: string | number | Date): string {
  return dateTimeFormat({ year: "numeric", month: "short", day: "numeric" }).format(new Date(iso));
}

export function formatTime(iso: string | number | Date): string {
  return dateTimeFormat({ hour: "2-digit", minute: "2-digit" }).format(new Date(iso));
}

export function formatNumber(value: number, options: Intl.NumberFormatOptions = {}): string {
  return numberFormat(options).format(value);
}

export function formatPercent(value: number, maximumFractionDigits = 1): string {
  return numberFormat({ style: "percent", maximumFractionDigits }).format(value);
}

const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB", "PB"];

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes < 1024) return `${formatNumber(bytes)} B`;
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), BYTE_UNITS.length - 1);
  const value = bytes / Math.pow(1024, i);
  const digits = value >= 100 ? 0 : value >= 10 ? 1 : 2;
  return `${formatNumber(value, { maximumFractionDigits: digits })} ${BYTE_UNITS[i]}`;
}

export function formatRelative(iso: string | number | Date, now: Date = new Date()): string {
  const date = new Date(iso);
  const diff = now.getTime() - date.getTime();
  if (diff < 60_000) return translations[currentLocale as Locale].time.justNow;
  if (diff < 3_600_000)
    return translations[currentLocale as Locale].time.minutesAgo.replace(
      "{count}",
      String(Math.floor(diff / 60_000)),
    );
  if (diff < 86_400_000)
    return translations[currentLocale as Locale].time.hoursAgo.replace(
      "{count}",
      String(Math.floor(diff / 3_600_000)),
    );
  return formatDate(date);
}
