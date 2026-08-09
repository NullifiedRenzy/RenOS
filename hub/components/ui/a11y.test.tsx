import React from "react";
import { describe, expect, it, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import axe from "axe-core";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Tabs } from "@/components/ui/tabs";
import { Dialog } from "@/components/ui/dialog";
import { Table } from "@/components/ui/table";
import { SkipLink } from "@/components/ui/skip-link";
import { LocaleProvider } from "@/lib/i18n/use-locale";

const RULES = [
  "aria-allowed-attr",
  "aria-hidden-body",
  "aria-required-attr",
  "aria-required-children",
  "aria-required-parent",
  "aria-roles",
  "aria-valid-attr-value",
  "aria-valid-attr",
  "button-name",
  "duplicate-id",
  "frame-title",
  "heading-order",
  "html-has-lang",
  "html-lang-valid",
  "image-alt",
  "label",
  "link-name",
  "nested-interactive",
  "select-name",
  "scope-attr-valid",
  "tabindex",
];

async function assertNoViolations(name: string) {
  const results = await axe.run(document.body, {
    runOnly: { type: "rule", values: RULES },
  });
  const violations = results.violations.filter((v) => v.nodes.length > 0);
  const summary = violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(", ")}`);
  expect(summary, `${name} must have no axe violations`).toEqual([]);
}

afterEach(() => cleanup());

describe("accessibility (axe-core)", () => {
  it("Button has an accessible name", async () => {
    render(<Button variant="primary">Save</Button>);
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
    await assertNoViolations("Button");
  });

  it("Input label is associated and error is announced", async () => {
    render(<Input label="Hostname" error="Required" />);
    expect(screen.getByLabelText("Hostname")).toBeInTheDocument();
    expect(screen.getByRole("alert").textContent).toBe("Required");
    await assertNoViolations("Input with error");
  });

  it("Select has an accessible name", async () => {
    render(
      <Select
        label="Role"
        options={[{ value: "a", label: "Admin" }]}
        value="a"
        onChange={() => {}}
      />,
    );
    expect(screen.getByRole("combobox")).toBeInTheDocument();
    await assertNoViolations("Select");
  });

  it("Badge renders text content", async () => {
    render(<Badge tone="error" dot>offline</Badge>);
    expect(screen.getByText("offline")).toBeInTheDocument();
    await assertNoViolations("Badge");
  });

  it("Tabs expose valid tab roles and aria-selected", async () => {
    render(
      <>
        <Tabs
          tabs={[
            { id: "a", label: "Alpha" },
            { id: "b", label: "Beta" },
          ]}
          active="a"
          onChange={() => {}}
        />
        <div id="panel-a" role="tabpanel" aria-labelledby="tab-a">alpha</div>
        <div id="panel-b" role="tabpanel" aria-labelledby="tab-b">beta</div>
      </>,
    );
    expect(screen.getByRole("tablist")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Alpha" })).toHaveAttribute("aria-selected", "true");
    await assertNoViolations("Tabs");
  });

  it("Dialog is a labelled modal dialog", async () => {
    render(
      <Dialog open onClose={() => {}} title="Confirm" description="Are you sure?">
        <p>Body content</p>
      </Dialog>,
    );
    expect(screen.getByRole("dialog", { name: "Confirm" })).toBeInTheDocument();
    expect(screen.getByLabelText("Close dialog")).toBeInTheDocument();
    await assertNoViolations("Dialog");
  });

  it("Table headers use scope", async () => {
    render(
      <Table
        columns={[
          { key: "name", header: "Name", cell: (r: string) => r },
          { key: "status", header: "Status", cell: (r: string) => r },
        ]}
        rows={["renos-core", "online"]}
        rowKey={(r) => r}
      />,
    );
    await assertNoViolations("Table");
  });

  it("Skip link is a labelled link", async () => {
    render(
      <LocaleProvider>
        <SkipLink />
      </LocaleProvider>,
    );
    expect(screen.getByRole("link", { name: "Skip to main content" })).toBeInTheDocument();
    await assertNoViolations("SkipLink");
  });
});
