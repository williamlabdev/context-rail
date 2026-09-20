import { test, expect, type APIRequestContext } from "@playwright/test";

const PROJECT = "order-operations-portal";
const DIGEST = "sha256:5555555555555555555555555555555555555555555555555555555555555555";

// Seed a change through the API so the browser journey focuses on promotion.
async function seedChange(api: APIRequestContext, base: string, title: string, commit: string, acceptCandidate: boolean): Promise<string> {
  const changes = `${base}/v1/projects/${PROJECT}/changes`;
  let response = await api.post(changes, { data: { reason: "seed", request: {
    title, objective: "seeded change", acceptance_criteria: [{ text: "works" }], allowed_paths: ["main.go", "web/index.html", "main_test.go"],
    target_environment_id: "staging", business_constraints: { data_classification: "internal", expected_monthly_volume: "100" },
  } } });
  expect(response.ok()).toBeTruthy();
  const changeID = (await response.json()).change.change_id as string;
  response = await api.post(`${changes}/${changeID}/decision`, { data: { decision: "ACCEPT", actor: "founder-001", selected_option: "minimal_reversible_slice", rationale: "seed" } });
  expect(response.ok()).toBeTruthy();
  response = await api.post(`${changes}/${changeID}/work-order`, { data: { reason: "seed", actor: "founder-001", issuer: "founder-001" } });
  expect(response.ok()).toBeTruthy();
  const order = (await response.json()).work_order;
  if (!acceptCandidate) return changeID;
  response = await api.post(`${changes}/${changeID}/candidates`, { data: { reason: "seed", actor: "dev-1",
    run: { run_id: `ARR-${changeID}`, work_order_id: order.work_order_id, work_order_hash: order.work_order_hash, started_by: "dev-1", started_at: "2026-09-21T02:00:00Z", agent: { name: "claude-code", provider: "", model: "", version: "" } },
    observation: { branch: order.target_branch, base_branch: order.base_branch, head_commit: commit, changed_paths: ["main.go"], checks: [{ name: "test", status: "PASS" }, { name: "build", status: "PASS" }], reviews: [{ reviewer: "reviewer-2", kind: "human", verdict: "APPROVED" }] } } });
  expect(response.ok()).toBeTruthy();
  response = await api.post(`${changes}/${changeID}/candidates/CAND-001/decision`, { data: { decision: "ACCEPT", actor: "founder-001", rationale: "seed" } });
  expect(response.ok()).toBeTruthy();
  return changeID;
}

