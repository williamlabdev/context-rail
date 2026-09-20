import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { render, screen } from "@testing-library/react";
import { LocaleProvider, translate } from "../../src/i18n";
import { zhTW } from "../../src/i18n/zh-TW";
import { StatusBadge } from "../../src/components/StatusBadge";

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const full = join(dir, name);
    if (statSync(full).isDirectory()) return name === "i18n" ? [] : sourceFiles(full);
    return /\.(tsx?|ts)$/.test(name) ? [full] : [];
  });
}

// UI-16: every static message that goes through t() has a zh-TW entry, so a
// zh-TW screen never falls back to English copy (proper names and technical
// identifiers such as Cloud Run, sha256 or Work Order are allowed inside).
describe("workspace locale", () => {
  it("covers every t() key with a zh-TW translation", () => {
    const keys = new Set<string>(["Decision inputs", "Local development", "Cloud testing", "Staging inputs", "Staging verification", "Production threshold", "stage:decision", "stage:development", "stage:work-order", "stage:staging"]);
    for (const file of sourceFiles(join(__dirname, "../../src"))) {
      const source = readFileSync(file, "utf8");
      for (const match of source.matchAll(/\bt\(\s*"((?:[^"\\]|\\.)*)"/g)) keys.add(match[1]);
    }
    const missing = [...keys].filter((key) => !(key in zhTW.messages));
    expect(missing).toEqual([]);
    expect(keys.size).toBeGreaterThan(200);
  });

  it("never translates data values: ids, paths, digests and user input pass through untouched (UI-19)", () => {
    for (const value of ["order-operations-portal", "cloud-run/order-operations-portal-staging", "sha256:5555555555555555", "REL-002", "docs/operations/runbook.md", "CHG-003 · Manual order review"]) {
      expect(translate("zh-TW", value)).toBe(value);
      expect(translate("en", value)).toBe(value);
    }
    expect(translate("zh-TW", "Topology v{version}", { version: 7 })).toBe("拓撲 v7");
    expect(translate("en", "Topology v{version}", { version: 7 })).toBe("Topology v7");
  });

  it("shows a human label and keeps the machine code readable (UI-18)", () => {
    const english = render(<LocaleProvider initial="en"><StatusBadge status="GATE_BLOCKED" /></LocaleProvider>);
    expect(screen.getByText("Gate blocked")).toBeInTheDocument();
    expect(screen.getByText("GATE_BLOCKED")).toBeInTheDocument();
    english.unmount();
    const chinese = render(<LocaleProvider initial="zh-TW"><StatusBadge status="GATE_BLOCKED" /></LocaleProvider>);
    expect(screen.getByText("閘門阻擋")).toBeInTheDocument();
    expect(screen.getByText("GATE_BLOCKED")).toBeInTheDocument();
    chinese.unmount();
    // An identifier is not a status: it renders verbatim, in any locale.
    render(<LocaleProvider initial="zh-TW"><StatusBadge status="RR-001" /></LocaleProvider>);
    expect(screen.getByText("RR-001")).toBeInTheDocument();
  });
});
