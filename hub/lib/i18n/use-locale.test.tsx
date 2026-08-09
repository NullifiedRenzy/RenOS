import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { LocaleProvider, useLocale } from "@/lib/i18n/use-locale";

function Harness() {
  const { locale, setLocale, t } = useLocale();
  return (
    <div>
      <p data-testid="locale">{locale}</p>
      <p data-testid="label">{t("common.save")}</p>
      <p data-testid="interpolated">{t("time.minutesAgo", { count: 5 })}</p>
      <p data-testid="fallback">{t("missing.key" as never)}</p>
      <button type="button" onClick={() => setLocale("en")}>
        Set
      </button>
    </div>
  );
}

describe("LocaleProvider", () => {
  it("renders children and exposes the default locale", () => {
    render(
      <LocaleProvider>
        <Harness />
      </LocaleProvider>,
    );
    expect(screen.getByTestId("locale").textContent).toBe("en");
  });

  it("translates keys and interpolates parameters", () => {
    render(
      <LocaleProvider>
        <Harness />
      </LocaleProvider>,
    );
    expect(screen.getByTestId("label").textContent).toBe("Save");
    expect(screen.getByTestId("interpolated").textContent).toBe("5m ago");
  });

  it("falls back to the raw key for unknown translations", () => {
    render(
      <LocaleProvider>
        <Harness />
      </LocaleProvider>,
    );
    expect(screen.getByTestId("fallback").textContent).toBe("missing.key");
  });

  it("persists the chosen locale to localStorage", () => {
    render(
      <LocaleProvider>
        <Harness />
      </LocaleProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Set" }));
    expect(window.localStorage.getItem("renos.locale")).toBe("en");
  });

  it("restores a stored locale on mount", () => {
    window.localStorage.setItem("renos.locale", "en");
    render(
      <LocaleProvider>
        <Harness />
      </LocaleProvider>,
    );
    expect(screen.getByTestId("locale").textContent).toBe("en");
  });
});

describe("useLocale", () => {
  it("throws when used outside the provider", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    function Broken() {
      useLocale();
      return null;
    }
    expect(() => render(<Broken />)).toThrow("useLocale must be used within a LocaleProvider");
    consoleSpy.mockRestore();
  });
});
