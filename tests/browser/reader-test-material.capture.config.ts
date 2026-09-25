import { defineConfig } from "@playwright/test";

// Standalone config for tests/browser/reader-test-material.capture.ts — the
// reader-test screenshot capture script. It is intentionally separate from
// frontend/playwright.config.ts (which drives `npm run test:e2e`) so the
// capture script is never discovered or run by the normal e2e suite: its
// testMatch below is the only thing that ever selects that file. See the
// capture script's header comment for how to run this.
export default defineConfig({
  testDir: ".",
  testMatch: /reader-test-material\.capture\.ts$/,
  timeout: 30_000,
  workers: 1,
  fullyParallel: false,
  use: {
    baseURL: process.env.BASE_URL ?? "http://127.0.0.1:8080",
    trace: "retain-on-failure",
    // Below the 900px breakpoint (frontend/src/styles/index.css) the
    // workspace, the Changes list/detail split and the Brief/Agent Context
    // Pack dual-view all collapse to a single column, so the Brief screenshot
    // is one clean reading-width card instead of a cramped multi-column
    // sliver — matching the framing of the existing reader-test screenshots.
    viewport: { width: 860, height: 1400 },
    launchOptions: process.env.PLAYWRIGHT_EXECUTABLE_PATH
      ? { executablePath: process.env.PLAYWRIGHT_EXECUTABLE_PATH }
      : undefined,
  },
});
