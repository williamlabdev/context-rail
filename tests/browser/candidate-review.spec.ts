import { test, expect } from "@playwright/test";

// UI-10: a candidate that touches a path outside allowed_paths is BLOCKED with
// the violating path shown. UI-11: a candidate without an independent review
// is NEEDS_REVIEW and cannot be accepted. Then a compliant candidate is
// accepted by a human other than the run starter.
test.describe("Candidate review (VS-005a)", () => {
  test("out-of-scope path blocks, missing review blocks, compliant candidate is accepted", async ({ page }) => {
    await page.goto("/");
    const panel = page.getByTestId("changes-panel");
    await expect(panel).toBeVisible();

    // Prepare a decided change with a work order.
    await panel.getByRole("button", { name: "New Change" }).click();
    const create = page.getByTestId("change-form-create");
    await create.locator("input[name=title]").fill(`Candidate gate ${Date.now().toString(36).slice(-4)}`);
    await create.locator("textarea[name=objective]").fill("Add manual review with a note.");
    await create.locator("select[name=target_environment_id]").selectOption("staging");
    await create.locator("textarea[name=acceptance_criteria]").fill("approve and reject require a note");
    await create.locator("textarea[name=allowed_paths]").fill("main.go\nweb/index.html\nmain_test.go");
    await create.locator("input[name=data_classification]").fill("internal");
    await create.locator("input[name=expected_monthly_volume]").fill("500");
    await create.locator("input[name=reason]").fill("ops request");
    await create.getByRole("button", { name: "Open Change and evaluate" }).click();
    const decide = page.getByTestId("change-decision-form");
    await decide.locator("input[name=actor]").fill("founder-001");
    await decide.locator("textarea[name=rationale]").fill("ok");
    await decide.getByRole("button", { name: "Accept decision" }).click();
    const woForm = page.getByTestId("change-work-order-form");
    await woForm.locator("input[name=issuer]").fill("founder-001");
    await woForm.locator("input[name=reason]").fill("hand off");
    await woForm.getByRole("button", { name: "Compile Work Order" }).click();
    await expect(page.getByTestId("candidate-review")).toBeVisible();

    // UI-10: out-of-scope path → BLOCKED, path visible, accept disabled.
    let form = page.getByTestId("candidate-form");
    await form.locator("input[name=started_by]").fill("dev-1");
    await form.locator("input[name=head_commit]").fill("abc123");
    await form.locator("textarea[name=changed_paths]").fill("main.go\ninfra/iam/public-bucket.yaml");
    await form.locator("input[name=reviewer]").fill("reviewer-2");
    await form.locator("input[name=reason]").fill("agent finished");
    await form.getByRole("button", { name: "Submit candidate to gate" }).click();
    const first = page.getByTestId("candidate-CAND-001");
    await expect(first).toContainText("BLOCKED");
    await expect(page.getByTestId("candidate-violations-CAND-001")).toContainText("infra/iam/public-bucket.yaml");
    await expect(page.getByTestId("gate-CAND-001-allowed_paths")).toContainText("BLOCKED");
    await expect(page.getByTestId("gate-CAND-001-required_checks")).toContainText("PASS");
    await expect(first.getByRole("button", { name: "Accept candidate for promotion" })).toBeDisabled();
    const firstDecision = page.getByTestId("candidate-decision-CAND-001");
    await firstDecision.locator("input[name=actor]").fill("founder-001");
    await firstDecision.locator("input[name=rationale]").fill("out of scope");
    await firstDecision.getByRole("button", { name: "Reject candidate" }).click();
    await expect(page.getByTestId("candidate-human-CAND-001")).toContainText("REJECTED");

    // UI-11: no independent review → NEEDS_REVIEW.
    await page.getByRole("button", { name: "Submit another candidate" }).click();
    form = page.getByTestId("candidate-form");
    await form.locator("input[name=started_by]").fill("dev-1");
    await form.locator("input[name=head_commit]").fill("def456");
    await form.locator("textarea[name=changed_paths]").fill("main.go\nmain_test.go");
    await form.locator("input[name=reviewer]").fill("dev-1"); // self review
    await form.locator("input[name=reason]").fill("fixed scope, self-reviewed");
    await form.getByRole("button", { name: "Submit candidate to gate" }).click();
    const second = page.getByTestId("candidate-CAND-002");
    await expect(second).toContainText("NEEDS_REVIEW");
    await expect(page.getByTestId("gate-CAND-002-independent_review")).toContainText("NEEDS_REVIEW");
    await expect(second.getByRole("button", { name: "Accept candidate for promotion" })).toBeDisabled();

    // Compliant candidate with an independent reviewer → acceptable → accepted.
    await page.getByRole("button", { name: "Submit another candidate" }).click();
    form = page.getByTestId("candidate-form");
    await form.locator("input[name=started_by]").fill("dev-1");
    await form.locator("input[name=head_commit]").fill("fed789");
    await form.locator("textarea[name=changed_paths]").fill("main.go\nmain_test.go");
    await form.locator("input[name=reviewer]").fill("reviewer-2");
    await form.locator("input[name=reason]").fill("independent review obtained");
    await form.getByRole("button", { name: "Submit candidate to gate" }).click();
    const third = page.getByTestId("candidate-CAND-003");
    await expect(third).toContainText("CANDIDATE_ACCEPTABLE");
    await expect(page.getByTestId("gate-CAND-003-promotion_checks")).toContainText("DEFERRED_TO_PROMOTION");
    const thirdDecision = page.getByTestId("candidate-decision-CAND-003");
    await thirdDecision.locator("input[name=actor]").fill("founder-001");
    await thirdDecision.locator("input[name=rationale]").fill("gates green");
    await thirdDecision.getByRole("button", { name: "Accept candidate for promotion" }).click();
    await expect(page.getByTestId("candidate-human-CAND-003")).toContainText("ACCEPTED");
    await expect(page.getByTestId("change-detail")).toContainText("CANDIDATE_ACCEPTED");
  });
});
