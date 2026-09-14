import type { ProjectEntry } from "../api/projectRegistry";

export type WorkspaceState =
  | { status: "LOADING" }
  | { status: "READY"; projects: ProjectEntry[] }
  | { status: "EMPTY"; projects: [] }
  | { status: "ERROR"; message: string };

export function loadingWorkspaceState(): WorkspaceState {
  return { status: "LOADING" };
}

export function projectsWorkspaceState(projects: ProjectEntry[]): WorkspaceState {
  return projects.length === 0 ? { status: "EMPTY", projects: [] } : { status: "READY", projects };
}

export function errorWorkspaceState(message: string): WorkspaceState {
  return { status: "ERROR", message };
}
