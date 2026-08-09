import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CommandCenter } from "@/components/command-center/command-center";
import { useCommandCenterStore } from "@/stores/command-center-store";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

vi.mock("@/lib/api-client", () => ({
  api: {
    search: vi.fn().mockResolvedValue({
      query: "web",
      total: 1,
      results: {
        devices: [
          { type: "devices", id: "d1", title: "webserver-01", subtitle: "linux · online", href: "/devices/d1" },
        ],
      },
    }),
    getWorkflows: vi.fn().mockResolvedValue({ data: [] }),
    listDevices: vi.fn().mockResolvedValue({ devices: [] }),
    executeWorkflow: vi.fn(),
  },
}));

function renderCommandCenter() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <CommandCenter />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  useCommandCenterStore.setState({ isOpen: false, mode: "search" });
});

describe("CommandCenter", () => {
  it("renders nothing while closed", () => {
    const { container } = renderCommandCenter();
    expect(container).toBeEmptyDOMElement();
  });

  it("opens and shows the search dialog", () => {
    useCommandCenterStore.setState({ isOpen: true });
    renderCommandCenter();
    expect(screen.getByRole("dialog", { name: "Command center" })).toBeInTheDocument();
    expect(screen.getByLabelText("Search the platform")).toBeInTheDocument();
  });

  it("filters commands via the > prefix", async () => {
    useCommandCenterStore.setState({ isOpen: true });
    renderCommandCenter();
    const input = screen.getByLabelText(/Search the platform/);
    fireEvent.change(input, { target: { value: "> open devices" } });
    await waitFor(() => {
      expect(screen.getByText("Open Devices")).toBeInTheDocument();
    });
  });

  it("closes on Escape", () => {
    useCommandCenterStore.setState({ isOpen: true });
    renderCommandCenter();
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(useCommandCenterStore.getState().isOpen).toBe(false);
  });

  it("shows search results grouped under a category header", async () => {
    useCommandCenterStore.setState({ isOpen: true, mode: "search" });
    renderCommandCenter();
    const input = screen.getByLabelText(/Search the platform/);
    fireEvent.change(input, { target: { value: "web" } });
    await waitFor(() => {
      expect(screen.getByText("webserver-01")).toBeInTheDocument();
    });
    expect(screen.getByText("Devices")).toBeInTheDocument();
  });
});
