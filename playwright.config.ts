import { defineConfig, devices } from '@playwright/test';
import path from 'path';

/**
 * Playwright Configuration for Vido
 *
 * Timeout Standards:
 * - Action timeout: 15s (click, fill, etc.)
 * - Navigation timeout: 30s (page.goto, page.reload)
 * - Expect timeout: 10s (all assertions)
 * - Test timeout: 60s (entire test)
 *
 * @see https://playwright.dev/docs/test-configuration
 */

// Load environment variables
const TEST_ENV = process.env.TEST_ENV || 'local';
const BASE_URL = process.env.BASE_URL || 'http://localhost:4200';
const API_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

export default defineConfig({
  // Test directory
  testDir: path.resolve(__dirname, './tests/e2e'),

  // Output directory for test artifacts
  outputDir: path.resolve(__dirname, './test-results'),

  // Run tests in parallel within single file
  fullyParallel: true,

  // Prevent accidentally committed .only() from blocking CI
  forbidOnly: !!process.env.CI,

  // Retry failed tests in CI
  retries: process.env.CI ? 2 : 0,

  // Worker configuration
  workers: process.env.CI ? 1 : undefined,

  // Global test timeout: 60 seconds
  timeout: 60 * 1000,

  // Expect timeout: 10 seconds
  expect: {
    timeout: 10 * 1000,
  },

  // Reporters
  reporter: [
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
    ['junit', { outputFile: 'test-results/junit.xml' }],
    ['list'],
  ],

  // Shared settings for all projects
  use: {
    // Base URL for navigation
    baseURL: BASE_URL,

    // Action timeout: 15 seconds
    actionTimeout: 15 * 1000,

    // Navigation timeout: 30 seconds
    navigationTimeout: 30 * 1000,

    // Capture trace on first retry (best debugging data)
    trace: 'on-first-retry',

    // Screenshot on failure only
    screenshot: 'only-on-failure',

    // Video on failure only
    video: 'retain-on-failure',

    // Extra HTTP headers
    extraHTTPHeaders: {
      'Accept-Language': 'zh-TW,zh;q=0.9,en;q=0.8',
    },
  },

  // Browser projects
  projects: [
    // Desktop Chrome (primary)
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },

    // Desktop Firefox
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },

    // Desktop Safari
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    // Mobile Chrome
    {
      name: 'mobile-chrome',
      use: { ...devices['Pixel 5'] },
    },

    // Mobile Safari
    {
      name: 'mobile-safari',
      use: { ...devices['iPhone 13'] },
    },
  ],

  // Web server configuration
  webServer: [
    // Backend (Go) - must start first
    ...(process.env.CI
      ? [] // CI starts backend separately
      : [
          {
            command: 'go run ./apps/api/cmd/api',
            url: 'http://localhost:8080/health',
            reuseExistingServer: true,
            timeout: 120 * 1000,
          },
        ]),
    // Frontend (Nx + Vite)
    {
      command: 'npx nx serve web',
      url: 'http://localhost:4200',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
    },
  ],
});
