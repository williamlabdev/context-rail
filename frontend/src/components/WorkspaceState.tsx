import type { WorkspaceState } from "../state/workspaceState";
import { useLocale } from "../i18n";

interface WorkspaceStateProps {
  state: WorkspaceState;
  title?: string;
}

export function WorkspaceState({ state, title }: WorkspaceStateProps) {
  const { t } = useLocale();
  const name = title ?? t("Project workspace");
  if (state.status === "LOADING") {
    return <div className="workspace-state" data-testid="state-loading" role="status"><span className="spinner" /> {t("Loading {title}…", { title: name })}</div>;
  }
  if (state.status === "EMPTY") {
    return (
      <div className="workspace-state" data-testid="state-empty">
        <p className="eyebrow">{t("EMPTY")}</p>
        <h2>{t("No Projects configured")}</h2>
        <p>{t("Provide an explicit local Project root to inspect. This read-only slice does not create Projects.")}</p>
      </div>
    );
  }
  if (state.status === "ERROR") {
    return (
      <div className="workspace-state error-state" data-testid="state-error" role="alert">
        <p className="eyebrow">{t("UNAVAILABLE")}</p>
        <h2>{t("{title} unavailable", { title: name })}</h2>
        <p>{state.message}</p>
      </div>
    );
  }
  return null;
}
