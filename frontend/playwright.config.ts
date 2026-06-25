import { defineConfig } from "@playwright/test";

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? "http://127.0.0.1:5173";
const webServerCommand = process.env.PLAYWRIGHT_WEB_SERVER_COMMAND;
const useLocalServer = process.env.PLAYWRIGHT_BASE_URL === undefined;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL,
    screenshot: "only-on-failure",
    trace: "on-first-retry",
  },
  webServer: webServerCommand
    ? {
        command: webServerCommand,
        url: baseURL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      }
    : useLocalServer
    ? {
        command: "pnpm dev --host 127.0.0.1",
        url: baseURL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      }
    : undefined,
  projects: [
    {
      name: "chromium",
      use: {
        browserName: "chromium",
      },
    },
  ],
});
