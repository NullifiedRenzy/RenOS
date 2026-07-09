"use client";

import { useUiStore } from "@/stores/ui-store";

export function ThemeToggle() {
  const { theme, setTheme } = useUiStore();

  function cycle() {
    const next = theme === "light" ? "dark" : theme === "dark" ? "system" : "light";
    setTheme(next);
  }

  const icon = theme === "dark" ? "\u{1F319}" : theme === "light" ? "\u2600\uFE0F" : "\u{1F31B}";
  const label = theme === "dark" ? "Dark" : theme === "light" ? "Light" : "System";

  return (
    <button
      onClick={cycle}
      className="btn-ghost text-body-small gap-1"
      title={`Theme: ${label}`}
    >
      <span className="text-base">{icon}</span>
      <span className="hidden md:inline">{label}</span>
    </button>
  );
}
