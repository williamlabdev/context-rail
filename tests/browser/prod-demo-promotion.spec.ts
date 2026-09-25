import { test, expect, type APIRequestContext } from "@playwright/test";

const PROJECT = "order-operations-portal";
const DIGEST = "sha256:7777777777777777777777777777777777777777777777777777777777777777";

interface Env { id: string; type: string; sequence: number; target_ref: string; status: string }

// Make sure an isolated prod-demo environment sits right after staging. It is
// added BEFORE the change is decided, so the decision binds a topology that
// already contains the promotion target (a later add would be a material
// change and make the decision STALE, by design).
async function ensureProdDemo(api: APIRequestContext, base: string): Promise<Env> {
  const url = `${base}/v1/projects/${PROJECT}/environments`;
  let response = await api.get(url);
  expect(response.ok()).toBeTruthy();
  let state = await response.json();
  let current = state.versions.find((version: { version: number }) => version.version === state.current_version);
  let environments = current.environments as Env[];
  let prodDemo = environments.find((environment) => environment.id === "prod-demo");
  if (!prodDemo) {
    const staging = environments.find((environment) => environment.id === "staging");
    expect(staging).toBeTruthy();
    response = await api.post(url, { data: {
      expected_version: state.current_version, actor: "ops", reason: "isolated prod-demo with synthetic data",
      id: "prod-demo", display_name: "Prod demo", type: "prod-demo", sequence: (staging as Env).sequence + 1, target_ref: "cloud-run/order-operations-portal-prod-demo",
      owner: "demo/platform", required_evidence: ["release-approval", "staging-receipt", "smoke"], approver_policy: "distinct human approver",
    } });
    expect(response.ok(), await response.text()).toBeTruthy();
    state = await response.json();
    current = state.versions.find((version: { version: number }) => version.version === state.current_version);
    environments = current.environments as Env[];
    prodDemo = environments.find((environment) => environment.id === "prod-demo");
  }
  expect(prodDemo).toBeTruthy();
  return prodDemo as Env;
}

async function seedAcceptedChange(api: APIRequestContext, base: string, title: string, commit: string): Promise<string> {
  const changes = `${base}/v1/projects/${PROJECT}/changes`;
  let response = await api.post(changes, { data: { reason: "seed", request: {
    title, objective: "seeded change for prod-demo promotion", owner_summary: "seeded owner summary", acceptance_criteria: [{ text: "works" }], allowed_paths: ["main.go", "web/index.html", "main_test.go"],
    target_environment_id: "staging", business_constraints: { data_classification: "internal", expected_monthly_volume: "100" },
  } } });
  expect(response.ok(), await response.text()).toBeTruthy();
  const changeID = (await response.json()).change.change_id as string;
  response = await api.post(`${changes}/${changeID}/decision`, { data: { decision: "ACCEPT", actor: "founder-001", selected_option: "minimal_reversible_slice", rationale: "seed" } });
  expect(response.ok(), await response.text()).toBeTruthy();
  response = await api.post(`${changes}/${changeID}/work-order`, { data: { reason: "seed", actor: "founder-001", issuer: "founder-001" } });
  expect(response.ok(), await response.text()).toBeTruthy();
  const order = (await response.json()).work_order;
  response = await api.post(`${changes}/${changeID}/candidates`, { data: { reason: "seed", actor: "dev-1",
    run: { run_id: `ARR-${changeID}`, work_order_id: order.work_order_id, work_order_hash: order.work_order_hash, started_by: "dev-1", started_at: "2026-09-21T02:00:00Z", agent: { name: "claude-code", provider: "", model: "", version: "" } },
    observation: { branch: order.target_branch, base_branch: order.base_branch, head_commit: commit, changed_paths: ["main.go"], checks: [{ name: "test", status: "PASS" }, { name: "build", status: "PASS" }], reviews: [{ reviewer: "reviewer-2", kind: "human", verdict: "APPROVED" }] } } });
  expect(response.ok(), await response.text()).toBeTruthy();
  response = await api.post(`${changes}/${changeID}/candidates/CAND-001/decision`, { data: { decision: "ACCEPT", actor: "founder-001", rationale: "seed" } });
  expect(response.ok(), await response.text()).toBeTruthy();
  return changeID;
}

