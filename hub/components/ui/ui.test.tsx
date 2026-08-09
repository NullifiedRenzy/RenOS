import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Tabs } from "@/components/ui/tabs";
import { Dialog } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Table } from "@/components/ui/table";

describe("Button", () => {
  it("renders children and applies variant class", () => {
    const { container } = render(<Button variant="secondary">Save</Button>);
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    expect(container.querySelector("button")?.className).toContain("border");
  });

  it("disables and shows spinner when loading", () => {
    render(<Button isLoading>Save</Button>);
    const btn = screen.getByRole("button", { name: /Save/ }) as HTMLButtonElement;
    expect(btn.disabled).toBe(true);
    expect(screen.getByRole("status")).toBeTruthy();
  });
});

describe("Input", () => {
  it("associates label and error via ids", () => {
    render(<Input label="Hostname" error="Required" />);
    const input = screen.getByLabelText("Hostname");
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert").textContent).toBe("Required");
  });

  it("renders hint when no error", () => {
    render(<Input label="Port" hint="Default 22" />);
    expect(screen.getByText("Default 22")).toBeInTheDocument();
  });
});

describe("Select", () => {
  it("renders options and reflects value", () => {
    render(<Select options={[{ value: "a", label: "Alpha" }, { value: "b", label: "Beta" }]} value="b" onChange={() => {}} />);
    const select = screen.getByRole("combobox") as HTMLSelectElement;
    expect(select.value).toBe("b");
    expect(screen.getByRole("option", { name: "Alpha" })).toBeInTheDocument();
  });
});

describe("Badge", () => {
  it("renders tone class and dot", () => {
    const { container } = render(<Badge tone="success" dot>online</Badge>);
    expect(screen.getByText("online")).toBeInTheDocument();
    expect(container.querySelector("span > span")).toBeTruthy();
  });
});

describe("Tabs", () => {
  const tabs = [
    { id: "a", label: "Alpha" },
    { id: "b", label: "Beta" },
  ];

  it("moves focus and selection with arrow keys", () => {
    const onChange = vi.fn();
    render(<Tabs tabs={tabs} active="a" onChange={onChange} />);
    const first = screen.getByRole("tab", { name: "Alpha" });
    first.focus();
    fireEvent.keyDown(first, { key: "ArrowRight" });
    expect(onChange).toHaveBeenCalledWith("b");
    expect(screen.getByRole("tab", { name: "Beta" })).toHaveFocus();
  });

  it("marks active tab selected", () => {
    render(<Tabs tabs={tabs} active="b" onChange={() => {}} />);
    expect(screen.getByRole("tab", { name: "Beta" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Alpha" })).toHaveAttribute("aria-selected", "false");
  });
});

describe("Dialog", () => {
  it("renders nothing when closed", () => {
    render(<Dialog open={false} onClose={() => {}} title="T">Body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders content and closes on Esc", () => {
    const onClose = vi.fn();
    render(<Dialog open onClose={onClose} title="Confirm">Payload</Dialog>);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  it("closes when overlay clicked", () => {
    const onClose = vi.fn();
    render(<Dialog open onClose={onClose} title="Confirm">Payload</Dialog>);
    fireEvent.click(document.querySelector("[aria-hidden='true']")!);
    expect(onClose).toHaveBeenCalled();
  });
});

describe("Skeleton", () => {
  it("renders multiple lines", () => {
    const { container } = render(<Skeleton lines={3} />);
    expect(container.querySelectorAll(".skeleton").length).toBe(3);
  });
});

describe("Table", () => {
  it("renders rows and cells", () => {
    render(
      <Table
        columns={[
          { key: "name", header: "Name", cell: (r: { name: string }) => r.name },
          { key: "x", header: "Extra", cell: () => "cell" },
        ]}
        rows={[{ name: "webserver" }]}
        rowKey={(r) => r.name}
      />,
    );
    expect(screen.getByText("webserver")).toBeInTheDocument();
    expect(screen.getByText("cell")).toBeInTheDocument();
  });

  it("shows empty message", () => {
    render(
      <Table
        columns={[{ key: "name", header: "Name", cell: (r: { name: string }) => r.name }]}
        rows={[]}
        rowKey={(r) => r.name}
        emptyMessage="Nothing here"
      />,
    );
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });
});
