import { useEffect } from "react";
import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Providers } from "./providers";
import { useAuthStore } from "@/stores/auth-store";
import { api } from "@/lib/api-client";

vi.mock("@/lib/api-client", () => ({ api: { setAccessToken: vi.fn() } }));
vi.mock("@/stores/ui-store", () => {
  const state = { initTheme: vi.fn() };
  return { useUiStore: (select: (value: typeof state) => unknown) => select(state) };
});

afterEach(() => {
  cleanup();
  localStorage.clear();
  sessionStorage.clear();
  useAuthStore.setState({ user: null, token: null, isAuthenticated: false, hasHydrated: false });
  vi.clearAllMocks();
});

describe("authentication initialization", () => {
  it.each(["localStorage", "sessionStorage"] as const)("restores %s before child requests can run", (storage) => {
    window[storage].setItem("renos_auth", JSON.stringify({ user: { id: "test", role: "admin" }, token: "saved-token" }));
    const request = vi.fn();
    function ProtectedPage() {
      useEffect(() => {
        expect(api.setAccessToken).toHaveBeenCalledWith("saved-token");
        expect(useAuthStore.getState().isAuthenticated).toBe(true);
        request();
      }, []);
      return <div>Dashboard</div>;
    }
    render(<Providers><ProtectedPage /></Providers>);
    expect(request).toHaveBeenCalledOnce();
  });
});
