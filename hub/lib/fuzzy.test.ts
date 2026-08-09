import { describe, expect, it } from "vitest";
import { fuzzyMatch, fuzzyRank } from "@/lib/fuzzy";

describe("fuzzyMatch", () => {
  it("matches exact string with highest score", () => {
    const r = fuzzyMatch("devices", "devices");
    expect(r.matched).toBe(true);
    expect(r.score).toBeGreaterThan(0);
  });

  it("matches case-insensitively", () => {
    expect(fuzzyMatch("DEV", "devices").matched).toBe(true);
    expect(fuzzyMatch("dev", "DEVICES").matched).toBe(true);
  });

  it("matches subsequences (VSCode style)", () => {
    expect(fuzzyMatch("dvc", "devices").matched).toBe(true);
    expect(fuzzyMatch("wfl", "workflows").matched).toBe(true);
  });

  it("returns -1 when characters are missing", () => {
    expect(fuzzyMatch("xyzzy", "devices").matched).toBe(false);
    expect(fuzzyMatch("devicezzz", "device").matched).toBe(false);
  });

  it("empty query never matches", () => {
    const r = fuzzyMatch("", "devices");
    expect(r.matched).toBe(false);
  });

  it("rejects when query longer than target", () => {
    expect(fuzzyMatch("devicesssssss", "device").matched).toBe(false);
  });

  it("prefers prefix matches over scattered ones", () => {
    const prefix = fuzzyMatch("dev", "devices").score;
    const scattered = fuzzyMatch("dvc", "devices").score;
    expect(prefix).toBeGreaterThan(scattered);
  });

  it("scores consecutive matches higher", () => {
    const consecutive = fuzzyMatch("worf", "workflow").score;
    const gapped = fuzzyMatch("wffl", "workflow").score;
    expect(consecutive).toBeGreaterThan(gapped);
  });
});

describe("fuzzyRank", () => {
  const items = [
    { name: "devices" },
    { name: "device groups" },
    { name: "workflows" },
    { name: "vault" },
  ];

  it("returns best matches sorted by score", () => {
    const ranked = fuzzyRank(items, "dev", (i) => i.name);
    expect(ranked.length).toBeGreaterThanOrEqual(2);
    expect(ranked[0].name).toBe("devices");
  });

  it("returns limited results", () => {
    const ranked = fuzzyRank(items, "e", (i) => i.name, 2);
    expect(ranked.length).toBeLessThanOrEqual(2);
  });

  it("empty query returns first N items unsorted", () => {
    const ranked = fuzzyRank(items, "", (i) => i.name, 2);
    expect(ranked).toHaveLength(2);
    expect(ranked[0].name).toBe("devices");
  });

  it("no matches returns empty", () => {
    expect(fuzzyRank(items, "qqqqq", (i) => i.name)).toEqual([]);
  });
});
