import { fireEvent, render, screen } from "@testing-library/react";
import { ProjectRegistry } from "../../src/components/ProjectRegistry";
import { ProjectContextPage } from "../../src/pages/ProjectContextPage";
import type { ProjectEntry } from "../../src/api/projectRegistry";

const order: ProjectEntry = {
  project: { id: "order-operations-portal", name: "Order Operations Portal", root: "demo/order-operations-portal", purpose: "Review order exceptions." },
  repositories: [{ url: "github.com/example/order-operations-portal" }],
  services: [{ id: "order-operations-web", repository: null, runtime: "go" }],
  environments: [{ id: "development", type: "development", sequence: 1 }],
  documents: [{ path: "project.yaml", kind: "project-manifest", source_of_truth: true, status: "CURRENT" }],
  decisions: [{ decision_id: "DR-001", status: "ACCEPTED_FOR_STAGING" }],
  readiness: { ready_for_decision: { status: "READY", reasons: [] } },
  context: { status: "DERIVED" },
  read_only: true,
  observed_at: "2026-09-14T00:00:00Z",
};

const support: ProjectEntry = {
  ...order,
  project: { id: "support-insights", name: "Support Insights", root: "examples/support-insights", purpose: "Read support trends." },
  services: [{ id: "support-insights-pipeline", repository: "support-insights", runtime: "UNDECLARED" }],
  context: { status: "STALE" },
  documents: [{ path: "docs/ai/context-pack.json", kind: "derived-context", source_of_truth: false, status: "STALE", stale_sources: ["README.md"] }],
};

describe("Project context journey", () => {
  it("allows selecting a Project without adding mutation controls", () => {
    const onSelect = vi.fn();
    render(<ProjectRegistry projects={[order, support]} selectedProjectID="order-operations-portal" onSelect={onSelect} />);

    expect(screen.getByText("Order Operations Portal")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("project-card-support-insights"));
    expect(onSelect).toHaveBeenCalledWith("support-insights");
    expect(screen.queryByText(/create|edit|archive|approve|deploy/i)).not.toBeInTheDocument();
  });

  it("renders the selected Project detail independently", () => {
    render(<ProjectContextPage entry={support} />);

    expect(screen.getByTestId("project-context-page")).toHaveTextContent("Support Insights");
    // UI-18: human label on screen, machine code still readable next to it.
    expect(screen.getByText("Context Stale")).toBeInTheDocument();
    expect(screen.getAllByText("STALE").length).toBeGreaterThan(0);
    expect(screen.getByText("support-insights-pipeline · UNDECLARED")).toBeInTheDocument();
    expect(screen.getByText("Stale source: README.md")).toBeInTheDocument();
    expect(screen.getByText("Informational; not authorization")).toBeInTheDocument();
  });
});
