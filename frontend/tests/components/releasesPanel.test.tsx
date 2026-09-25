import { fireEvent, render, screen } from "@testing-library/react";
import { ReleasesPanel } from "../../src/components/ReleasesPanel";
import type { GateResult, Release, ReleaseView } from "../../src/api/releases";
import type { ReleasesController } from "../../src/state/useReleases";

const environment = { environment_id: "prod-demo", type: "prod-demo", target_ref: "cloud-run/oop-prod-demo", required_evidence: [], approver_policy: "", topology_version: 3, config_hash: "sha256:cfg1234567890abcdef1234567890" };

const passGate: GateResult = { gate: "source_release", status: "PASS", detail: "release is promotable" };
const blockedGate: GateResult = { gate: "new_revision", status: "BLOCKED", detail: "revision must be new" };

function release(overrides: Partial<Release> = {}): Release {
  return {
    release_id: "REL-002", project_id: "p", status: "PROMOTED",
    manifest: {
      release_id: "REL-002", project_id: "p", changes: [], transition: "staging-to-prod-demo", source_environment_id: "staging",
      environment, build: { image_digest: "sha256:7777777777777777777777777777777777777777777777777777777777777777", image_ref: "", build_id: "cb-77", source_commit: "ccc333", includes_commits: [], evidence_ref: "", observed_at: "t" },
      promotion: null, environment_delta: [], policy_version: "v1", manifest_hash: "sha256:manifest1234567890abcdef1234567890",
    },
    gates: [passGate], verdict: "PASS", approval: { actor: "approver-1", role: "release_manager", at: "2026-09-21T02:42:18Z", decision: "APPROVED", reason: "delta reviewed", manifest_hash: "sha256:manifest1234567890abcdef1234567890", expires_at: "2026-09-28T02:42:18Z" },
    deployments: [
      { attempt_id: "REL-002-D01", idempotency_key: "k1", operation_id: "", source: "ui", revision: "", service_url: "", deployed_digest: "sha256:7777777777777777777777777777777777777777777777777777777777777777", deployed_target_ref: "cloud-run/oop-prod-demo", deployed_config_hash: "sha256:cfg", deployed_at: "t", deployed_by: "release-mgr", smoke: null, gates: [blockedGate, passGate], outcome: "PROMOTION_FAILED", recorded_at: "2026-09-21T02:40:00Z" },
      { attempt_id: "REL-002-D02", idempotency_key: "k2", operation_id: "", source: "ui", revision: "oop-prod-demo-00001-ab12", service_url: "https://oop-prod-demo.a.run.app", deployed_digest: "sha256:7777777777777777777777777777777777777777777777777777777777777777", deployed_target_ref: "cloud-run/oop-prod-demo", deployed_config_hash: "sha256:cfg", deployed_at: "t", deployed_by: "release-mgr", smoke: { status: "PASS" }, gates: [passGate], outcome: "PROMOTED", recorded_at: "2026-09-21T02:45:00Z" },
    ],
    receipt: {
      kind: "Receipt", schema_version: "receipt/v1", receipt_id: "RR-002", release_id: "REL-002", project_id: "p", status: "ISSUED", transition: "staging-to-prod-demo",
      environment, changes: [], build: { image_digest: "sha256:7777777777777777777777777777777777777777777777777777777777777777", image_ref: "", build_id: "cb-77", source_commit: "ccc333", includes_commits: [], evidence_ref: "", observed_at: "t" },
      approval: { actor: "approver-1", role: "release_manager", at: "2026-09-21T02:42:18Z", decision: "APPROVED", reason: "delta reviewed", manifest_hash: "sha256:manifest1234567890abcdef1234567890", expires_at: "2026-09-28T02:42:18Z" },
      deployment: { attempt_id: "REL-002-D02", idempotency_key: "k2", operation_id: "", source: "ui", revision: "oop-prod-demo-00001-ab12", service_url: "https://oop-prod-demo.a.run.app", deployed_digest: "sha256:7777777777777777777777777777777777777777777777777777777777777777", deployed_target_ref: "cloud-run/oop-prod-demo", deployed_config_hash: "sha256:cfg", deployed_at: "t", deployed_by: "release-mgr", smoke: { status: "PASS" }, gates: [passGate], outcome: "PROMOTED", recorded_at: "2026-09-21T02:45:00Z" },
      manifest_hash: "sha256:manifest1234567890abcdef1234567890", evidence_refs: [], issued_at: "2026-09-21T02:45:00Z", promotion: null, environment_delta: [],
      previous_receipt_id: "RR-001", receipt_hash: "sha256:receipthash1234567890abcdef1234567890",
    },
    reason: "promote", created_by: "release-mgr", created_at: "t", updated_at: "t",
    ...overrides,
  };
}

