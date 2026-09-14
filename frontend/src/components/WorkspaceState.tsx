import type { WorkspaceState } from "../state/workspaceState";

interface WorkspaceStateProps {
  state: WorkspaceState;
  title?: string;
}

export function WorkspaceState({ state, title = "Project workspace" }: WorkspaceStateProps) {
  if (state.status === "LOADING") {
    return <div className="workspace-state" data-testid="state-loading" role="status"><span className="spinner" /> Loading {title}…</div>;
  }
  if (state.status === "EMPTY") {
    return (
      <div className="workspace-state" data-testid="state-empty">
        <p className="eyebrow">EMPTY</p>
        <h2>No Projects configured</h2>
        <p>Provide an explicit local Project root to inspect. This read-only slice does not create Projects.</p>
      </div>
    );
  }
  if (state.status === "ERROR") {
    return (
      <div className="workspace-state error-state" data-testid="state-error" role="alert">
        <p className="eyebrow">UNAVAILABLE</p>
        <h2>{title} unavailable</h2>
        <p>{state.message}</p>
      </div>
    );
  }
  return null;
}
