import { describe, expect, it } from "vitest";
import {
  formatBytes,
  formatDate,
  formatDateTime,
  formatNumber,
  formatPercent,
  formatRelative,
  formatTime,
  getLocale,
  setLocale,
} from "@/lib/i18n/locale";

const NOW = new Date("2026-08-06T12:00:00Z");

describe("locale", () => {
  it("tracks the current locale and rejects unknown ones", () => {
    setLocale("en");
    expect(getLocale()).toBe("en");
    setLocale("xx" as never);
    expect(getLocale()).toBe("en");
  });

  it("formats absolute dates and times", () => {
    const iso = "2026-08-06T12:34:00Z";
    expect(formatDate(iso)).toBeTruthy();
    expect(formatDateTime(iso)).toContain("2026");
    expect(formatTime(iso)).toMatch(/\d{1,2}:\d{2}/);
  });

  it("formats numbers and percentages", () => {
    expect(formatNumber(1234.5)).toBeTruthy();
    expect(formatNumber(1234.5, { maximumFractionDigits: 0 })).not.toContain(".");
    expect(formatPercent(0.426)).toContain("%");
  });

  it("formats byte sizes with the right units", () => {
    expect(formatBytes(0)).toContain("B");
    expect(formatBytes(1023)).toContain("B");
    expect(formatBytes(1024)).toContain("KB");
    expect(formatBytes(1024 * 1024 * 5)).toContain("MB");
    expect(formatBytes(1024 ** 3 * 2.5)).toContain("GB");
    expect(formatBytes(-1)).toBe("—");
  });

  it("returns relative labels within the day and absolute dates after", () => {
    const justNow = new Date(NOW.getTime() - 30_000).toISOString();
    const minuteAgo = new Date(NOW.getTime() - 61_000).toISOString();
    const fiftyNineMinutesAgo = new Date(NOW.getTime() - 59 * 60_000).toISOString();
    const hourAgo = new Date(NOW.getTime() - 3_661_000).toISOString();
    const threeHoursAgo = new Date(NOW.getTime() - 3 * 3_600_000).toISOString();
    const lastWeek = new Date(NOW.getTime() - 7 * 86_400_000).toISOString();

    expect(formatRelative(justNow, NOW)).toBe("Just now");
    expect(formatRelative(fiftyNineMinutesAgo, NOW)).toMatch(/^\d+m ago$/);
    expect(formatRelative(hourAgo, NOW)).toMatch(/^\d+h ago$/);
    expect(formatRelative(lastWeek, NOW)).toBeTruthy();
  });
});