function controller(overrides: Partial<ReleasesController> = {}): ReleasesController {
  return {
    status: "READY", releases: [], selectedID: null, selected: null, error: null, busy: false,
    reload: vi.fn(), select: vi.fn(),
    create: vi.fn().mockResolvedValue(true), build: vi.fn().mockResolvedValue(true), approve: vi.fn().mockResolvedValue(true), deployment: vi.fn().mockResolvedValue(true),
    ...overrides,
  };
}

describe("ReleasesPanel release detail (readability review item 5)", () => {
  it("defaults to a short, human-readable summary and hides the full technical record", () => {
    const view: ReleaseView = { release: release(), live_gates: [passGate], staleness: { stale: false } };
    render(<ReleasesPanel controller={controller({ releases: [view], selectedID: "REL-002", selected: view })} changes={[]} environments={[]} />);

    expect(screen.getByTestId("release-view-summary")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("release-view-technical")).toHaveAttribute("aria-pressed", "false");

    // Short, human summary is present; the full technical record is not in the document.
    const summary = screen.getByTestId("release-receipt-summary");
    expect(summary).toHaveTextContent("RR-002");
    expect(summary).toHaveTextContent("approver-1");
    expect(summary.textContent).not.toContain("sha256:7777777777777777777777777777777777777777777777777777777777777777");
    expect(screen.queryByTestId("release-receipt")).not.toBeInTheDocument();
    expect(screen.queryByTestId("release-manifest")).not.toBeInTheDocument();
    expect(screen.queryByTestId("release-heading-technical")).not.toBeInTheDocument();

    // The earlier failed attempt still surfaces in the summary view (any NON-PASS gate is decision-relevant).
    expect(screen.getByTestId("release-attempt-REL-002-D01")).toBeInTheDocument();
    expect(screen.getByTestId("attempt-gates-REL-002-D01-new_revision")).toHaveTextContent("BLOCKED");
    // The later, promoted attempt is technical detail only.
    expect(screen.queryByTestId("release-attempt-REL-002-D02")).not.toBeInTheDocument();
  });

  it("switching to the technical report reveals the full receipt, manifest and hashes; renderer adds no new facts", () => {
    const view: ReleaseView = { release: release(), live_gates: [passGate], staleness: { stale: false } };
    render(<ReleasesPanel controller={controller({ releases: [view], selectedID: "REL-002", selected: view })} changes={[]} environments={[]} />);

    fireEvent.click(screen.getByTestId("release-view-technical"));
    expect(screen.getByTestId("release-view-technical")).toHaveAttribute("aria-pressed", "true");

    const receipt = screen.getByTestId("release-receipt");
    expect(receipt).toHaveTextContent("RR-002");
    expect(receipt.textContent).toContain("sha256:7777777777777777777777777777777777777777777777777777777777777777");
    expect(screen.getByTestId("receipt-chain")).toHaveTextContent("RR-001");
    expect(screen.getByTestId("release-manifest")).toBeInTheDocument();
    expect(screen.getByTestId("release-heading-technical")).toHaveTextContent("staging-to-prod-demo");
    // Every attempt, including the promoted one, is visible for the auditor.
    expect(screen.getByTestId("release-attempt-REL-002-D01")).toBeInTheDocument();
    expect(screen.getByTestId("release-attempt-REL-002-D02")).toBeInTheDocument();
  });

  it("shows only the blocking gates in the summary live gate table, not the passing ones", () => {
    const blockedRelease = release({ status: "GATE_BLOCKED", receipt: null, approval: null, deployments: [] });
    const view: ReleaseView = { release: blockedRelease, live_gates: [passGate, blockedGate], staleness: { stale: false } };
    render(<ReleasesPanel controller={controller({ releases: [view], selectedID: "REL-002", selected: view })} changes={[]} environments={[]} />);

    const gates = screen.getByTestId("release-gates-summary");
    expect(gates).toHaveTextContent("new_revision");
    expect(gates).not.toHaveTextContent("source_release");
    expect(screen.queryByTestId("release-receipt-summary")).not.toBeInTheDocument();
  });
});
