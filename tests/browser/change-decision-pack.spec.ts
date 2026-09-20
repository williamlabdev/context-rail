import { test, expect } from "@playwright/test";

// UI-06 → UI-09: open a Change, get blocked as NEEDS_INPUT, supply inputs,
// accept a decision, see Brief and Agent Context Pack share one lineage,
// compile a hashed Work Order, then watch everything go STALE when the
// topology changes materially.
test.describe("Change decision pack (VS-004)", () => {
  test("request → NEEDS_INPUT → decision → brief/pack → work order → STALE on topology change", async ({ page }) => {
    await page.goto("/");
    const panel = page.getByTestId("changes-panel");
    await expect(panel).toBeVisible();

    // UI-07: under-specified change is blocked, with owners, and cannot be decided.
    await panel.getByRole("button", { name: "New Change" }).click();
    const create = page.getByTestId("change-form-create");
    const title = `Manual order review ${Date.now().toString(36).slice(-4)}`;
    await create.locator("input[name=title]").fill(title);
    await create.locator("textarea[name=objective]").fill("Let operations approve or reject an order exception with a note.");
    await create.locator("select[name=target_environment_id]").selectOption("staging");
    await create.locator("input[name=reason]").fill("ops request");
    await create.getByRole("button", { name: "Open Change and evaluate" }).click();

    const detail = page.getByTestId("change-detail");
    await expect(detail).toContainText("NEEDS_INPUT");
    const missing = page.getByTestId("change-missing-inputs");
    await expect(missing).toContainText("acceptance_criteria");
    await expect(missing).toContainText("allowed_paths");
    await expect(missing).toContainText("business_constraints.expected_monthly_volume");
    await expect(missing).toContainText("business_owner");
    await expect(page.getByTestId("change-decision-form")).toHaveCount(0);
    await expect(page.getByTestId("change-unknowns")).toContainText("UNKNOWN");

    // UI-08: supply inputs → new version, DECISION_READY, old version in history.
    const inputs = page.getByTestId("change-inputs-form");
    await inputs.locator("textarea[name=acceptance_criteria]").fill("approve and reject require a note\nunknown order returns 404");
    await inputs.locator("textarea[name=allowed_paths]").fill("main.go\nweb/index.html\nmain_test.go");
    await inputs.locator("input[name=data_classification]").fill("internal");
    await inputs.locator("input[name=expected_monthly_volume]").fill("500 reviews");
    await inputs.locator("input[name=reason]").fill("requester and business owner supplied inputs");
    await inputs.getByRole("button", { name: "Re-evaluate with these inputs" }).click();
    await expect(detail).toContainText("DECISION_READY");
    await expect(detail).toContainText("change v2");
    await expect(page.getByTestId("change-history")).toContainText("v1");

    // UI-06/09: human accepts a candidate.
    const decide = page.getByTestId("change-decision-form");
    await expect(page.getByTestId("change-options")).toContainText("Minimal reversible slice");
    await decide.locator("input[name=actor]").fill("founder-001");
    await decide.locator("textarea[name=rationale]").fill("smallest reversible slice proves the path");
    await decide.getByRole("button", { name: "Accept decision" }).click();
    await expect(page.getByTestId("change-decision")).toContainText("DEC-");
    await expect(page.getByTestId("change-decision")).toContainText("ACCEPTED_FOR_DEVELOPMENT");

    // Brief and pack share the same lineage strip.
    const brief = page.getByTestId("change-brief");
    const pack = page.getByTestId("change-pack");
    await expect(brief).toContainText("CHANGE DECISION BRIEF");
    await expect(pack).toContainText("AGENT CONTEXT PACK");
    const briefLineage = await brief.getByTestId("lineage-strip").innerText();
    const packLineage = await pack.getByTestId("lineage-strip").innerText();
    expect(briefLineage).toBe(packLineage);
    await expect(pack).toContainText("deploy:production");
    await expect(pack).toContainText("AC-001");

    // Work order compiles with a hash and the same lineage.
    const woForm = page.getByTestId("change-work-order-form");
    await woForm.locator("input[name=issuer]").fill("founder-001");
    await woForm.locator("input[name=reason]").fill("hand off to agent");
    await woForm.getByRole("button", { name: "Compile Work Order" }).click();
    const order = page.getByTestId("change-work-order");
    await expect(order).toContainText("AWO-");
    await expect(order).toContainText("ISSUED");
    await expect(order).toContainText("sha256:");
    expect(await order.getByTestId("lineage-strip").innerText()).toBe(packLineage);
    await expect(page.getByTestId("change-work-order-form")).toHaveCount(0);

    // Material topology change → decision, pack and work order STALE.
    const topology = page.getByTestId("environment-topology");
    await topology.getByTestId("environment-row-staging").getByRole("button", { name: "Edit" }).click();
    const edit = page.getByTestId("environment-form-edit");
    await edit.locator("input[name=target_ref]").fill(`cloud-run/oop-staging-${Date.now().toString(36).slice(-4)}`);
    await edit.locator("input[name=reason]").fill("staging service moved");
    await edit.getByRole("button", { name: "Save as new version" }).click();
    await expect(page.getByTestId("change-stale")).toContainText("STALE");
    await expect(page.getByTestId("change-work-order")).toContainText("STALE");
    await expect(page.getByTestId("change-pack")).toContainText("STALE");
  });
});