// Seed the staging release through the API so the browser journey focuses on the promotion.
async function seedStagingReceipt(api: APIRequestContext, base: string, changeID: string, commit: string, revision: string): Promise<{ releaseID: string; receiptID: string; targetRef: string }> {
  const releases = `${base}/v1/projects/${PROJECT}/releases`;
  let response = await api.post(releases, { data: { reason: "staging release", actor: "release-mgr", change_ids: [changeID], target_environment_id: "staging" } });
  expect(response.ok(), await response.text()).toBeTruthy();
  const releaseID = (await response.json()).release.release_id as string;
  response = await api.post(`${releases}/${releaseID}/build`, { data: { reason: "cloud build", actor: "release-mgr", image_digest: DIGEST, source_commit: commit, build_id: "cb-77", evidence_ref: "cloudbuild/cb-77" } });
  expect(response.ok(), await response.text()).toBeTruthy();
  const targetRef = (await response.json()).release.manifest.environment.target_ref as string;
  response = await api.post(`${releases}/${releaseID}/approval`, { data: { actor: "approver-1", role: "release_manager", decision: "APPROVE", rationale: "staging manifest reviewed" } });
  expect(response.ok(), await response.text()).toBeTruthy();
  response = await api.post(`${releases}/${releaseID}/deployment`, { data: { reason: "deployed staging", actor: "release-mgr", idempotency_key: `k-${releaseID}-staging`, revision, service_url: "https://oop-staging.a.run.app", deployed_digest: DIGEST, deployed_target_ref: targetRef, smoke: { status: "PASS", evidence_ref: "https://oop-staging.a.run.app/healthz" } } });
  expect(response.ok(), await response.text()).toBeTruthy();
  const view = await response.json();
  expect(view.release.status).toBe("PROMOTED");
  return { releaseID, receiptID: view.release.receipt.receipt_id as string, targetRef };
}

