import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { browserName: 'chromium' },
    },
  ],
  // Uncomment to auto-start the Go backend before tests:
  // webServer: {
  //   command: 'cd .. && make dev-go',
  //   url: 'http://localhost:8080',
  //   reuseExistingServer: true,
  //   timeout: 30_000,
  // },
});
