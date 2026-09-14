import { test, expect } from "@playwright/test";

test.describe("Project Registry read-only workspace", () => {
  test("browses both Projects and replaces detail state on switching", async ({ page }) => {
    const methods: string[] = [];
    page.on("request", (request) => methods.push(request.method()));

    await page.goto("/");
    await expect(page.getByTestId("project-card-order-operations-portal")).toBeVisible();
    await expect(page.getByTestId("project-card-support-insights")).toBeVisible();
    await expect(page.getByTestId("project-context-page")).toContainText("Order Operations Portal");

    await page.getByTestId("project-card-support-insights").click();
    await expect(page.getByTestId("project-context-page")).toContainText("Support Insights");
    await expect(page.getByTestId("project-context-page")).toContainText("Context STALE");
    await expect(page.getByTestId("project-context-page")).toContainText("UNDECLARED");
    await expect(page.getByTestId("document-docs/ai/context-pack.json")).toContainText("STALE");

    expect(methods.every((method) => method === "GET")).toBe(true);
    await expect(page.getByRole("button", { name: /create|edit|archive|upload|approve|deploy/i })).toHaveCount(0);
  });

  test("shows an explicit loading state while the registry response is delayed", async ({ page }) => {
    await page.route("**/v1/projects", async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 100));
      await route.continue();
    });

    await page.goto("/");
    await expect(page.getByTestId("state-loading").first()).toBeVisible();
    await expect(page.getByTestId("project-card-order-operations-portal")).toBeVisible();
  });

  test("shows an explicit empty state without a create action", async ({ page }) => {
    await page.route("**/v1/projects", (route) => route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ kind: "ProjectRegistrySnapshot", schema_version: "project-registry/v1", projects: [] }),
    }));

    await page.goto("/");
    await expect(page.getByTestId("state-empty").first()).toContainText("No Projects configured");
    await expect(page.getByRole("button", { name: /create/i })).toHaveCount(0);
  });

  test("shows an explicit unavailable state without stale detail or mutation controls", async ({ page }) => {
    await page.route("**/v1/projects", (route) => route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({
        kind: "ProjectRegistryError",
        code: "PROJECT_UNAVAILABLE",
        message: "A configured Project is unavailable to the local registry.",
        read_only: true,
      }),
    }));

    await page.goto("/");
    await expect(page.getByTestId("state-error").first()).toContainText("PROJECT_UNAVAILABLE");
    await expect(page.getByRole("button", { name: /create|edit|archive|upload|approve|deploy/i })).toHaveCount(0);
  });
});
