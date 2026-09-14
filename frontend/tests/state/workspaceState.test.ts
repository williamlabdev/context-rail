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
    expect(projectsWorkspaceState([{ project: { id: "p-1", name: "Project 1" } }])).toEqual({
      status: "READY",
      projects: [{ project: { id: "p-1", name: "Project 1" } }],
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
