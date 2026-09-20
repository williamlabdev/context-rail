// Client for the Change Decision Pack API (VS-004).

export interface AcceptanceCriterion { id: string; text: string }

export interface ChangeRequest {
  title: string;
  objective: string;
  scope_included: string[];
  scope_excluded: string[];
  acceptance_criteria: AcceptanceCriterion[];
  allowed_paths: string[];
  forbidden_actions: string[];
  target_environment_id: string;
  business_constraints: Record<string, string>;
  requested_by: string;
}

export interface MissingInput { field: string; owner_role: string; reason: string }

export interface EnvironmentRef {
  id: string; type: string; sequence: number; target_ref: string; required_evidence: string[]; status: string; protection?: string;
}

export interface Evaluation {
  status: "DECISION_READY" | "NEEDS_INPUT" | string;
  missing_inputs: MissingInput[];
  observations: string[];
  source_snapshot_hash: string;
  topology_version: number;
  topology_config_hash: string;
  target_environment: EnvironmentRef | null;
  source_environment: EnvironmentRef | null;
  allowed_transition: string;
  project_context_status: string;
  decision_inputs_readiness: string;
  evaluated_at: string;
}

export interface Option {
  id: string; title: string; summary: string; cost_drivers: string[]; risks: string[]; recommended: boolean; advisor_source: string;
}

export interface ChangeVersion {
  version: number; created_at: string; actor: string; reason: string; request: ChangeRequest; evaluation: Evaluation; options: Option[]; unknowns: string[];
}

export interface Lineage {
  decision_id: string; decision_version: number; change_id: string; project_id: string;
  source_snapshot_hash: string; topology_version: number; topology_config_hash: string; policy_version: string;
}

export interface HumanDecision { actor: string; role: string; at: string; decision: string; reason: string }

export interface DecisionRecord {
  decision_id: string; project_id: string; change_id: string; change_version: number; version: number; status: string; risk_level: string;
  selected_option: string; alternatives: Option[]; rationale: string; objective: string;
  accepted_scope: string[]; out_of_scope: string[]; allowed_paths: string[]; forbidden_actions: string[];
  acceptance_criteria: AcceptanceCriterion[]; business_constraints: Record<string, string>; unknowns: string[];
  target_environment: EnvironmentRef; source_environment: EnvironmentRef | null; allowed_transition: string; production_action: string;
  source_snapshot_hash: string; topology_version: number; topology_config_hash: string; policy_version: string; evidence_refs: string[];
  human_decision: HumanDecision; created_at: string;
}

export interface BriefSection { heading: string; lines: string[] }

export interface ChangeDecisionBrief { artifact_type: string; lineage: Lineage; audience: string; headline: string; sections: BriefSection[]; rendered_at: string }

export interface AgentContextPack {
  artifact_type: string; schema_version: string; lineage: Lineage; status: string; objective: string;
  environment_topology: Record<string, unknown>; business_constraints: Record<string, string>;
  accepted_scope: string[]; out_of_scope: string[]; allowed_paths: string[]; forbidden_actions: string[];
  acceptance_tests: Array<{ id: string; expected: string }>; required_checks: string[]; unknowns: string[]; evidence_refs: string[]; rendered_at: string;
}

export interface AgentWorkOrder {
  kind: string; schema_version: string; work_order_id: string; lineage: Lineage; status: string; human_gate: string; issuer: string;
  repository: string; base_branch: string; target_branch: string; target_environment_id: string; allowed_transition: string;
  allowed_paths: string[]; forbidden_actions: string[]; acceptance_ids: string[]; required_checks: string[]; output_paths: string[];
  issued_at: string; expires_at: string; work_order_hash: string;
}

export interface AgentRun {
  kind?: string; schema_version?: string; run_id: string; work_order_id: string; work_order_hash: string;
  agent: { name: string; provider: string; model: string; version: string };
  adapter?: string; started_by: string; started_at?: string; finished_at?: string; declared_commands?: string[]; declared_changed_paths?: string[];
}
export interface CheckResult { name: string; status: string; evidence_ref?: string; observed_at?: string }
export interface ReviewEvidence { reviewer: string; kind: "human" | "ai" | string; verdict: string; evidence_ref?: string; at?: string }
export interface Observation {
  source?: string; repository?: string; branch: string; base_branch?: string; base_commit?: string; head_commit: string; pull_request?: string;
  changed_paths: string[]; checks: CheckResult[]; reviews: ReviewEvidence[];
  compensating_controls?: { declared: boolean; evidence_ref?: string; reason?: string } | null; observed_at?: string;
}
export interface GateResult { gate: string; status: string; detail: string; paths?: string[]; missing?: string[] }
export interface Violation { rule: string; path?: string; detail: string }
export interface Candidate {
  candidate_id: string; change_id: string; work_order_id: string; lineage: Lineage; run: AgentRun; observation: Observation;
  gates: GateResult[]; violations: Violation[]; verdict: string; status: string; recommended_action: string;
  human_decision: HumanDecision | null; submitted_by: string; reason: string; created_at: string;
}
export interface ReadBackRequest { source: string; repository: string; base: string; head: string }
export interface SubmitCandidateInput { reason: string; actor?: string; run: AgentRun; observation: Observation; read_back?: ReadBackRequest }
export interface CandidateDecisionInput { actor: string; role?: string; decision: "ACCEPT" | "REJECT"; rationale: string }

