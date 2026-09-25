import { fireEvent, render, screen } from "@testing-library/react";
import { ChangesPanel } from "../../src/components/ChangesPanel";
import type { ChangeView } from "../../src/api/changes";
import type { ChangesController } from "../../src/state/useChanges";
import type { TopologyEnvironment } from "../../src/api/topology";

const environments: TopologyEnvironment[] = [
  { id: "testing", display_name: "testing", type: "testing", sequence: 2, target_ref: "ci", owner: "", required_evidence: [], approver_policy: "", status: "ACTIVE", action: null },
  { id: "staging", display_name: "staging", type: "staging", sequence: 3, target_ref: "cloud-run/s", owner: "", required_evidence: ["smoke"], approver_policy: "", status: "ACTIVE", action: null },
  { id: "old", display_name: "old", type: "other", sequence: 4, target_ref: "x", owner: "", required_evidence: [], approver_policy: "", status: "RETIRED", action: null },
];

const lineage = { decision_id: "DEC-001", decision_version: 1, change_id: "CHG-001", project_id: "p", source_snapshot_hash: "sha256:abcdef1234567890", topology_version: 1, topology_config_hash: "sha256:cfg", policy_version: "project-workspace-v1" };

const needsInput: ChangeView = {
  change: {
    change_id: "CHG-001", project_id: "p", title: "Attachments", status: "NEEDS_INPUT", current_version: 1, created_at: "t", updated_at: "t", decisions: [], work_order: null, candidates: [],
    versions: [{
      version: 1, created_at: "t", actor: "a", reason: "r",
      request: { title: "Attachments", objective: "Upload", scope_included: [], scope_excluded: [], acceptance_criteria: [], allowed_paths: [], forbidden_actions: [], target_environment_id: "staging", business_constraints: {}, requested_by: "a" },
      evaluation: {
        status: "NEEDS_INPUT", observations: [], source_snapshot_hash: "sha256:abcdef1234567890", topology_version: 1, topology_config_hash: "sha256:cfg",
        target_environment: null, source_environment: null, allowed_transition: "", project_context_status: "DERIVED", decision_inputs_readiness: "READY", evaluated_at: "t",
        missing_inputs: [
          { field: "acceptance_criteria", owner_role: "requester", reason: "needs a testable expectation" },
          { field: "business_constraints.expected_monthly_volume", owner_role: "business_owner", reason: "cost stays UNKNOWN" },
        ],
      },
      options: [{ id: "minimal_reversible_slice", title: "Minimal reversible slice", summary: "s", cost_drivers: [], risks: [], recommended: true, advisor_source: "rule-advisor" }],
      unknowns: ["expected_monthly_volume not declared; cost status UNKNOWN"],
    }],
  },
  decision: null, brief: null, agent_context_pack: null, work_order: null, staleness: { stale: false },
};

const accepted: ChangeView = {
  ...needsInput,
  change: {
    ...needsInput.change, status: "ACCEPTED",
    versions: [{ ...needsInput.change.versions[0], evaluation: { ...needsInput.change.versions[0].evaluation, status: "DECISION_READY", missing_inputs: [] } }],
  },
  decision: {
    decision_id: "DEC-001", project_id: "p", change_id: "CHG-001", change_version: 1, version: 1, status: "ACCEPTED_FOR_DEVELOPMENT", risk_level: "low",
    selected_option: "minimal_reversible_slice", alternatives: [], rationale: "ok", objective: "Upload", accepted_scope: [], out_of_scope: [], allowed_paths: ["a.go"],
    forbidden_actions: ["deploy:production"], acceptance_criteria: [{ id: "AC-001", text: "x" }], business_constraints: {}, unknowns: [],
    target_environment: { id: "staging", type: "staging", sequence: 3, target_ref: "cloud-run/s", required_evidence: ["smoke"], status: "ACTIVE" }, source_environment: null,
    allowed_transition: "testing-to-staging", production_action: "forbidden", source_snapshot_hash: "sha256:abcdef1234567890", topology_version: 1, topology_config_hash: "sha256:cfg",
    policy_version: "project-workspace-v1", evidence_refs: [], human_decision: { actor: "founder-001", role: "sa", at: "t", decision: "ACCEPTED", reason: "ok" }, created_at: "t",
  },
  brief: {
    artifact_type: "change_decision_brief", schema_version: "change-decision-brief/v2", lineage, audience: "people", state: "ACCEPTED",
    owner_summary: "Reviewers can attach files to an order exception.", owner_summary_missing: false, objective: "POST /api/orders/{id}/attachments",
    selected: { id: "minimal_reversible_slice", title: "Minimal reversible slice", summary: "s" },
    route: { source_environment_id: "testing", target_environment_id: "staging", target_type: "staging", target_ref: "cloud-run/x-staging", transition: "testing-to-staging", production_action: "forbidden" },
    risk_level: "low", unknowns: ["project context is STALE"], in_scope: ["attach up to three files"], out_of_scope: ["payment"],
    required_evidence: ["build", "custom-check"], decided_by: { actor: "founder-001", role: "founder", at: "2026-09-21T02:42:18Z", rationale: "smallest reversible slice" },
    alternatives: [{ id: "defer", title: "Defer / keep current process", summary: "wait" }], rendered_at: "t",
  },
  agent_context_pack: {
    artifact_type: "agent_context_pack", schema_version: "agent-context-pack/v1", lineage, status: "ISSUED_FOR_DEVELOPMENT_CANDIDATE", objective: "Upload",
    environment_topology: {}, business_constraints: {}, accepted_scope: [], out_of_scope: [], allowed_paths: ["a.go"], forbidden_actions: ["deploy:production"],
    acceptance_tests: [{ id: "AC-001", expected: "x" }], required_checks: ["smoke"], unknowns: [], evidence_refs: [], rendered_at: "t",
  },
};

