import { render, screen } from "@testing-library/react";
import { DocumentStatusList } from "../../src/components/DocumentStatusList";
import { ReadinessSummary } from "../../src/components/ReadinessSummary";

describe("uncertain status presentation", () => {
  it("keeps every uncertain document state visible and non-success", () => {
    render(
      <DocumentStatusList
        documents={[
          { path: "stale.md", source_of_truth: true, status: "STALE" },
          { path: "missing.md", source_of_truth: true, status: "MISSING" },
          { path: "conflict.md", source_of_truth: true, status: "CONFLICT" },
          { path: "unknown.md", source_of_truth: true, status: "UNKNOWN" },
          { path: "undeclared.md", source_of_truth: true, status: "UNDECLARED" },
        ]}
      />,
    );

    for (const status of ["STALE", "MISSING", "CONFLICT", "UNKNOWN", "UNDECLARED"]) {
      expect(screen.getByText(status)).toBeInTheDocument();
    }
    expect(screen.queryByText("CURRENT")).not.toBeInTheDocument();
    expect(screen.queryByText("PASS")).not.toBeInTheDocument();
  });

  it("shows readiness reasons and states that it is informational", () => {
    render(
      <ReadinessSummary
        readiness={{
          ready_for_local_development: { status: "NEEDS_INPUT", reasons: ["no human-accepted DecisionRecord found"] },
          production: { status: "BLOCKED", reasons: ["production is human-gated"] },
        }}
      />,
    );

    expect(screen.getByText("NEEDS_INPUT")).toBeInTheDocument();
    expect(screen.getByText("BLOCKED")).toBeInTheDocument();
    expect(screen.getByText("no human-accepted DecisionRecord found")).toBeInTheDocument();
    expect(screen.getByText("Informational; not authorization")).toBeInTheDocument();
  });
});
