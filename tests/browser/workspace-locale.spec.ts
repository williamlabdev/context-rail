import { test, expect } from "@playwright/test";

// UI-15: refresh, back and deep links keep the same Project with all of its
// saved state and never mix Projects. UI-16 … UI-19: switching zh-TW / en
// swaps every static copy, survives a refresh, shows human labels for
// machine codes while the code stays readable, and leaves data values alone.
test.describe("Workspace persistence and locale (UI-15 … UI-19)", () => {
  test("UI-15: the selected Project survives refresh, back and a deep link without mixing state", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("project-context-page")).toContainText("Order Operations Portal");
    await expect(page).toHaveURL(/\/projects\/order-operations-portal$/);
    const topology = page.getByTestId("environment-topology");
    await expect(topology).toContainText("Topology v");
    const versionText = (await topology.locator(".topology-heading-meta").innerText()).match(/Topology v(\d+)/)?.[1];
    expect(versionText).toBeTruthy();

    await page.reload();
    await expect(page.getByTestId("project-context-page")).toContainText("Order Operations Portal");
    await expect(topology.locator(".topology-heading-meta")).toContainText(`Topology v${versionText}`);

    await page.getByTestId("project-card-support-insights").click();
    await expect(page).toHaveURL(/\/projects\/support-insights$/);
    await expect(page.getByTestId("project-context-page")).toContainText("Support Insights");
    await expect(page.getByTestId("project-context-page")).not.toContainText("Order Operations Portal");
    await expect(page.getByTestId("changes-empty")).toBeVisible();

    await page.reload();
    await expect(page).toHaveURL(/\/projects\/support-insights$/);
    await expect(page.getByTestId("project-context-page")).toContainText("Support Insights");

    await page.goBack();
    await expect(page).toHaveURL(/\/projects\/order-operations-portal$/);
    await expect(page.getByTestId("project-context-page")).toContainText("Order Operations Portal");
    await expect(topology.locator(".topology-heading-meta")).toContainText(`Topology v${versionText}`);

    await page.goto("/projects/support-insights");
    await expect(page.getByTestId("project-context-page")).toContainText("Support Insights");
    await expect(page.getByTestId("project-card-support-insights")).toHaveAttribute("aria-pressed", "true");
  });

  test("UI-16 … UI-19: zh-TW / en switch, persisted across refresh, human labels with codes, data values untouched", async ({ page }) => {
    await page.goto("/projects/order-operations-portal");
    await expect(page.getByTestId("project-context-page")).toContainText("Order Operations Portal");
    await expect(page.locator(".app-title")).toHaveText("Project Workspace");

    // UI-16: switch to zh-TW — every static copy changes.
    await page.getByTestId("locale-zh-TW").click();
    await expect(page.locator(".app-title")).toHaveText("專案工作區");
    await expect(page.locator("html")).toHaveAttribute("lang", "zh-TW");
    await expect(page.getByRole("heading", { name: "變更", exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { name: "發布", exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { name: "晉升路徑", exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { name: "文件 / AI 情境", exact: true })).toBeVisible();
    await expect(page.getByRole("heading", { name: "就緒度", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "新增變更" })).toBeVisible();
    await expect(page.getByRole("button", { name: "新增環境" })).toBeVisible();
    await expect(page.getByRole("button", { name: "New Change" })).toHaveCount(0);

    // UI-18: machine codes show a human label and keep the code readable.
    const hero = page.locator(".hero-status .status-badge").first();
    await expect(hero).toContainText("情境");
    await expect(hero.locator(".status-code")).toHaveText("DERIVED");
    const productionRow = page.getByTestId("environment-row-production");
    await expect(productionRow).toContainText("啟用");
    await expect(productionRow).toContainText("ACTIVE");

    // UI-19: data values are not translated or polluted.
    await expect(page.getByTestId("project-context-page")).toContainText("Order Operations Portal");
    await expect(page.locator(".source-root code")).toContainText("order-operations-portal");
    // The staging target_ref is whatever the operator last set (earlier journeys move it); it must appear verbatim.
    const stagingTarget = (await page.request.get("/v1/projects/order-operations-portal/environments").then((r) => r.json()));
    const current = stagingTarget.versions.find((v: { version: number }) => v.version === stagingTarget.current_version);
    const stagingRef = current.environments.find((e: { id: string }) => e.id === "staging").target_ref as string;
    expect(stagingRef).toMatch(/^cloud-run\//);
    await expect(page.getByTestId("environment-row-staging")).toContainText(stagingRef);
    await expect(page.getByTestId("environment-row-staging")).toContainText("staging");
    await expect(page.getByTestId("documents-docs/ai/context-pack.json")).toContainText("docs/ai/context-pack.json");
    await expect(page.getByTestId("project-card-order-operations-portal")).toContainText("order-operations-portal");

    // UI-17: refresh keeps the locale and the Project context.
    await page.reload();
    await expect(page).toHaveURL(/\/projects\/order-operations-portal$/);
    await expect(page.locator(".app-title")).toHaveText("專案工作區");
    await expect(page.getByTestId("locale-zh-TW")).toHaveAttribute("aria-pressed", "true");
    await expect(page.getByTestId("project-context-page")).toContainText("Order Operations Portal");
    await page.getByTestId("project-card-support-insights").click();
    await expect(page.getByTestId("project-context-page")).toContainText("Support Insights");
    await expect(page.locator(".app-title")).toHaveText("專案工作區");

    // A zh-TW form still submits the same data: open a Change in zh-TW and read the untranslated id/title back.
    await page.goto("/projects/order-operations-portal");
    await page.getByRole("button", { name: "新增變更" }).click();
    const create = page.getByTestId("change-form-create");
    const title = `Locale check ${Date.now().toString(36).slice(-4)}`;
    await create.locator("input[name=title]").fill(title);
    await create.locator("textarea[name=objective]").fill("Keep data values intact across locales.");
    await create.locator("select[name=target_environment_id]").selectOption("staging");
    await create.locator("input[name=reason]").fill("locale check");
    await create.getByRole("button", { name: "開啟變更並評估" }).click();
    const detail = page.getByTestId("change-detail");
    await expect(detail).toContainText(title);
    await expect(detail).toContainText("需要輸入");
    await expect(detail).toContainText("NEEDS_INPUT");
    await expect(page.getByTestId("change-missing-inputs")).toContainText("acceptance_criteria");

    // Back to English: copy flips, data stays.
    await page.getByTestId("locale-en").click();
    await expect(page.locator(".app-title")).toHaveText("Project Workspace");
    await expect(detail).toContainText(title);
    await expect(detail).toContainText("Needs input");
    await page.reload();
    await expect(page.locator(".app-title")).toHaveText("Project Workspace");
  });
});
