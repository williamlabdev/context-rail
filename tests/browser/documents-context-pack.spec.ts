import { test, expect } from "@playwright/test";

// UI-20: open Documents / AI Context, verify a missing document and rebuild
// the Context Pack. The gap maps to decision / development / staging
// readiness, the pack shows PARTIAL with source path / version / hash and
// audit, and the AI never turns the gap into a fact.
test.describe("Documents / AI Context (UI-20)", () => {
  test("a declared-but-missing document blocks its stages, the rebuilt pack is PARTIAL and lists it, withdrawing restores DERIVED", async ({ page }) => {
    const runbook = "docs/operations/runbook.md";
    await page.goto("/");
    const panel = page.getByTestId("documents-panel");
    await expect(panel).toBeVisible();
    const meta = page.getByTestId("documents-meta");
    await expect(meta).toContainText("Baseline v");
    const versionOf = async () => Number(/Baseline v(\d+)/.exec(await meta.innerText())?.[1]);
    const start = await versionOf();
    await expect(page.getByTestId("document-readiness-decision")).toContainText("READY");
    await expect(page.getByTestId("documents-project.yaml")).toContainText("CURRENT");

    // Declare a runbook the repository does not have: verified, never created.
    await panel.getByRole("button", { name: "Declare required document" }).click();
    const declare = page.getByTestId("document-declare-form");
    await declare.locator("input[name=path]").fill(runbook);
    await declare.locator("input[name=kind]").fill("runbook");
    await declare.locator("input[name=stage-staging]").check();
    await declare.locator("input[name=reason]").fill("staging needs an operations runbook");
    await declare.getByRole("button", { name: "Declare as new baseline version" }).click();

    await expect(meta).toContainText(`Baseline v${start + 1}`);
    const row = page.getByTestId(`documents-${runbook}`);
    await expect(row).toContainText("MISSING");
    await expect(row).toContainText("DECLARED");
    await expect(row).toContainText("not synthesized");
    await expect(page.getByTestId("document-readiness-decision")).toContainText("NEEDS_INPUT");
    await expect(page.getByTestId("document-readiness-decision")).toContainText("runbook.md is MISSING");
    await expect(page.getByTestId("document-readiness-staging")).toContainText("NEEDS_INPUT");
    await expect(page.getByTestId("document-readiness-development")).toContainText("READY");

    // Rebuild: PARTIAL, the gap is listed with the stages it blocks, sources carry path / version / hash.
    const rebuild = page.getByTestId("context-pack-rebuild-form");
    await rebuild.locator("input[name=rebuild_reason]").fill("rebuild with the declared gap");
    await rebuild.getByRole("button", { name: "Rebuild Context Pack" }).click();
    const pack = page.getByTestId("context-pack");
    await expect(pack).toContainText("PARTIAL");
    await expect(page.getByTestId(`context-pack-missing-${runbook}`)).toContainText("blocks decision, staging");
    const packRow = page.getByTestId("context-pack-sources-project.yaml");
    await expect(packRow).toContainText("v1");
    await expect(packRow.locator("code").nth(2)).not.toHaveText("");
    await expect(page.getByTestId(`context-pack-sources-${runbook}`)).toContainText("not synthesized");
    await expect(page.getByTestId("context-pack-limitations")).toContainText("nothing was synthesized");
    await expect(page.getByTestId("context-pack-readiness-decision")).toContainText("NEEDS_INPUT");
    await expect(meta).toContainText("PARTIAL");
    const packID = (await pack.locator("code").first().innerText()).trim();
    expect(packID).toMatch(/^CP-\d{3}$/);

    // A new Change opened now needs input: the missing document is named, not invented.
    const changes = page.getByTestId("changes-panel");
    await changes.getByRole("button", { name: "New Change" }).click();
    const create = page.getByTestId("change-form-create");
    await create.locator("input[name=title]").fill(`Runbook gap ${Date.now().toString(36).slice(-4)}`);
    await create.locator("textarea[name=objective]").fill("Add attachments to order exceptions.");
    await create.locator("textarea[name=owner_summary]").fill("Operations staff can settle an order exception with a written note.");
    await create.locator("select[name=target_environment_id]").selectOption("staging");
    await create.locator("input[name=reason]").fill("ops request while a decision document is missing");
    await create.getByRole("button", { name: "Open Change and evaluate" }).click();
    await expect(page.getByTestId("change-detail")).toContainText("NEEDS_INPUT");
    await expect(page.getByTestId("change-missing-inputs")).toContainText("decision_documents");
    await expect(page.getByTestId("change-missing-inputs")).toContainText("runbook.md is MISSING");

    // Withdraw needs a reason; then the stages recover and a rebuild is DERIVED again.
    await row.getByRole("button", { name: "Withdraw" }).click();
    await expect(page.getByTestId("withdraw-reason-missing")).toBeVisible();
    await page.getByTestId("withdraw-reason").fill("runbook deferred to P1");
    await row.getByRole("button", { name: "Withdraw" }).click();
    await expect(meta).toContainText(`Baseline v${start + 2}`);
    await expect(page.getByTestId(`documents-${runbook}`)).toHaveCount(0);
    await expect(page.getByTestId("document-readiness-decision")).toContainText("READY");
    await expect(page.getByTestId("context-pack-drift")).toContainText("STALE");
    await rebuild.locator("input[name=rebuild_reason]").fill("runbook withdrawn");
    await rebuild.getByRole("button", { name: "Rebuild Context Pack" }).click();
    await expect(pack).toContainText("DERIVED");
    await expect(page.getByTestId("context-pack-drift")).toHaveCount(0);
    await expect(page.getByTestId("context-pack-missing")).toHaveCount(0);
    await expect(meta).toContainText("DERIVED");

    // Audit records every step.
    await panel.getByRole("button", { name: /Show audit/ }).click();
    const audit = page.getByTestId("documents-audit");
    await expect(audit).toContainText("baseline.declare");
    await expect(audit).toContainText("context_pack.rebuild");
    await expect(audit).toContainText("baseline.withdraw");
  });
});
