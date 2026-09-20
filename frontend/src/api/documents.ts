// Client for the governed document baseline and the Context Pack rebuild (UI-20).
//
// Declaring a required document never creates it: a missing file is listed
// with the readiness stages it blocks, and the rebuilt Context Pack is
// PARTIAL until the consumer adds it. Nothing is synthesized.

export interface DeclaredDocument {
  path: string; kind: string; source_of_truth: boolean; required_for: string[]; origin: "manifest" | "operator" | string; declared_in_version: number;
}
export interface BaselineChange { operation: string; path?: string; reason: string; impact: string }
export interface BaselineVersion {
  version: number; previous_version: number; created_at: string; actor: string; origin: string; change: BaselineChange; documents: DeclaredDocument[]; baseline_hash: string;
}
export interface DocumentSource {
  path: string; kind: string; source_of_truth: boolean; required_for: string[]; origin: string; version: string; status: string;
  content_hash?: string; bytes?: number; observed_at: string; stale_sources?: string[]; access_scope: string;
}
export interface MissingSource { path: string; kind: string; required_for: string[]; readiness_impact: string[]; origin: string }
export interface DocumentReadiness { status: string; reasons: string[] }
export interface ContextPack {
  kind: string; schema_version: string; derived: boolean; status: string; project_id: string; pack_id: string; generated_at: string; generated_by: string;
  reason: string; baseline_version: number; baseline_hash: string; topology_version?: number; topology_config_hash?: string; source_snapshot_hash: string;
  sources: DocumentSource[]; missing: MissingSource[]; readiness: Record<string, DocumentReadiness>; decision_refs: string[]; limitations: string[];
  policy_version: string; pack_hash: string;
}
export interface AuditEvent { sequence: number; at: string; actor: string; action: string; object: string; reason: string }
export interface DocumentsView {
  project_id: string; current_version: number; baseline: BaselineVersion; documents: DocumentSource[]; missing: MissingSource[];
  readiness: Record<string, DocumentReadiness>; declared_context: { path: string; status: string }; live_context_status: string;
  latest_pack: ContextPack | null; pack_drift: { stale: boolean; reason?: string }; packs: string[]; versions: BaselineVersion[]; audit: AuditEvent[];
  observed_at: string; observed_root: string; read_only_consumer: boolean;
}

export const documentStages = ["decision", "development", "work-order", "staging"] as const;

export class DocumentRequestError extends Error {
  constructor(public readonly code: string, message: string, public readonly status: number) {
    super(`${code}: ${message}`);
    this.name = "DocumentRequestError";
  }
}

export interface DeclareInput { expected_version: number; actor?: string; reason: string; path: string; kind: string; required_for: string[] }
export interface WithdrawInput { expected_version: number; actor?: string; reason: string; path: string }
export interface RebuildInput { actor?: string; reason: string }

async function readJSON<T>(response: Response): Promise<T> {
  const value = (await response.json()) as Record<string, unknown>;
  if (!response.ok) {
    throw new DocumentRequestError(String(value.code ?? "DOCUMENT_ERROR"), String(value.message ?? "Document request failed."), response.status);
  }
  return value as unknown as T;
}

function base(projectID: string): string {
  return `/v1/projects/${encodeURIComponent(projectID)}`;
}

async function post<T>(fetcher: typeof fetch, path: string, body: unknown): Promise<T> {
  return readJSON<T>(await fetcher(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }));
}

export async function fetchDocuments(projectID: string, fetcher: typeof fetch = fetch): Promise<DocumentsView> {
  return readJSON<DocumentsView>(await fetcher(`${base(projectID)}/documents`));
}
export async function declareDocument(projectID: string, input: DeclareInput, fetcher: typeof fetch = fetch): Promise<DocumentsView> {
  return post<DocumentsView>(fetcher, `${base(projectID)}/documents`, input);
}
export async function withdrawDocument(projectID: string, input: WithdrawInput, fetcher: typeof fetch = fetch): Promise<DocumentsView> {
  return post<DocumentsView>(fetcher, `${base(projectID)}/documents/withdraw`, input);
}
export async function rebuildContextPack(projectID: string, input: RebuildInput, fetcher: typeof fetch = fetch): Promise<DocumentsView> {
  return post<DocumentsView>(fetcher, `${base(projectID)}/context-pack/rebuild`, input);
}