export interface Change {
  change_id: string; project_id: string; title: string; status: string; current_version: number;
  versions: ChangeVersion[]; decisions: DecisionRecord[]; work_order: AgentWorkOrder | null; candidates: Candidate[]; created_at: string; updated_at: string;
}

export interface ChangeView {
  change: Change;
  decision: DecisionRecord | null;
  brief: ChangeDecisionBrief | null;
  agent_context_pack: AgentContextPack | null;
  work_order: AgentWorkOrder | null;
  staleness: { stale: boolean; reason?: string };
}

export interface ChangeList { kind: "ChangeList"; schema_version: string; project_id: string; changes: ChangeView[] }

export class ChangeRequestError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(`${code}: ${message}`);
    this.name = "ChangeRequestError";
  }
}

export interface CreateChangeInput { reason: string; actor?: string; request: Partial<ChangeRequest> & { title: string } }
export interface InputsPatch {
  reason: string; actor?: string;
  objective?: string; scope_included?: string[]; scope_excluded?: string[]; acceptance_criteria?: AcceptanceCriterion[];
  allowed_paths?: string[]; forbidden_actions?: string[]; target_environment_id?: string; business_constraints?: Record<string, string>;
}
export interface DecideInput { actor: string; role?: string; decision: "ACCEPT" | "REJECT"; selected_option?: string; rationale: string; risk_level?: string; reason?: string }
export interface WorkOrderInput { reason: string; actor?: string; issuer?: string; target_branch?: string }

async function readJSON<T>(response: Response): Promise<T> {
  const value = (await response.json()) as Record<string, unknown>;
  if (!response.ok) {
    throw new ChangeRequestError(String(value.code ?? "CHANGE_ERROR"), String(value.message ?? "Change request failed."), response.status);
  }
  return value as unknown as T;
}

export function currentVersion(change: Change): ChangeVersion {
  return change.versions[change.versions.length - 1];
}

function changesPath(projectID: string): string {
  return `/v1/projects/${encodeURIComponent(projectID)}/changes`;
}

async function post<T>(fetcher: typeof fetch, path: string, body: unknown): Promise<T> {
  return readJSON<T>(await fetcher(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }));
}

export async function fetchChanges(projectID: string, fetcher: typeof fetch = fetch): Promise<ChangeList> {
  return readJSON<ChangeList>(await fetcher(changesPath(projectID)));
}

export async function fetchChange(projectID: string, changeID: string, fetcher: typeof fetch = fetch): Promise<ChangeView> {
  return readJSON<ChangeView>(await fetcher(`${changesPath(projectID)}/${encodeURIComponent(changeID)}`));
}

export async function createChange(projectID: string, input: CreateChangeInput, fetcher: typeof fetch = fetch): Promise<ChangeView> {
  return post<ChangeView>(fetcher, changesPath(projectID), input);
}

export async function supplyInputs(projectID: string, changeID: string, input: InputsPatch, fetcher: typeof fetch = fetch): Promise<ChangeView> {
  return post<ChangeView>(fetcher, `${changesPath(projectID)}/${encodeURIComponent(changeID)}/inputs`, input);
}

export async function decideChange(projectID: string, changeID: string, input: DecideInput, fetcher: typeof fetch = fetch): Promise<ChangeView> {
  return post<ChangeView>(fetcher, `${changesPath(projectID)}/${encodeURIComponent(changeID)}/decision`, input);
}

export async function compileWorkOrder(projectID: string, changeID: string, input: WorkOrderInput, fetcher: typeof fetch = fetch): Promise<ChangeView> {
  return post<ChangeView>(fetcher, `${changesPath(projectID)}/${encodeURIComponent(changeID)}/work-order`, input);
}

export async function submitCandidate(projectID: string, changeID: string, input: SubmitCandidateInput, fetcher: typeof fetch = fetch): Promise<ChangeView> {
  return post<ChangeView>(fetcher, `${changesPath(projectID)}/${encodeURIComponent(changeID)}/candidates`, input);
}

export async function decideCandidate(projectID: string, changeID: string, candidateID: string, input: CandidateDecisionInput, fetcher: typeof fetch = fetch): Promise<ChangeView> {
  return post<ChangeView>(fetcher, `${changesPath(projectID)}/${encodeURIComponent(changeID)}/candidates/${encodeURIComponent(candidateID)}/decision`, input);
}
