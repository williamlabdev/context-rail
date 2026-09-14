import {
  errorWorkspaceState,
  loadingWorkspaceState,
  projectsWorkspaceState,
} from "../../src/state/workspaceState";

describe("workspace state contract", () => {
  it("represents loading explicitly", () => {
    expect(loadingWorkspaceState()).toEqual({ status: "LOADING" });
  });

  it("represents ready and empty registry states explicitly", () => {
    const project = {
      project: { id: "p-1", name: "Project 1", root: "fixture/p-1" },
      readiness: {},
      context: { status: "UNKNOWN" },
      read_only: true as const,
      observed_at: "2026-09-14T00:00:00Z",
    };
    expect(projectsWorkspaceState([project])).toEqual({
      status: "READY",
      projects: [project],
    });
    expect(projectsWorkspaceState([])).toEqual({ status: "EMPTY", projects: [] });
  });

  it("represents errors without mutation affordances", () => {
    expect(errorWorkspaceState("Registry unavailable")).toEqual({
      status: "ERROR",
      message: "Registry unavailable",
    });
  });
});
