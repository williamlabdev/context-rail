import { useEffect, useState } from "react";
import { fetchProject, fetchProjectRegistry, type ProjectEntry } from "./api/projectRegistry";
import { ProjectContextPage } from "./pages/ProjectContextPage";
import { ProjectRegistry } from "./components/ProjectRegistry";
import { WorkspaceState } from "./components/WorkspaceState";
import { errorWorkspaceState, loadingWorkspaceState, projectsWorkspaceState, type WorkspaceState as WorkspaceStateValue } from "./state/workspaceState";
import { useTopology } from "./state/useTopology";

export function App() {
  const [registryState, setRegistryState] = useState<WorkspaceStateValue>(loadingWorkspaceState());
  const [selectedProjectID, setSelectedProjectID] = useState<string | null>(null);
  const [detailState, setDetailState] = useState<WorkspaceStateValue>(loadingWorkspaceState());
  const topology = useTopology(selectedProjectID);

  useEffect(() => {
    let active = true;
    fetchProjectRegistry()
      .then((snapshot) => {
        if (!active) return;
        const nextState = projectsWorkspaceState(snapshot.projects);
        setRegistryState(nextState);
        const firstProject = snapshot.projects[0]?.project.id;
        if (firstProject) {
          setSelectedProjectID(firstProject);
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

  const selectProject = (projectID: string) => {
    setSelectedProjectID(projectID);
    setDetailState(loadingWorkspaceState());
    void loadProject(projectID, true, setDetailState);
  };

  const selectedEntry = detailState.status === "READY" ? detailState.projects[0] : undefined;
  return (
    <div className="app-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">CONTEXT RAIL</p>
          <p className="app-title">Project Workspace</p>
        </div>
        <div className="header-chips">
          <span className="read-only-chip">REGISTRY READ-ONLY</span>
          <span className="read-only-chip chip-versioned">TOPOLOGY VERSIONED</span>
        </div>
      </header>
      <div className="workspace-layout">
        {registryState.status === "READY" ? (
          <ProjectRegistry projects={registryState.projects} selectedProjectID={selectedProjectID} onSelect={selectProject} />
        ) : (
          <WorkspaceState state={registryState} title="Project Registry" />
        )}
        {selectedEntry ? <ProjectContextPage entry={selectedEntry} topology={topology} /> : <WorkspaceState state={detailState} title="Project context" />}
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
