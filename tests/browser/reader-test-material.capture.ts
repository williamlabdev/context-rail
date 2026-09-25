import { test } from "@playwright/test";
import * as path from "node:path";

// Captures the two screenshots used as material for the Change Decision
// Brief reader test (docs/validation/CONTEXT_RAIL_BRIEF_READER_TEST_*.zh-TW.md
// §2), by driving the real Change → decision flow twice through the UI —
// once with the request content written in English, once with the same
// content faithfully written in Traditional Chinese by the requester. The
// renderer never translates data (frontend/src/components/ChangeBrief.tsx,
// see its header comment); the zh-TW screenshot looks different from the en
// one only because the requester wrote the free-text fields in Chinese, the
// same way a real bilingual requester would produce two separate records.
//
// NOT part of `npm run test:e2e`: this file's name does not end in
// `.spec.ts` / `.test.ts`, so the default suite's testMatch
// (frontend/playwright.config.ts) never discovers it, and it is also
// excluded by name from reader-test-material.capture.config.ts's own
// testMatch when that config is pointed elsewhere. Run it standalone with
// its own config, from frontend/, against a server on a FRESH, empty
// CONTEXT_RAIL_STATE_DIR (fresh so the first Change in the run is DEC-001;
// see the doc's §2 for why the two screenshots do not need to share one
// decision id):
//
//   cd frontend && PLAYWRIGHT_EXECUTABLE_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
//     BASE_URL=http://127.0.0.1:8094 \
//     npx playwright test --config=../tests/browser/reader-test-material.capture.config.ts
//
// Only free text a requester would actually write is translated (owner
// summary, scope included/excluded, decision rationale, the technical
// objective). Everything else — ids, env names, evidence/option codes, the
// "minimal_reversible_slice" approach, staging as the target — is identical
// in both runs and is already rendered in the reader's language by the
// renderer itself (frontend/src/i18n/advisor.ts), never by this script.

const outDir = path.resolve(__dirname, "../../docs/validation/reader-test-materials");

interface RequestContent {
  title: string;
  objective: string;
  ownerSummary: string;
  scopeIncluded: string;
  scopeExcluded: string;
  acceptanceCriteria: string;
  expectedVolume: string;
  reason: string;
  rationale: string;
}

const en: RequestContent = {
  title: "Order status visibility for support",
  objective:
    "Let the support console show order fulfillment and shipping status read-only, without touching payment or refund flows.",
  ownerSummary:
    "Customer support can see an order's fulfillment and shipping status without asking engineering. Payments and refunds are unchanged.",
  scopeIncluded: "Read-only order status and shipping milestones in the support console",
  scopeExcluded: "Payment processing\nRefund issuance",
  acceptanceCriteria:
    "Support console shows order status and shipping milestones\nPayment and refund flows are unchanged",
  expectedVolume: "10,000 lookups/month",
  reason: "reader test material capture",
  rationale:
    "Smallest reversible slice; read-only, keeps payments and refunds untouched, proves the path to staging.",
};

// Natural Taiwan Traditional Chinese, written by the requester with the same
// meaning as `en` above — not a translation performed by the renderer.
const zhTW: RequestContent = {
  title: "客服可查訂單出貨狀態",
  objective: "讓客服主控台唯讀顯示訂單的出貨與配送狀態，不動到付款或退款流程。",
  ownerSummary: "客服人員不用問工程部門，就能查到訂單的出貨與配送狀態。付款與退款不變。",
  scopeIncluded: "客服主控台中唯讀的訂單狀態與出貨里程碑",
  scopeExcluded: "付款處理\n退款發放",
  acceptanceCriteria: "客服主控台顯示訂單狀態與出貨里程碑\n付款與退款流程不變",
  expectedVolume: "每月 10,000 次查詢",
  reason: "讀者測試素材擷取",
  rationale: "最小、可退回的一步；唯讀，不動到付款與退款，先證明能走到 staging。",
};

async function captureBrief(
  page: import("@playwright/test").Page,
  locale: "en" | "zh-TW",
  content: RequestContent,
  fileName: string,
): Promise<void> {
  await page.goto("/projects/order-operations-portal");
  if (locale === "zh-TW") {
    await page.evaluate(() => localStorage.setItem("context-rail.locale", "zh-TW"));
    await page.reload();
  }
  const changesPanel = page.getByTestId("changes-panel");
  await changesPanel.waitFor();

  // "New Change" has no data-testid and its label is localized, so target it
  // by position/class (the only button.button-primary directly under the
  // list, scoped to the Changes panel so it can't match another panel's
  // same-classed button, e.g. Releases' "New release") instead of by
  // translated text.
  await changesPanel.locator(".changes-list > button.button-primary").click();
  const create = page.getByTestId("change-form-create");
  await create.locator("input[name=title]").fill(content.title);
  await create.locator("select[name=target_environment_id]").selectOption("staging");
  await create.locator("textarea[name=owner_summary]").fill(content.ownerSummary);
  await create.locator("textarea[name=objective]").fill(content.objective);
  await create.locator("textarea[name=scope_included]").fill(content.scopeIncluded);
  await create.locator("textarea[name=scope_excluded]").fill(content.scopeExcluded);
  await create.locator("textarea[name=acceptance_criteria]").fill(content.acceptanceCriteria);
  await create.locator("textarea[name=allowed_paths]").fill("main.go\nweb/index.html\nmain_test.go");
  await create.locator("input[name=data_classification]").fill("internal");
  await create.locator("input[name=expected_monthly_volume]").fill(content.expectedVolume);
  await create.locator("input[name=reason]").fill(content.reason);
  await create.locator("button.button-primary[type=submit]").click();

  // All required inputs were supplied up front, so the Change goes straight
  // to DECISION_READY — no NEEDS_INPUT round trip needed.
  const decisionForm = page.getByTestId("change-decision-form");
  await decisionForm.waitFor();
  // The advisor recommends "minimal_reversible_slice" by default; select it
  // explicitly so the capture does not depend on that default.
  await decisionForm.locator("input[name=selected_option][value=minimal_reversible_slice]").check();
  await decisionForm.locator("input[name=actor]").fill("founder-001");
  await decisionForm.locator("textarea[name=rationale]").fill(content.rationale);
  await decisionForm.locator("button.button-primary[type=submit]").click();

  const brief = page.getByTestId("change-brief");
  await brief.getByTestId("brief-decided-by").waitFor();
  await brief.getByTestId("brief-trace").waitFor();
  // Traceability starts collapsed — same framing as the current screenshots.
  await brief.screenshot({ path: path.join(outDir, fileName) });
}

test("capture Change Decision Brief reader-test screenshots (en + zh-TW)", async ({ page }) => {
  await captureBrief(page, "en", en, "brief-en.png");
  // A fresh Change, not an edit of the first: the en and zh-TW screenshots
  // are two separate records with the same content, exactly as
  // docs/validation/CONTEXT_RAIL_BRIEF_READER_TEST_2026-09-25.zh-TW.md §2 says.
  await captureBrief(page, "zh-TW", zhTW, "brief-zh-TW.png");
});
