import { describe, expect, it } from "vitest";
import { getCommands, COMMAND_GROUPS } from "@/lib/commands";

describe("command registry", () => {
  it("exposes a non-empty command set with unique ids", () => {
    const commands = getCommands();
    expect(commands.length).toBeGreaterThan(20);
    const ids = commands.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("includes the required palette actions", () => {
    const ids = new Set(getCommands().map((c) => c.id));
    for (const id of [
      "open-dashboard",
      "open-devices",
      "restart-device",
      "run-workflow",
      "open-plugins",
      "create-secret",
      "open-terminal",
      "open-logs",
      "open-notifications",
    ]) {
      expect(ids.has(id), `missing command ${id}`).toBe(true);
    }
  });

  it("every command has title, keywords, icon, group and run", () => {
    for (const c of getCommands()) {
      expect(c.title.length).toBeGreaterThan(0);
      expect(typeof c.keywords).toBe("string");
      expect(c.icon).toBeTruthy();
      expect(COMMAND_GROUPS).toContain(c.group);
      expect(typeof c.run).toBe("function");
    }
  });

  it("device and workflow commands declare a picker", () => {
    const cmds = getCommands();
    expect(cmds.find((c) => c.id === "restart-device")?.pick).toBe("devices");
    expect(cmds.find((c) => c.id === "run-workflow")?.pick).toBe("workflows");
    expect(cmds.find((c) => c.id === "open-terminal")?.pick).toBe("devices");
  });
});
