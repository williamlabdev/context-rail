export const projectStatuses = [
  "CURRENT",
  "DRAFT",
  "MISSING",
  "STALE",
  "CONFLICT",
  "REVOKED",
  "UNKNOWN",
  "UNDECLARED",
] as const;

export type ProjectStatus = (typeof projectStatuses)[number] | string;

export interface ProjectRecord {
  id: string;
  name: string;
  version?: number;
  classification?: string;
  status?: ProjectStatus;
  purpose?: string;
  owners?: Record<string, string>;
  root: string;
}

export interface RepositoryRecord {
  provider?: string;
  visibility?: string;
  url?: string;
  default_branch?: string;
  status?: ProjectStatus;
}

export interface ServiceRecord {
  id: string;
  repository: string | null;
  path?: string;
  runtime?: string;
  role?: string;
}

export interface EnvironmentRecord {
  id: string;
  type?: string;
  sequence?: number;
  target_ref?: string;
  required_evidence?: string[];
  action?: string | null;
}

export interface DocumentStatus {
  path: string;
  kind?: string;
  source_of_truth: boolean;
  status: ProjectStatus;
  stale_sources?: string[];
}

export interface DecisionSummary {
  decision_id?: string;
  request_id?: string;
  status?: string;
}

export interface ReadinessStatus {
  status: string;
  reasons: string[];
}

export interface ReadinessSet {
  ready_for_decision?: ReadinessStatus;
  ready_for_local_development?: ReadinessStatus;
  ready_for_cloud_testing?: ReadinessStatus;
  ready_for_staging?: ReadinessStatus;
  staging_verified?: ReadinessStatus;
  production?: ReadinessStatus;
}

export interface ProjectEntry {
  project: ProjectRecord;
  repositories?: RepositoryRecord[];
  services?: ServiceRecord[];
  environments?: EnvironmentRecord[];
  documents?: DocumentStatus[];
  decisions?: DecisionSummary[];
  readiness: ReadinessSet;
  context: { status: ProjectStatus };
  read_only: true;
  observed_at: string;
}

export interface ProjectRegistrySnapshot {
  kind: "ProjectRegistrySnapshot";
  schema_version: "project-registry/v1";
  projects: ProjectEntry[];
}

export type ProjectRegistryErrorCode =
  | "PROJECT_NOT_FOUND"
  | "PROJECT_INVALID"
  | "PROJECT_UNAVAILABLE"
  | "REGISTRY_UNAVAILABLE"
  | "METHOD_NOT_ALLOWED";

export interface ProjectRegistryError {
  kind: "ProjectRegistryError";
  code: ProjectRegistryErrorCode | string;
  message: string;
  read_only: true;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

export function parseProjectRegistrySnapshot(value: unknown): ProjectRegistrySnapshot {
  if (!isRecord(value) || value.kind !== "ProjectRegistrySnapshot") {
    throw new Error("Expected ProjectRegistrySnapshot");
  }
  if (value.schema_version !== "project-registry/v1") {
    throw new Error("Expected project-registry/v1");
  }
  if (!Array.isArray(value.projects)) {
    throw new Error("ProjectRegistrySnapshot projects must be an array");
  }
  return value as unknown as ProjectRegistrySnapshot;
}

export function parseProjectRegistryError(value: unknown): ProjectRegistryError {
  if (!isRecord(value) || value.kind !== "ProjectRegistryError") {
    throw new Error("Expected ProjectRegistryError");
  }
  if (typeof value.code !== "string" || typeof value.message !== "string" || value.read_only !== true) {
    throw new Error("Invalid ProjectRegistryError");
  }
  return value as unknown as ProjectRegistryError;
}

async function readResponse(response: Response): Promise<unknown> {
  const value: unknown = await response.json();
  if (!response.ok) {
    const error = parseProjectRegistryError(value);
    throw new Error(`${error.code}: ${error.message}`);
  }
  return parseProjectRegistrySnapshot(value);
}

export async function fetchProjectRegistry(
  fetcher: typeof fetch = fetch,
): Promise<ProjectRegistrySnapshot> {
  return readResponse(await fetcher("/v1/projects")) as Promise<ProjectRegistrySnapshot>;
}

export async function fetchProject(
  projectID: string,
  fetcher: typeof fetch = fetch,
): Promise<ProjectRegistrySnapshot> {
  return readResponse(await fetcher(`/v1/projects/${encodeURIComponent(projectID)}`)) as Promise<ProjectRegistrySnapshot>;
}
