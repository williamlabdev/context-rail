import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "../tests/browser",
  timeout: 15_000,
  // The journeys share one server and one governance state directory; they
  // run one file at a time, in name order, exactly as the Evidence Bundles record them.
  workers: 1,
  fullyParallel: false,
  use: {
    baseURL: process.env.BASE_URL ?? "http://127.0.0.1:8080",
    trace: "retain-on-failure",
    launchOptions: process.env.PLAYWRIGHT_EXECUTABLE_PATH
      ? { executablePath: process.env.PLAYWRIGHT_EXECUTABLE_PATH }
      : undefined,
  },
});
