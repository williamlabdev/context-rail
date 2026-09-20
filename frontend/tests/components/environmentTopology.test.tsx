import { fireEvent, render, screen } from "@testing-library/react";
import { EnvironmentTopology } from "../../src/components/EnvironmentTopology";
import { ProjectContextPage } from "../../src/pages/ProjectContextPage";
import type { TopologyState } from "../../src/api/topology";
import type { TopologyController } from "../../src/state/useTopology";
import type { ProjectEntry } from "../../src/api/projectRegistry";

const state: TopologyState = {
  kind: "EnvironmentTopologyState",
  schema_version: "environment-topology/v1",
  project_id: "order-operations-portal",
  current_version: 2,
  versions: [
    {
      version: 1, previous_version: 0, created_at: "2026-09-20T12:00:00Z", actor: "system", origin: "manifest",
      change: { operation: "BOOTSTRAP", material: false, reason: "imported from project.yaml", impact: "baseline" },
      environments: [], config_hash: "sha256:aaaa",
    },
    {
      version: 2, previous_version: 1, created_at: "2026-09-20T12:00:01Z", actor: "operator-1", origin: "operator",
      change: {
        operation: "EDIT", environment_id: "staging", material: true, reason: "move staging service",
        impact: "material change to target_ref; dependent decisions and approvals are marked STALE",
        fields_changed: [{ field: "target_ref", from: "cloud-run/a", to: "cloud-run/b", material: true }],
      },
      environments: [
        { id: "development", display_name: "development", type: "development", sequence: 1, target_ref: "local", owner: "", required_evidence: ["local-test"], approver_policy: "", status: "ACTIVE", action: null },
        { id: "testing", display_name: "testing", type: "testing", sequence: 2, target_ref: "ci", owner: "", required_evidence: ["test"], approver_policy: "", status: "RETIRED", action: null },
        { id: "staging", display_name: "staging", type: "staging", sequence: 3, target_ref: "cloud-run/b", owner: "", required_evidence: ["smoke"], approver_policy: "", status: "ACTIVE", action: null },
        { id: "production", display_name: "production", type: "production", sequence: 4, target_ref: "protected/read-only", owner: "", required_evidence: ["release-approval"], approver_policy: "", status: "ACTIVE", action: "blocked-in-demo", protection: "READ_ONLY_BLOCKED_IN_P0" },
      ],
      config_hash: "sha256:bbbbbbbbbbbbbbbb",
    },
  ],
  invalidations: [
    { decision_id: "DR-001", environment_id: "staging", topology_version: 2, status: "STALE", reason: "topology v2 (edit): target_ref changed", at: "2026-09-20T12:00:01Z" },
  ],
  audit: [],
};

function controller(overrides: Partial<TopologyController> = {}): TopologyController {
  return {
    status: "READY", state, error: null, busy: false,
    reload: vi.fn(),
    add: vi.fn().mockResolvedValue(true),
    edit: vi.fn().mockResolvedValue(true),
    reorder: vi.fn().mockResolvedValue(true),
    retire: vi.fn().mockResolvedValue(true),
    restore: vi.fn().mockResolvedValue(true),
    ...overrides,
  };
}

describe("EnvironmentTopology", () => {
  it("renders the current version, its diff, statuses and protection", () => {
    render(<EnvironmentTopology controller={controller()} />);
    expect(screen.getByText("Topology v2")).toBeInTheDocument();
    expect(screen.getByTestId("topology-last-change")).toHaveTextContent("EDIT · staging");
    expect(screen.getByTestId("topology-last-change")).toHaveTextContent("target_ref: cloud-run/a → cloud-run/b");
    expect(screen.getByTestId("environment-row-testing")).toHaveTextContent("RETIRED");
    expect(screen.getByTestId("environment-row-production")).toHaveTextContent("BLOCKED IN P0");
    expect(screen.getByTestId("topology-invalidations")).toHaveTextContent("DR-001");
  });

  it("requires a reason before quick actions and passes it through", async () => {
    const ctl = controller();
    render(<EnvironmentTopology controller={ctl} />);
    fireEvent.click(screen.getByRole("button", { name: "Move staging up" }));
    expect(screen.getByTestId("topology-reason-missing")).toBeInTheDocument();
    expect(ctl.reorder).not.toHaveBeenCalled();

    fireEvent.change(screen.getByTestId("topology-quick-reason"), { target: { value: "staging before testing" } });
    fireEvent.click(screen.getByRole("button", { name: "Move staging up" }));
    expect(ctl.reorder).toHaveBeenCalledWith(["development", "staging", "testing", "production"], "staging before testing");
  });

  it("offers Restore for retired nodes and blocks retiring production", () => {
    const ctl = controller();
    render(<EnvironmentTopology controller={ctl} />);
    const testingRow = screen.getByTestId("environment-row-testing");
    expect(testingRow).toHaveTextContent("Restore");
    const productionRow = screen.getByTestId("environment-row-production");
    const retire = Array.from(productionRow.querySelectorAll("button")).find((button) => button.textContent === "Retire");
    expect(retire).toBeDisabled();
  });

  it("submits an add form as a new version request", () => {
    const ctl = controller();
    render(<EnvironmentTopology controller={ctl} />);
    fireEvent.click(screen.getByRole("button", { name: "Add environment" }));
    const form = screen.getByTestId("environment-form-add");
    fireEvent.change(form.querySelector("input[name=id]")!, { target: { value: "uat" } });
    fireEvent.change(form.querySelector("select[name=type]")!, { target: { value: "uat" } });
    fireEvent.change(form.querySelector("input[name=target_ref]")!, { target: { value: "cloud-run/uat" } });
    fireEvent.change(form.querySelector("input[name=required_evidence]")!, { target: { value: "test, uat-signoff" } });
    fireEvent.change(form.querySelector("input[name=reason]")!, { target: { value: "add uat" } });
    fireEvent.submit(form);
    expect(ctl.add).toHaveBeenCalledWith(expect.objectContaining({ id: "uat", type: "uat", target_ref: "cloud-run/uat", required_evidence: ["test", "uat-signoff"], reason: "add uat" }));
  });

  it("surfaces a version conflict with a reload action", () => {
    render(<EnvironmentTopology controller={controller({ error: { code: "TOPOLOGY_VERSION_CONFLICT", message: "expected v1 but current is v2" } })} />);
    expect(screen.getByTestId("topology-error")).toHaveTextContent("TOPOLOGY_VERSION_CONFLICT");
    expect(screen.getByRole("button", { name: "Reload topology" })).toBeInTheDocument();
  });
});

describe("ProjectContextPage with topology", () => {
  const entry: ProjectEntry = {
    project: { id: "order-operations-portal", name: "Order Operations Portal", root: "demo/order-operations-portal" },
    repositories: [], services: [], environments: [],
    documents: [],
    decisions: [{ decision_id: "DR-001", status: "ACCEPTED_FOR_STAGING" }, { decision_id: "DR-009", status: "DRAFT" }],
    readiness: {}, context: { status: "DERIVED" }, read_only: true, observed_at: "2026-09-20T00:00:00Z",
  };

  it("overlays STALE on decisions invalidated by the topology without touching others", () => {
    render(<ProjectContextPage entry={entry} topology={controller()} />);
    expect(screen.getByText("DR-001 · ACCEPTED_FOR_STAGING → STALE (topology v2)")).toBeInTheDocument();
    expect(screen.getByText("DR-009 · DRAFT")).toBeInTheDocument();
  });
});
