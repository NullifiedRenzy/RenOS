import { describe, expect, it } from "vitest";
import { translations, locales, defaultLocale, type TranslationKey } from "@/lib/i18n/translations";

const en = translations.en;

describe("translations dictionary", () => {
  it("exposes English as the only locale and default", () => {
    expect(locales).toEqual(["en"]);
    expect(defaultLocale).toBe("en");
  });

  it("contains every key reachable through TranslationKey", () => {
    const keys: TranslationKey[] = [
      "common.save",
      "common.cancel",
      "common.signOut",
      "common.user",
      "nav.mainNavigation",
      "nav.collapseSidebar",
      "nav.expandSidebar",
      "nav.dashboard",
      "nav.allDevices",
      "nav.groups",
      "nav.jobs",
      "nav.schedules",
      "nav.aiAssistant",
      "nav.observability",
      "nav.health",
      "nav.logs",
      "nav.cluster",
      "nav.deploy",
      "nav.updates",
      "nav.vault",
      "nav.notifications",
      "nav.mobile",
      "nav.tagsAndGroups",
      "nav.marketplace",
      "nav.iam",
      "nav.developer",
      "nav.plugins",
      "nav.policies",
      "nav.users",
      "nav.settings",
      "nav.designSystem",
      "skip.toMainContent",
      "theme.light",
      "theme.dark",
      "theme.system",
      "theme.switchToDark",
      "theme.switchToLight",
      "theme.switchToSystem",
      "search.placeholder",
      "search.ariaLabel",
      "locale.label",
      "notifications.title",
      "notifications.markAllRead",
      "notifications.viewAll",
      "notifications.empty",
      "notifications.unread",
      "notifications.types.alert",
      "notifications.types.workflow",
      "notifications.types.job",
      "notifications.types.ai",
      "notifications.types.system",
      "notifications.types.mention",
      "time.never",
      "time.justNow",
    ];
    for (const key of keys) {
      const value = key.split(".").reduce<unknown>((node, part) => {
        if (node && typeof node === "object" && part in (node as Record<string, unknown>)) {
          return (node as Record<string, unknown>)[part];
        }
        return undefined;
      }, en);
      expect(typeof value, `missing translation for ${key}`).toBe("string");
    }
  });

  it("has no empty or whitespace-only strings", () => {
    const walk = (node: Record<string, unknown>, path: string[]) => {
      for (const [k, v] of Object.entries(node)) {
        const full = [...path, k].join(".");
        if (typeof v === "string") {
          expect(v.trim().length, `empty string at ${full}`).toBeGreaterThan(0);
        } else if (v && typeof v === "object") {
          walk(v as Record<string, unknown>, [...path, k]);
        }
      }
    };
    walk(en, []);
  });

  it("covers every InAppNotificationType with a type label", () => {
    const types = ["alert", "workflow", "job", "ai", "system", "mention"];
    for (const type of types) {
      expect(typeof en.notifications.types[type as keyof typeof en.notifications.types]).toBe("string");
    }
  });

  it("keeps interpolation placeholders consistent between keys and values", () => {
    expect(en.time.minutesAgo).toContain("{count}");
    expect(en.time.hoursAgo).toContain("{count}");
    expect(en.notifications.ariaOpen).toContain("{count}");
    expect(en.theme.themeLabel).toContain("{label}");
  });
});
