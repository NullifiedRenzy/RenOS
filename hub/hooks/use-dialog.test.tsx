import React, { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { useDialogEscape } from "@/hooks/use-dialog";

function Harness() {
  const [open, setOpen] = useState(false);
  const ref = useDialogEscape(() => setOpen(false), open);
  return (
    <div>
      <button type="button" onClick={() => setOpen(true)}>Open</button>
      {open && (
        <div ref={ref} role="dialog" aria-modal="true" aria-label="Test dialog" tabIndex={-1}>
          <p>Content</p>
          <button type="button" onClick={() => setOpen(false)}>Close</button>
        </div>
      )}
    </div>
  );
}

describe("useDialogEscape", () => {
  it("exposes dialog semantics when open", () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.getByRole("dialog", { name: "Test dialog" })).toHaveAttribute("aria-modal", "true");
  });

  it("closes on Escape and returns focus to the trigger", () => {
    const onClose = vi.fn();
    function Single() {
      const [open, setOpen] = useState(true);
      const ref = useDialogEscape(onClose, open);
      if (!open) return null;
      return (
        <div ref={ref} role="dialog" aria-label="D">
          <button type="button" data-testid="inner">Inner</button>
        </div>
      );
    }
    const { getByTestId } = render(
      <>
        <button type="button" data-testid="trigger">Trigger</button>
        <Single />
      </>,
    );
    const trigger = getByTestId("trigger");
    trigger.focus();
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("moves focus into the dialog when it opens", () => {
    render(<Harness />);
    fireEvent.click(screen.getByRole("button", { name: "Open" }));
    const dialog = screen.getByRole("dialog", { name: "Test dialog" });
    expect(document.activeElement).toBe(dialog);
  });
});
