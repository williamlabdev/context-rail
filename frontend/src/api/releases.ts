// Client for the promotion / release API (VS-006).

export interface GateResult { gate: string; change_id?: string; status: string; detail: string }

export interface ReleaseChange {
  change_id: string; title: string; decision_id: string; decision_version: number; work_order_id: string; work_order_hash: string;
  candidate_id: string; run_id: string; run_started_by: string; branch: string; head_commit: string; review_gate: string;
  source_snapshot_hash: string; target_environment_id: string; candidate_accepted_by: string;
}

export interface EnvironmentConfig {
  environment_id: string; type: string; target_ref: string; required_evidence: string[]; approver_policy: string; topology_version: number; config_hash: string;
}

export interface BuildEvidence { image_digest: string; image_ref: string; build_id: string; source_commit: string; includes_commits: string[]; evidence_ref: string; observed_at: string }

export interface Manifest {
  release_id: string; project_id: string; changes: ReleaseChange[]; transition: string; source_environment_id: string;
  environment: EnvironmentConfig; build: BuildEvidence | null; policy_version: string; manifest_hash: string;
}

export interface Approval { actor: string; role: string; at: string; decision: string; reason: string; manifest_hash: string; expires_at: string; waiver?: string }
export interface SmokeResult { status: string; evidence_ref?: string; at?: string }
export interface DeploymentRecord {
  attempt_id: string; idempotency_key: string; operation_id: string; source: string; revision: string; service_url: string;
  deployed_digest: string; deployed_target_ref: string; deployed_config_hash: string; deployed_at: string; deployed_by: string;
  smoke: SmokeResult | null; gates: GateResult[]; outcome: string; recorded_at: string;
}
export interface Receipt {
  kind: string; schema_version: string; receipt_id: string; release_id: string; project_id: string; status: string; transition: string;
  environment: EnvironmentConfig; changes: ReleaseChange[]; build: BuildEvidence; approval: Approval; deployment: DeploymentRecord;
  manifest_hash: string; evidence_refs: string[]; issued_at: string; receipt_hash: string;
}
export interface Release {
  release_id: string; project_id: string; status: string; manifest: Manifest; gates: GateResult[]; verdict: string;
  approval: Approval | null; deployments: DeploymentRecord[]; receipt: Receipt | null; reason: string; created_by: string; created_at: string; updated_at: string;
}
export interface ReleaseView { release: Release; live_gates: GateResult[]; staleness: { stale: boolean; reason?: string } }
export interface ReleaseList { kind: "ReleaseList"; schema_version: string; project_id: string; releases: ReleaseView[] }

export class ReleaseRequestError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(`${code}: ${message}`);
    this.name = "ReleaseRequestError";
  }
}

export interface CreateReleaseInput { reason: string; actor?: string; change_ids: string[]; target_environment_id: string }
export interface BuildInput { reason: string; actor?: string; image_digest: string; image_ref?: string; build_id?: string; source_commit: string; includes_commits?: string[]; evidence_ref?: string }
export interface ApprovalInput { actor: string; role?: string; decision: "APPROVE" | "REJECT"; rationale: string }
export interface DeploymentInput {
  reason: string; actor?: string; idempotency_key: string; operation_id?: string; source?: string; revision: string; service_url?: string;
  deployed_digest: string; deployed_target_ref: string; deployed_config_hash?: string; deployed_at?: string; smoke: SmokeResult | null;
}

async function readJSON<T>(response: Response): Promise<T> {
  const value = (await response.json()) as Record<string, unknown>;
  if (!response.ok) {
    throw new ReleaseRequestError(String(value.code ?? "RELEASE_ERROR"), String(value.message ?? "Release request failed."), response.status);
  }
  return value as unknown as T;
}

function releasesPath(projectID: string): string {
  return `/v1/projects/${encodeURIComponent(projectID)}/releases`;
}

async function post<T>(fetcher: typeof fetch, path: string, body: unknown): Promise<T> {
  return readJSON<T>(await fetcher(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }));
}

export async function fetchReleases(projectID: string, fetcher: typeof fetch = fetch): Promise<ReleaseList> {
  return readJSON<ReleaseList>(await fetcher(releasesPath(projectID)));
}
export async function createRelease(projectID: string, input: CreateReleaseInput, fetcher: typeof fetch = fetch): Promise<ReleaseView> {
  return post<ReleaseView>(fetcher, releasesPath(projectID), input);
}
export async function recordBuild(projectID: string, releaseID: string, input: BuildInput, fetcher: typeof fetch = fetch): Promise<ReleaseView> {
  return post<ReleaseView>(fetcher, `${releasesPath(projectID)}/${encodeURIComponent(releaseID)}/build`, input);
}
export async function approveRelease(projectID: string, releaseID: string, input: ApprovalInput, fetcher: typeof fetch = fetch): Promise<ReleaseView> {
  return post<ReleaseView>(fetcher, `${releasesPath(projectID)}/${encodeURIComponent(releaseID)}/approval`, input);
}
export async function recordDeployment(projectID: string, releaseID: string, input: DeploymentInput, fetcher: typeof fetch = fetch): Promise<ReleaseView> {
  return post<ReleaseView>(fetcher, `${releasesPath(projectID)}/${encodeURIComponent(releaseID)}/deployment`, input);
}

export function newIdempotencyKey(): string {
  const random = typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : Math.random().toString(36).slice(2);
  return `deploy-${random}`;
}