test.describe("Prod-demo promotion (VS-007)", () => {
  test("same digest promoted from staging to prod-demo: delta bound by approval, staging revision refused, chained receipt", async ({ page, request, baseURL }) => {
    const base = baseURL ?? "http://127.0.0.1:8080";
    const prodDemo = await ensureProdDemo(request, base);
    const suffix = Date.now().toString(36).slice(-4);
    const changeID = await seedAcceptedChange(request, base, `Prod-demo candidate ${suffix}`, "ccc333");
    const stagingRevision = `oop-staging-00011-${suffix}`;
    const staging = await seedStagingReceipt(request, base, changeID, "ccc333", stagingRevision);

    await page.goto("/");
    const panel = page.getByTestId("releases-panel");
    await expect(panel).toBeVisible();
    await page.getByTestId(`release-card-${staging.releaseID}`).click();
    const detail = page.getByTestId("release-detail");
    await expect(detail).toContainText("PROMOTED");
    await expect(page.getByTestId("release-receipt")).toContainText(staging.receiptID);

    // Promote: the same digest goes to the next environment; no build form, delta shown.
    const promote = page.getByTestId("release-promote-form");
    await expect(promote).toContainText(`from staging (${staging.receiptID})`);
    await promote.locator("select[name=promotion_target]").selectOption("prod-demo");
    await promote.locator("input[name=promotion_reason]").fill("promote reviewed digest to prod-demo");
    await promote.getByRole("button", { name: "Open promotion release" }).click();

    await expect(detail).toContainText("READY_FOR_APPROVAL");
    await expect(detail).toContainText("staging-to-prod-demo");
    const promotionID = (await detail.locator("code").first().innerText()).trim();
    expect(promotionID).not.toBe(staging.releaseID);
    const promotionCard = page.getByTestId("release-promotion");
    await expect(promotionCard).toContainText(staging.releaseID);
    await expect(promotionCard).toContainText(staging.receiptID);
    await expect(promotionCard).toContainText(stagingRevision);
    await expect(page.getByTestId("release-delta-target_ref")).toContainText(prodDemo.target_ref);
    await expect(page.getByTestId("release-delta-type")).toContainText("prod-demo");
    await expect(page.getByTestId("release-build-form")).toHaveCount(0);
    await expect(page.getByTestId("release-gates-build")).toContainText("inherited from");
    await expect(page.getByTestId("release-gates-source_release")).toContainText("PASS");
    await expect(page.getByTestId("release-gates-promotion_order")).toContainText("PASS");
    await expect(page.getByTestId("release-gates-environment_delta")).toContainText("PASS");

    // Third human decision for the promotion, bound to the delta-carrying manifest.
    const approve = page.getByTestId("release-approval-form");
    await approve.locator("input[name=actor]").fill("approver-1");
    await approve.locator("input[name=rationale]").fill("delta reviewed: new target, stricter evidence");
    await approve.getByRole("button", { name: "Approve release" }).click();
    await expect(page.getByTestId("release-approval")).toContainText("APPROVED");

    // Wrong: re-using the staging revision is not a promotion.
    let deploy = page.getByTestId("release-deployment-form");
    expect(await deploy.locator("input[name=deployed_target_ref]").inputValue()).toBe(prodDemo.target_ref);
    expect(await deploy.locator("input[name=deployed_digest]").inputValue()).toBe(DIGEST);
    await deploy.locator("input[name=revision]").fill(stagingRevision);
    await deploy.locator("input[name=reason]").fill("copied the staging revision name");
    await deploy.getByRole("button", { name: "Record deployment and verify" }).click();
    await expect(detail).toContainText("PROMOTION_FAILED");
    await expect(page.getByTestId(`attempt-gates-${promotionID}-D01-new_revision`)).toContainText("BLOCKED");
    await expect(page.getByTestId("release-approval")).toContainText("APPROVED");

    // Right: a new revision on the prod-demo service with the same digest and a passing smoke.
    deploy = page.getByTestId("release-deployment-form");
    await deploy.locator("input[name=revision]").fill(`oop-prod-demo-00001-${suffix}`);
    await deploy.locator("input[name=service_url]").fill("https://oop-prod-demo.a.run.app");
    await deploy.locator("input[name=smoke_ref]").fill("https://oop-prod-demo.a.run.app/healthz");
    await deploy.locator("input[name=reason]").fill("promoted via scripts/record-promotion.sh");
    await deploy.getByRole("button", { name: "Record deployment and verify" }).click();
    const receipt = page.getByTestId("release-receipt");
    await expect(receipt).toContainText("PROMOTED");
    await expect(receipt).toContainText("staging-to-prod-demo");
    await expect(receipt).toContainText(`oop-prod-demo-00001-${suffix}`);
    await expect(receipt).toContainText(DIGEST);
    await expect(receipt).toContainText(prodDemo.target_ref);
    await expect(page.getByTestId("receipt-chain")).toContainText(`Promoted from receipt ${staging.receiptID}`);
    await expect(page.getByTestId("receipt-chain")).toContainText(stagingRevision);
    await expect(page.getByTestId("release-deployment-form")).toHaveCount(0);
    // Production stays blocked: no further promotion is offered after prod-demo.
    await expect(page.getByTestId("release-promote-form")).toHaveCount(0);

    // The same staging receipt cannot be promoted to prod-demo twice.
    await page.getByTestId(`release-card-${staging.releaseID}`).click();
    const again = page.getByTestId("release-promote-form");
    await again.locator("select[name=promotion_target]").selectOption("prod-demo");
    await again.locator("input[name=promotion_reason]").fill("try again");
    await again.getByRole("button", { name: "Open promotion release" }).click();
    await expect(detail).toContainText("GATE_BLOCKED");
    await expect(page.getByTestId("release-gates-source_release")).toContainText("already promoted");
    await expect(page.getByTestId("release-approval-form")).toHaveCount(0);
  });
});
