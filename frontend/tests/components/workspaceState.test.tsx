import { render, screen } from "@testing-library/react";
import { WorkspaceState } from "../../src/components/WorkspaceState";

describe("workspace state presentation", () => {
  it("renders loading, empty and error states explicitly", () => {
    const { rerender } = render(<WorkspaceState state={{ status: "LOADING" }} title="Project Registry" />);
    expect(screen.getByTestId("state-loading")).toHaveTextContent("Loading Project Registry");

    rerender(<WorkspaceState state={{ status: "EMPTY", projects: [] }} title="Project Registry" />);
    expect(screen.getByTestId("state-empty")).toHaveTextContent("No Projects configured");
    expect(screen.queryByRole("button", { name: /create/i })).not.toBeInTheDocument();

    rerender(<WorkspaceState state={{ status: "ERROR", message: "Registry unavailable" }} title="Project Registry" />);
    expect(screen.getByRole("alert")).toHaveTextContent("Registry unavailable");
  });
});
