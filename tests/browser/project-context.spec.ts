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
});