test.describe("Staging promotion (VS-006)", () => {
  test("UI-13 bundle blocks as a whole, UI-12 normal promotion yields a receipt, UI-14 drift stales the approval", async ({ page, request, baseURL }) => {
    const base = baseURL ?? "http://127.0.0.1:8080";
    const good = await seedChange(request, base, `Promotable ${Date.now().toString(36).slice(-4)}`, "aaa111", true);
    const half = await seedChange(request, base, `Half done ${Date.now().toString(36).slice(-4)}`, "", false);

    await page.goto("/");
    const panel = page.getByTestId("releases-panel");
    await expect(panel).toBeVisible();

    // UI-13: bundle with one change lacking an accepted candidate.
    await panel.getByRole("button", { name: "New release" }).click();
    let form = page.getByTestId("release-form-create");
    await form.locator(`input[name=change-${good}]`).check();
    await form.locator(`input[name=change-${half}]`).check();
    await form.locator("select[name=target_environment_id]").selectOption("staging");
    await form.locator("input[name=reason]").fill("bundle both");
    await form.getByRole("button", { name: "Open release and run promotion gate" }).click();
    const detail = page.getByTestId("release-detail");
    await expect(detail).toContainText("GATE_BLOCKED");
    await expect(page.getByTestId(`release-gates-change-${good}`)).toContainText("PASS");
    await expect(page.getByTestId(`release-gates-change-${half}`)).toContainText("BLOCKED");
    await expect(page.getByTestId(`release-gates-change-${half}`)).toContainText("no candidate accepted");
    await expect(page.getByTestId("release-approval-form")).toHaveCount(0);
    await expect(page.getByTestId("release-blocked-note")).toContainText("No partial promotion");
    const blockedID = (await detail.locator("code").first().innerText()).trim();

    // UI-12: normal promotion of the good change alone.
    await panel.getByRole("button", { name: "New release" }).click();
    form = page.getByTestId("release-form-create");
    await form.locator(`input[name=change-${good}]`).check();
    await form.locator("select[name=target_environment_id]").selectOption("staging");
    await form.locator("input[name=reason]").fill("promote good change");
    await form.getByRole("button", { name: "Open release and run promotion gate" }).click();
    await expect(detail).toContainText("AWAITING_BUILD");
    const releaseID = (await detail.locator("code").first().innerText()).trim();
    expect(releaseID).not.toBe(blockedID);

    const build = page.getByTestId("release-build-form");
    await build.locator("input[name=image_digest]").fill(DIGEST);
    await build.locator("input[name=source_commit]").fill("aaa111");
    await build.locator("input[name=build_id]").fill("cb-42");
    await build.locator("input[name=evidence_ref]").fill("https://console.cloud.google.com/cloud-build/builds/cb-42");
    await build.locator("input[name=reason]").fill("cloud build finished");
    await build.getByRole("button", { name: "Record build" }).click();
    await expect(detail).toContainText("READY_FOR_APPROVAL");

    const approve = page.getByTestId("release-approval-form");
    await approve.locator("input[name=actor]").fill("approver-1");
    await approve.locator("input[name=rationale]").fill("manifest reviewed");
    await approve.getByRole("button", { name: "Approve release" }).click();
    await expect(page.getByTestId("release-approval")).toContainText("APPROVED");
    await expect(detail).toContainText("APPROVED");

    const deployForm = page.getByTestId("release-deployment-form");
    const deployedTarget = await deployForm.locator("input[name=deployed_target_ref]").inputValue();
    expect(deployedTarget).toMatch(/^cloud-run\//);
    await deployForm.locator("input[name=revision]").fill("oop-staging-00007-abc");
    await deployForm.locator("input[name=service_url]").fill("https://oop-staging.a.run.app");
    await deployForm.locator("input[name=smoke_ref]").fill("https://oop-staging.a.run.app/healthz");
    await deployForm.locator("input[name=reason]").fill("deployed via script");
    await deployForm.getByRole("button", { name: "Record deployment and verify" }).click();
    const receipt = page.getByTestId("release-receipt");
    await expect(receipt).toContainText("PROMOTED");
    await expect(receipt).toContainText("oop-staging-00007-abc");
    await expect(receipt).toContainText(DIGEST);
    await expect(receipt).toContainText(deployedTarget);
    await expect(receipt).toContainText("approver-1");
    await expect(receipt).toContainText("aaa111");
    await expect(page.getByTestId("release-deployment-form")).toHaveCount(0);

    // UI-14: a new release for the same change, approved, then the staging target moves.
    await panel.getByRole("button", { name: "New release" }).click();
    form = page.getByTestId("release-form-create");
    await form.locator(`input[name=change-${good}]`).check();
    await form.locator("input[name=reason]").fill("second release");
    await form.getByRole("button", { name: "Open release and run promotion gate" }).click();
    await expect(detail).toContainText("AWAITING_BUILD");
    const build2 = page.getByTestId("release-build-form");
    await build2.locator("input[name=image_digest]").fill(DIGEST);
    await build2.locator("input[name=source_commit]").fill("aaa111");
    await build2.locator("input[name=reason]").fill("same image");
    await build2.getByRole("button", { name: "Record build" }).click();
    const approve2 = page.getByTestId("release-approval-form");
    await approve2.locator("input[name=actor]").fill("approver-1");
    await approve2.locator("input[name=rationale]").fill("ok");
    await approve2.getByRole("button", { name: "Approve release" }).click();
    await expect(page.getByTestId("release-approval")).toContainText("APPROVED");

    const topology = page.getByTestId("environment-topology");
    await topology.getByTestId("environment-row-staging").getByRole("button", { name: "Edit" }).click();
    const edit = page.getByTestId("environment-form-edit");
    await edit.locator("input[name=target_ref]").fill(`cloud-run/oop-staging-${Date.now().toString(36).slice(-4)}`);
    await edit.locator("input[name=reason]").fill("staging moved after approval");
    await edit.getByRole("button", { name: "Save as new version" }).click();

    await expect(page.getByTestId("release-stale")).toContainText("STALE");
    await expect(page.getByTestId("release-approval")).toContainText("STALE");
    await expect(page.getByTestId("release-deployment-form")).toHaveCount(0);
    await expect(page.getByTestId("release-gates-target_config_drift")).toContainText("STALE");
  });
});