function controller(overrides: Partial<ChangesController> = {}): ChangesController {
  return {
    status: "READY", changes: [], selectedID: null, selected: null, error: null, busy: false,
    reload: vi.fn(), select: vi.fn(), create: vi.fn().mockResolvedValue(true), inputs: vi.fn().mockResolvedValue(true),
    decide: vi.fn().mockResolvedValue(true), workOrder: vi.fn().mockResolvedValue(true),
    submitCandidate: vi.fn().mockResolvedValue(true), decideCandidate: vi.fn().mockResolvedValue(true), ...overrides,
  };
}

describe("ChangesPanel", () => {
  it("shows NEEDS_INPUT with owners, no decision form, and only the missing fields in the inputs form", () => {
    render(<ChangesPanel controller={controller({ changes: [needsInput], selectedID: "CHG-001", selected: needsInput })} environments={environments} />);
    const missing = screen.getByTestId("change-missing-inputs");
    expect(missing).toHaveTextContent("acceptance_criteria");
    expect(missing).toHaveTextContent("business_owner");
    expect(screen.queryByTestId("change-decision-form")).not.toBeInTheDocument();
    const form = screen.getByTestId("change-inputs-form");
    expect(form.querySelector("textarea[name=acceptance_criteria]")).toBeInTheDocument();
    expect(form.querySelector("input[name=expected_monthly_volume]")).toBeInTheDocument();
    expect(form.querySelector("textarea[name=allowed_paths]")).not.toBeInTheDocument();
    expect(screen.getByTestId("change-unknowns")).toHaveTextContent("UNKNOWN");
  });

  it("submits supplied inputs as a patch with a reason", () => {
    const ctl = controller({ changes: [needsInput], selectedID: "CHG-001", selected: needsInput });
    render(<ChangesPanel controller={ctl} environments={environments} />);
    const form = screen.getByTestId("change-inputs-form");
    fireEvent.change(form.querySelector("textarea[name=acceptance_criteria]")!, { target: { value: "one\ntwo" } });
    fireEvent.change(form.querySelector("input[name=expected_monthly_volume]")!, { target: { value: "500" } });
    fireEvent.change(form.querySelector("input[name=reason]")!, { target: { value: "supplied" } });
    fireEvent.submit(form);
    expect(ctl.inputs).toHaveBeenCalledWith("CHG-001", expect.objectContaining({
      reason: "supplied", acceptance_criteria: [{ id: "", text: "one" }, { id: "", text: "two" }], business_constraints: { expected_monthly_volume: "500" },
    }));
  });

  it("renders brief and pack with an identical lineage strip and offers the work order", () => {
    render(<ChangesPanel controller={controller({ changes: [accepted], selectedID: "CHG-001", selected: accepted })} environments={environments} />);
    const strips = screen.getAllByTestId("lineage-strip");
    expect(strips).toHaveLength(2);
    expect(strips[0].textContent).toBe(strips[1].textContent);
    expect(screen.getByTestId("change-pack")).toHaveTextContent("deploy:production");
    expect(screen.getByTestId("change-work-order-form")).toBeInTheDocument();
    expect(screen.queryByTestId("change-decision-form")).not.toBeInTheDocument();
  });

  it("leads the brief with the owner summary and keeps machine identity in the traceability block", () => {
    render(<ChangesPanel controller={controller({ changes: [accepted], selectedID: "CHG-001", selected: accepted })} environments={environments} />);
    const brief = screen.getByTestId("change-brief");
    expect(screen.getByTestId("brief-summary")).toHaveTextContent("Reviewers can attach files to an order exception.");
    expect(screen.getByTestId("brief-route")).toHaveTextContent("Minimal reversible slice");
    expect(screen.getByTestId("brief-route")).not.toHaveTextContent("minimal_reversible_slice");
    expect(screen.getByTestId("brief-risk")).toHaveTextContent("project context is STALE");
    expect(screen.getByTestId("brief-out-of-scope")).toHaveTextContent("payment");
    expect(screen.getByTestId("brief-next-gate")).toHaveTextContent("The build succeeded");
    expect(screen.getByTestId("brief-next-gate")).toHaveTextContent("custom-check");
    expect(screen.getByTestId("brief-alternatives")).toHaveTextContent("Defer / keep current process");
    // Order follows the reader's questions: summary before risk before scope before who decided.
    const order = ["brief-summary", "brief-risk", "brief-in-scope", "brief-next-gate", "brief-decided-by", "brief-trace"].map((id) => screen.getByTestId(id));
    for (let index = 1; index < order.length; index++) {
      expect(order[index - 1].compareDocumentPosition(order[index]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }
    const trace = screen.getByTestId("brief-trace");
    expect(trace).toHaveTextContent("sha256:abcdef1234567890");
    expect(trace).toHaveTextContent("POST /api/orders/{id}/attachments");
    expect(trace.hasAttribute("open")).toBe(false);
    expect(brief.textContent?.indexOf("sha256:abcdef1234567890")).toBeGreaterThan(brief.textContent!.indexOf("Who decided"));
  });

  it("says the owner summary is missing on an older decision instead of substituting it", () => {
    const legacy: ChangeView = { ...accepted, brief: { ...accepted.brief!, owner_summary: "", owner_summary_missing: true } };
    render(<ChangesPanel controller={controller({ changes: [legacy], selectedID: "CHG-001", selected: legacy })} environments={environments} />);
    expect(screen.queryByTestId("brief-summary")).not.toBeInTheDocument();
    const missing = screen.getByTestId("brief-summary-missing");
    expect(missing).toHaveTextContent("No plain-language summary was supplied");
    expect(missing).toHaveTextContent("POST /api/orders/{id}/attachments");
  });

  it("marks everything stale and withholds the work order form when the decision is stale", () => {
    const stale: ChangeView = { ...accepted, change: { ...accepted.change, status: "STALE" }, staleness: { stale: true, reason: "topology v2 changed target_ref" }, agent_context_pack: { ...accepted.agent_context_pack!, status: "STALE" } };
    render(<ChangesPanel controller={controller({ changes: [stale], selectedID: "CHG-001", selected: stale })} environments={environments} />);
    expect(screen.getByTestId("change-stale")).toHaveTextContent("topology v2");
    expect(screen.queryByTestId("change-work-order-form")).not.toBeInTheDocument();
  });

  it("excludes retired environments from the create form and submits the request", () => {
    const ctl = controller();
    render(<ChangesPanel controller={ctl} environments={environments} />);
    fireEvent.click(screen.getByRole("button", { name: "New Change" }));
    const form = screen.getByTestId("change-form-create");
    const options = Array.from(form.querySelectorAll("select[name=target_environment_id] option")).map((option) => (option as HTMLOptionElement).value);
    expect(options).toEqual(["", "testing", "staging"]);
    fireEvent.change(form.querySelector("input[name=title]")!, { target: { value: "T" } });
    fireEvent.change(form.querySelector("select[name=target_environment_id]")!, { target: { value: "staging" } });
    fireEvent.change(form.querySelector("textarea[name=allowed_paths]")!, { target: { value: "a.go\nb.go" } });
    fireEvent.change(form.querySelector("input[name=reason]")!, { target: { value: "why" } });
    fireEvent.submit(form);
    expect(ctl.create).toHaveBeenCalledWith(expect.objectContaining({ reason: "why", request: expect.objectContaining({ title: "T", target_environment_id: "staging", allowed_paths: ["a.go", "b.go"] }) }));
  });
});
