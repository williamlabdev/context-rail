import { test, expect } from "@playwright/test";

// UI-05: add, edit, reorder, retire and restore an environment. Every action
// must produce a new topology version, material changes must invalidate the
// observed decisions, and production must stay protected.
test.describe("Environment topology (VS-003)", () => {
  test("add → edit → reorder → retire → restore, each as a new version", async ({ page }) => {
    const id = `uat-${Date.now().toString(36).slice(-5)}`;
    await page.goto("/");
    const panel = page.getByTestId("environment-topology");
    await expect(panel).toBeVisible();
    await expect(panel).toContainText("Topology v");
    await expect(page.getByTestId("environment-row-production")).toContainText("BLOCKED IN P0");

    const versionOf = async () => {
      const text = await panel.locator(".topology-heading-meta").innerText();
      return Number(/Topology v(\d+)/.exec(text)?.[1]);
    };
    // Retrying assertion: the row status and the heading version render from
    // the same state, but a snapshot read can race the re-render.
    const expectVersion = (version: number) => expect(panel.locator(".topology-heading-meta")).toContainText(`Topology v${version}`);
    const start = await versionOf();

    // add
    await page.getByRole("button", { name: "Add environment" }).click();
    const add = page.getByTestId("environment-form-add");
    await add.locator("input[name=id]").fill(id);
    await add.locator("input[name=display_name]").fill("UAT");
    await add.locator("select[name=type]").selectOption("uat");
    await add.locator("input[name=sequence]").fill("3");
    await add.locator("input[name=target_ref]").fill(`cloud-run/oop-${id}`);
    await add.locator("input[name=owner]").fill("qa-lead");
    await add.locator("input[name=required_evidence]").fill("test, uat-signoff");
    await add.locator("input[name=reason]").fill("UAT gate before staging");
    await add.getByRole("button", { name: "Create version with new environment" }).click();
    await expect(page.getByTestId(`environment-row-${id}`)).toContainText("ACTIVE");
    await expectVersion(start + 1);
    await expect(page.getByTestId("topology-last-change")).toContainText(`ADD · ${id}`);
    await expect(page.getByTestId("topology-invalidations")).toContainText("DR-001");
    await expect(page.getByTestId("project-context-page")).toContainText("→ STALE");

    // edit (display-only)
    const row = page.getByTestId(`environment-row-${id}`);
    await row.getByRole("button", { name: "Edit" }).click();
    const edit = page.getByTestId("environment-form-edit");
    await edit.locator("input[name=display_name]").fill("User Acceptance");
    await edit.locator("input[name=reason]").fill("clearer label");
    await edit.getByRole("button", { name: "Save as new version" }).click();
    await expect(row).toContainText("User Acceptance");
    await expectVersion(start + 2);
    await expect(page.getByTestId("topology-last-change")).toContainText("INFORMATIONAL");

    // reorder requires a reason
    await row.getByRole("button", { name: `Move ${id} down` }).click();
    await expect(page.getByTestId("topology-reason-missing")).toBeVisible();
    await page.getByTestId("topology-quick-reason").fill("UAT after staging");
    await row.getByRole("button", { name: `Move ${id} down` }).click();
    await expectVersion(start + 3);
    await expect(page.getByTestId("topology-last-change")).toContainText("REORDER");

    // retire, then restore
    await page.getByTestId("topology-quick-reason").fill("UAT merged into staging");
    await row.getByRole("button", { name: "Retire" }).click();
    await expect(row).toContainText("RETIRED");
    await expectVersion(start + 4);
    await page.getByTestId("topology-quick-reason").fill("UAT needed again");
    await row.getByRole("button", { name: "Restore" }).click();
    await expect(row).toContainText("ACTIVE");
    await expectVersion(start + 5);

    // production stays protected
    const production = page.getByTestId("environment-row-production");
    await expect(production.getByRole("button", { name: "Retire" })).toBeDisabled();

    // history lists every version
    await page.getByRole("button", { name: /Show version history/ }).click();
    await expect(page.getByTestId("topology-history")).toContainText("BOOTSTRAP");
    await expect(page.getByTestId("topology-history")).toContainText("RESTORE");
  });

  test("a stale expected_version is refused and the operator is told to reload", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("environment-topology")).toContainText("Topology v");
    // Simulate another operator moving the topology forward: the UI's cached
    // version becomes stale and the server must answer 409.
    await page.route("**/v1/projects/*/environments/reorder", (route) => route.fulfill({
      status: 409,
      contentType: "application/json",
      body: JSON.stringify({ kind: "EnvironmentTopologyError", code: "TOPOLOGY_VERSION_CONFLICT", message: "expected topology v1 but current is v7; reload and retry" }),
    }));
    await page.getByTestId("topology-quick-reason").fill("try to move");
    await page.getByRole("button", { name: "Move testing up" }).click();
    await expect(page.getByTestId("topology-error")).toContainText("TOPOLOGY_VERSION_CONFLICT");
    await expect(page.getByRole("button", { name: "Reload topology" })).toBeVisible();
  });
});
