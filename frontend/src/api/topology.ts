// Client for the versioned Environment Topology API (VS-003).
//
// Every mutation carries expected_version and a reason; the server refuses
// stale versions with 409 TOPOLOGY_VERSION_CONFLICT and never hard-deletes.

export type EnvironmentStatus = "DRAFT" | "ACTIVE" | "PAUSED" | "RETIRED" | string;

export const standardTypes = ["development", "testing", "staging", "prod-demo", "production", "uat", "other"] as const;

export interface TopologyEnvironment {
  id: string;
  display_name: string;
  type: string;
  sequence: number;
  target_ref: string;
  owner: string;
  required_evidence: string[];
  approver_policy: string;
  status: EnvironmentStatus;
  action: string | null;
  protection?: string;
}

export interface FieldChange {
  field: string;
  from: unknown;
  to: unknown;
  material: boolean;
}

export interface TopologyChange {
  operation: string;
  environment_id?: string;
  fields_changed?: FieldChange[];
  material: boolean;
  reason: string;
  impact: string;
}

export interface TopologyVersion {
  version: number;
  previous_version: number;
  created_at: string;
  actor: string;
  origin: "manifest" | "operator" | string;
  change: TopologyChange;
  environments: TopologyEnvironment[];
  config_hash: string;
}

export interface Invalidation {
  decision_id: string;
  environment_id?: string;
  topology_version: number;
  status: string;
  reason: string;
  at: string;
}

export interface AuditEvent {
  sequence: number;
  at: string;
  actor: string;
  action: string;
  object: string;
  version: number;
  reason: string;
}

export interface TopologyState {
  kind: "EnvironmentTopologyState";
  schema_version: "environment-topology/v1";
  project_id: string;
  current_version: number;
  versions: TopologyVersion[];
  invalidations: Invalidation[];
  audit: AuditEvent[];
}

export interface TopologyError {
  kind: "EnvironmentTopologyError" | "ProjectRegistryError";
  code: string;
  message: string;
}

export class TopologyRequestError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(`${code}: ${message}`);
    this.name = "TopologyRequestError";
  }
}

export interface MutationBase {
  expected_version: number;
  reason: string;
  actor?: string;
}

export interface AddEnvironmentInput extends MutationBase {
  id: string;
  display_name?: string;
  type: string;
  sequence?: number;
  target_ref?: string;
  owner?: string;
  required_evidence?: string[];
  approver_policy?: string;
}

export interface EditEnvironmentInput extends MutationBase {
  display_name?: string;
  type?: string;
  sequence?: number;
  target_ref?: string;
  owner?: string;
  required_evidence?: string[];
  approver_policy?: string;
}

export interface ReorderInput extends MutationBase {
  order: string[];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function parseTopologyState(value: unknown): TopologyState {
  if (!isRecord(value) || value.kind !== "EnvironmentTopologyState") {
    throw new Error("Expected EnvironmentTopologyState");
  }
  if (value.schema_version !== "environment-topology/v1") {
    throw new Error("Expected environment-topology/v1");
  }
  if (!Array.isArray(value.versions) || typeof value.current_version !== "number") {
    throw new Error("EnvironmentTopologyState must carry versions and current_version");
  }
  return value as unknown as TopologyState;
}

export function currentVersion(state: TopologyState): TopologyVersion | undefined {
  return state.versions.find((version) => version.version === state.current_version);
}

/** Latest STALE invalidation per decision id, keyed for overlay rendering. */
export function staleDecisions(state: TopologyState | null): Map<string, Invalidation> {
  const map = new Map<string, Invalidation>();
  for (const invalidation of state?.invalidations ?? []) {
    const existing = map.get(invalidation.decision_id);
    if (!existing || existing.topology_version < invalidation.topology_version) {
      map.set(invalidation.decision_id, invalidation);
    }
  }
  return map;
}

async function readTopologyResponse(response: Response): Promise<TopologyState> {
  const value: unknown = await response.json();
  if (!response.ok) {
    const error = value as Partial<TopologyError>;
    throw new TopologyRequestError(error.code ?? "TOPOLOGY_ERROR", error.message ?? "Topology request failed.", response.status);
  }
  return parseTopologyState(value);
}

function environmentsPath(projectID: string): string {
  return `/v1/projects/${encodeURIComponent(projectID)}/environments`;
}

async function send(fetcher: typeof fetch, method: string, path: string, body: unknown): Promise<TopologyState> {
  return readTopologyResponse(await fetcher(path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  }));
}

export async function fetchTopology(projectID: string, fetcher: typeof fetch = fetch): Promise<TopologyState> {
  return readTopologyResponse(await fetcher(environmentsPath(projectID)));
}

export async function addEnvironment(projectID: string, input: AddEnvironmentInput, fetcher: typeof fetch = fetch): Promise<TopologyState> {
  return send(fetcher, "POST", environmentsPath(projectID), input);
}

export async function editEnvironment(projectID: string, environmentID: string, input: EditEnvironmentInput, fetcher: typeof fetch = fetch): Promise<TopologyState> {
  return send(fetcher, "PATCH", `${environmentsPath(projectID)}/${encodeURIComponent(environmentID)}`, input);
}

export async function reorderEnvironments(projectID: string, input: ReorderInput, fetcher: typeof fetch = fetch): Promise<TopologyState> {
  return send(fetcher, "POST", `${environmentsPath(projectID)}/reorder`, input);
}

export async function retireEnvironment(projectID: string, environmentID: string, input: MutationBase, fetcher: typeof fetch = fetch): Promise<TopologyState> {
  return send(fetcher, "POST", `${environmentsPath(projectID)}/${encodeURIComponent(environmentID)}/retire`, input);
}

export async function restoreEnvironment(projectID: string, environmentID: string, input: MutationBase, fetcher: typeof fetch = fetch): Promise<TopologyState> {
  return send(fetcher, "POST", `${environmentsPath(projectID)}/${encodeURIComponent(environmentID)}/restore`, input);
}
