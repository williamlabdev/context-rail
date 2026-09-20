import { useEffect, useState } from "react";
import { fetchProject, fetchProjectRegistry, type ProjectEntry } from "./api/projectRegistry";
import { ProjectContextPage } from "./pages/ProjectContextPage";
import { ProjectRegistry } from "./components/ProjectRegistry";
import { WorkspaceState } from "./components/WorkspaceState";
import { errorWorkspaceState, loadingWorkspaceState, projectsWorkspaceState, type WorkspaceState as WorkspaceStateValue } from "./state/workspaceState";
import { useTopology } from "./state/useTopology";
import { useChanges } from "./state/useChanges";
import { useReleases } from "./state/useReleases";
import { useDocuments } from "./state/useDocuments";
import { locales, useLocale } from "./i18n";

/** UI-15: the selected Project lives in the URL (/projects/{id}) so refresh, back and deep links restore it. */
function projectFromLocation(): string | null {
  if (typeof window === "undefined") return null;
  const match = /^\/projects\/([a-z0-9][a-z0-9._-]*)\/?$/.exec(window.location.pathname);
  return match ? match[1] : null;
}

function pushProjectLocation(projectID: string, replace = false) {
  if (typeof window === "undefined") return;
  const path = `/projects/${encodeURIComponent(projectID)}`;
  if (window.location.pathname === path) return;
  if (replace) window.history.replaceState({ projectID }, "", path);
  else window.history.pushState({ projectID }, "", path);
}

export function App() {
  const { t, locale, setLocale } = useLocale();
  const [registryState, setRegistryState] = useState<WorkspaceStateValue>(loadingWorkspaceState());
  const [selectedProjectID, setSelectedProjectID] = useState<string | null>(null);
  const [detailState, setDetailState] = useState<WorkspaceStateValue>(loadingWorkspaceState());
  const topology = useTopology(selectedProjectID);
  const documents = useDocuments(selectedProjectID);
  // Reload the ledger whenever the topology version or the document baseline
  // moves so STALE verdicts and NEEDS_INPUT document gaps appear.
  const changes = useChanges(selectedProjectID, `${topology.state?.current_version ?? ""}:${documents.view?.current_version ?? ""}:${documents.view?.latest_pack?.pack_id ?? ""}:${documents.view?.live_context_status ?? ""}`);
  // Releases depend on both the topology (drift) and the change ledger (accepted candidates).
  const releases = useReleases(selectedProjectID, `${topology.state?.current_version ?? ""}:${changes.changes.map((view) => `${view.change.change_id}@${view.change.updated_at}`).join(",")}`);

  useEffect(() => {
    let active = true;
    fetchProjectRegistry()
      .then((snapshot) => {
        if (!active) return;
        const nextState = projectsWorkspaceState(snapshot.projects);
        setRegistryState(nextState);
        // Deep link wins when it names a configured Project; otherwise the first Project.
        const requested = projectFromLocation();
        const known = snapshot.projects.some((entry) => entry.project.id === requested);
        const firstProject = (known && requested) || snapshot.projects[0]?.project.id;
        if (firstProject) {
          setSelectedProjectID(firstProject);
          pushProjectLocation(firstProject, true);
          void loadProject(firstProject, active, setDetailState);
        } else {
          setDetailState({ status: "EMPTY", projects: [] });
        }
      })
      .catch((error: unknown) => {
        if (!active) return;
        setRegistryState(errorWorkspaceState(error instanceof Error ? error.message : "Registry request failed."));
        setDetailState(errorWorkspaceState("The Project Registry could not be loaded."));
      });
    return () => { active = false; };
  }, []);

  const selectProject = (projectID: string, fromHistory = false) => {
    setSelectedProjectID(projectID);
    if (!fromHistory) pushProjectLocation(projectID);
    setDetailState(loadingWorkspaceState());
    void loadProject(projectID, true, setDetailState);
  };

  // Back / forward restore the Project the URL names (UI-15).
  useEffect(() => {
    const onPopState = () => {
      const projectID = projectFromLocation();
      if (projectID) selectProject(projectID, true);
    };
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selectedEntry = detailState.status === "READY" ? detailState.projects[0] : undefined;
  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">CONTEXT RAIL</p>
          <p className="app-title">{t("Project Workspace")}</p>
        </div>
        <div className="header-chips">
          <span className="read-only-chip">{t("REGISTRY READ-ONLY")}</span>
          <span className="read-only-chip chip-versioned">{t("TOPOLOGY VERSIONED")}</span>
          <span className="read-only-chip chip-ledger">{t("CHANGES GOVERNED")}</span>
          <span className="read-only-chip chip-release">{t("PROMOTION GATED")}</span>
          <span className="read-only-chip chip-versioned">{t("DOCUMENTS BASELINED")}</span>
          <span className="locale-switch" role="group" aria-label={t("Language")} data-testid="locale-switch">
            {locales.map((candidate) => (
              <button type="button" key={candidate} data-testid={`locale-${candidate}`} aria-pressed={locale === candidate} onClick={() => setLocale(candidate)}>{candidate === "en" ? "EN" : "繁中"}</button>
            ))}
          </span>
        </div>
      </header>
      <div className="workspace-layout">
        {registryState.status === "READY" ? (
          <ProjectRegistry projects={registryState.projects} selectedProjectID={selectedProjectID} onSelect={selectProject} />
        ) : (
          <WorkspaceState state={registryState} title={t("Project Registry")} />
        )}
        {selectedEntry ? <ProjectContextPage entry={selectedEntry} topology={topology} changes={changes} releases={releases} documents={documents} /> : <WorkspaceState state={detailState} title={t("Project context")} />}
      </div>
    </div>
  );
}

async function loadProject(
  projectID: string,
  active: boolean,
  setDetailState: (state: WorkspaceStateValue) => void,
) {
  try {
    const snapshot = await fetchProject(projectID);
    if (!active) return;
    const entry: ProjectEntry | undefined = snapshot.projects[0];
    setDetailState(entry ? { status: "READY", projects: [entry] } : errorWorkspaceState("The selected Project was not returned."));
  } catch (error: unknown) {
    if (!active) return;
    setDetailState(errorWorkspaceState(error instanceof Error ? error.message : "Project request failed."));
  }
}
