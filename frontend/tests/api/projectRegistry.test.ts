import {
  parseProjectRegistryError,
  parseProjectRegistrySnapshot,
} from "../../src/api/projectRegistry";

describe("Project Registry API contract", () => {
  it("accepts a read-only snapshot", () => {
    const snapshot = parseProjectRegistrySnapshot({
      kind: "ProjectRegistrySnapshot",
      schema_version: "project-registry/v1",
      projects: [
        {
          project: { id: "support-insights", name: "Support Insights" },
          readiness: { ready_for_decision: { status: "READY", reasons: [] } },
          context: { status: "STALE" },
          read_only: true,
          observed_at: "2026-09-14T00:00:00Z",
        },
      ],
    });

    expect(snapshot.projects[0]?.project.id).toBe("support-insights");
    expect(snapshot.projects[0]?.context.status).toBe("STALE");
    expect(snapshot.projects[0]?.read_only).toBe(true);
  });

  it("rejects a snapshot that is not the read-only registry contract", () => {
    expect(() =>
      parseProjectRegistrySnapshot({
        kind: "OtherSnapshot",
        schema_version: "project-registry/v1",
        projects: [],
      }),
    ).toThrow("ProjectRegistrySnapshot");
  });

  it("parses safe API errors", () => {
    const error = parseProjectRegistryError({
      kind: "ProjectRegistryError",
      code: "PROJECT_NOT_FOUND",
      message: "The requested Project is not configured for this local registry.",
      read_only: true,
    });

    expect(error.code).toBe("PROJECT_NOT_FOUND");
    expect(error.read_only).toBe(true);
  });
});
