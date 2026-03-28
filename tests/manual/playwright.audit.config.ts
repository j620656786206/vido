import { defineConfig, devices } from '@playwright/test';
import path from 'path';

/**
 * Audit config — targets the NAS instance directly, no local server.
 */
export default defineConfig({
  testDir: path.resolve(__dirname),
  outputDir: path.resolve(__dirname, '../../test-results/app-audit'),
  timeout: 60000,
  retries: 0,
  use: {
    baseURL: 'http://192.168.50.52:8088',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
