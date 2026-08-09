import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { InboxTab } from "@/components/notifications/inbox-tab";

const notifications = [
  {
    id: "n1",
    user_id: "u1",
    title: "Device offline",
    body: "webserver-01 went offline.",
    type: "alert",
    entity_type: "device",
    entity_id: "d1",
    metadata: {},
    is_read: false,
    read_at: null,
    created_at: new Date().toISOString(),
  },
  {
    id: "n2",
    user_id: "u1",
    title: "Workflow completed",
    body: "Backup finished successfully.",
    type: "workflow",
    entity_type: "workflow",
    entity_id: "wf1",
    metadata: {},
    is_read: true,
    read_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
  },
];

const apiMock = vi.hoisted(() => ({
  getInAppNotifications: vi.fn(),
  getInAppUnreadCount: vi.fn(),
  markInAppRead: vi.fn(),
  markAllInAppRead: vi.fn(),
}));

vi.mock("@/lib/api-client", () => ({
  api: apiMock,
}));

function renderInbox() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <InboxTab />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  apiMock.getInAppNotifications.mockReset();
  apiMock.getInAppUnreadCount.mockReset();
  apiMock.markInAppRead.mockReset();
  apiMock.markAllInAppRead.mockReset();
  apiMock.getInAppNotifications.mockResolvedValue(notifications);
  apiMock.getInAppUnreadCount.mockResolvedValue({ count: 1 });
  apiMock.markInAppRead.mockResolvedValue({ read: true });
  apiMock.markAllInAppRead.mockResolvedValue({ read: true });
});

describe("InboxTab", () => {
  it("renders notifications with type labels", async () => {
    renderInbox();
    await waitFor(() => {
      expect(screen.getByText("Device offline")).toBeInTheDocument();
    });
    expect(screen.getByText("Workflow completed")).toBeInTheDocument();
    expect(screen.getByText("Alert")).toBeInTheDocument();
  });

  it("shows unread badge count", async () => {
    renderInbox();
    await waitFor(() => {
      expect(screen.getByText("1")).toBeInTheDocument();
    });
  });

  it("filters by unread via query param", async () => {
    renderInbox();
    const unreadFilter = await screen.findByRole("button", { name: "Unread" });
    fireEvent.click(unreadFilter);
    await waitFor(() => {
      expect(apiMock.getInAppNotifications).toHaveBeenCalledWith(
        expect.objectContaining({ unread: true }),
      );
    });
  });

  it("marks a single notification read", async () => {
    renderInbox();
    await screen.findByText("Device offline");
    const markBtn = screen.getByLabelText('Mark "Device offline" as read');
    fireEvent.click(markBtn);
    await waitFor(() => {
      expect(apiMock.markInAppRead).toHaveBeenCalledWith("n1");
    });
  });

  it("marks all read", async () => {
    renderInbox();
    await screen.findByText("Device offline");
    fireEvent.click(screen.getByRole("button", { name: /Mark all read/i }));
    await waitFor(() => {
      expect(apiMock.markAllInAppRead).toHaveBeenCalled();
    });
  });

  it("renders empty state", async () => {
    apiMock.getInAppNotifications.mockResolvedValue([]);
    apiMock.getInAppUnreadCount.mockResolvedValue({ count: 0 });
    renderInbox();
    await waitFor(() => {
      expect(screen.getByText("No notifications")).toBeInTheDocument();
    });
  });
});
