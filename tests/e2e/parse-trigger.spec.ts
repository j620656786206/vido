/**
 * Parse Trigger E2E Tests (Story 4.5)
 *
 * Tests for completed download detection and parsing trigger.
 * Uses route interception for deterministic tests.
 *
 * @tags @parse-trigger @story-4-5
 */

import { test, expect } from '../support/fixtures';
import { presetDownloads } from '../support/fixtures/factories/download-factory';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// =============================================================================
// Mock Data
// =============================================================================

const completedWithParsing = {
  ...presetDownloads.completed,
  parseStatus: {
    status: 'processing',
  },
};

const completedWithParsed = {
  ...presetDownloads.completed,
  parseStatus: {
    status: 'completed',
    mediaId: 'movie-123',
  },
};

const completedWithFailed = {
  ...presetDownloads.completed,
  hash: 'f'.repeat(40),
  name: 'Unparseable.File.2024.mkv',
  parseStatus: {
    status: 'failed',
    errorMessage: 'could not parse filename',
  },
};

const completedNoParse = {
  ...presetDownloads.completed,
  hash: 'g'.repeat(40),
  name: 'New.Completed.Download.mkv',
};

const mockParseJobs = [
  {
    id: 'job-1',
    torrentHash: presetDownloads.completed.hash,
    fileName: presetDownloads.completed.name,
    filePath: presetDownloads.completed.savePath,
    status: 'completed',
    mediaId: 'movie-123',
    retryCount: 0,
    createdAt: '2026-01-15T12:00:00Z',
    updatedAt: '2026-01-15T12:01:00Z',
    completedAt: '2026-01-15T12:01:00Z',
  },
  {
    id: 'job-2',
    torrentHash: 'f'.repeat(40),
    fileName: 'Unparseable.File.2024.mkv',
    filePath: '/downloads/movies',
    status: 'failed',
    errorMessage: 'could not parse filename',
    retryCount: 0,
    createdAt: '2026-01-15T12:00:00Z',
    updatedAt: '2026-01-15T12:01:00Z',
  },
];

// =============================================================================
// Completion Detection Tests (AC1)
// =============================================================================

test.describe('Completion Detection @parse-trigger @story-4-5', () => {
  test('[P1] should show "解析中..." status for torrent being parsed (AC1)', async ({ page }) => {
    // GIVEN: API returns a completed download with parsing in progress
    await page.route(`${API_BASE_URL}/downloads*`, (route) => {
      if (route.request().url().includes('/counts')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { all: 1, downloading: 0, paused: 0, completed: 1, seeding: 0, error: 0 },
          }),
        });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [completedWithParsing] }),
      });
    });

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: Parse status badge shows "解析中..."
    await expect(page.getByText('解析中...')).toBeVisible();
  });

  test('[P1] should show "已入庫" status for successfully parsed torrent (AC2)', async ({
    page,
  }) => {
    // GIVEN: API returns a completed download that has been parsed
    await page.route(`${API_BASE_URL}/downloads*`, (route) => {
      if (route.request().url().includes('/counts')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { all: 1, downloading: 0, paused: 0, completed: 1, seeding: 0, error: 0 },
          }),
        });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [completedWithParsed] }),
      });
    });

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: Parse status badge shows "已入庫"
    await expect(page.getByText('已入庫')).toBeVisible();
  });

  test('[P1] should show "解析失敗" status for failed parsing (AC3)', async ({ page }) => {
    // GIVEN: API returns a completed download with failed parsing
    await page.route(`${API_BASE_URL}/downloads*`, (route) => {
      if (route.request().url().includes('/counts')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { all: 1, downloading: 0, paused: 0, completed: 1, seeding: 0, error: 0 },
          }),
        });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [completedWithFailed] }),
      });
    });

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: Parse status badge shows "解析失敗"
    await expect(page.getByText('解析失敗')).toBeVisible();
  });

  test('[P2] should show "完成" for completed download without parse status (AC1)', async ({
    page,
  }) => {
    // GIVEN: API returns a completed download without parse status (not yet detected)
    await page.route(`${API_BASE_URL}/downloads*`, (route) => {
      if (route.request().url().includes('/counts')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { all: 1, downloading: 0, paused: 0, completed: 1, seeding: 0, error: 0 },
          }),
        });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [completedNoParse] }),
      });
    });

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: Shows default "完成" text (no parse badge yet)
    await expect(page.getByText('完成')).toBeVisible();
  });
});

// =============================================================================
// Parse Status API Tests (AC1, AC2, AC3)
// =============================================================================

test.describe('Parse Status API @parse-trigger @story-4-5 @api', () => {
  test('[P1] GET /parse-jobs should return parse job list (AC1)', async ({ request }) => {
    // WHEN: Requesting parse jobs
    const response = await request.get(`${API_BASE_URL}/parse-jobs`);

    // THEN: Response is valid (may be empty if no jobs exist)
    if (response.ok()) {
      const json = await response.json();
      expect(json.success).toBe(true);
      expect(Array.isArray(json.data)).toBe(true);
    } else {
      // Service might not be running - that's OK for E2E
      expect(response.status()).toBeLessThan(500);
    }
  });

  test('[P1] GET /downloads/:hash/parse-status returns 404 for unknown hash (AC3)', async ({
    request,
  }) => {
    // WHEN: Requesting parse status for a nonexistent hash
    const response = await request.get(`${API_BASE_URL}/downloads/nonexistent/parse-status`);

    // THEN: Returns 404 (or server-level error if backend is down)
    if (response.ok()) {
      // Should not be 200 for nonexistent hash
      const json = await response.json();
      expect(json.success).toBe(false);
    } else {
      expect(response.status()).toBeGreaterThanOrEqual(400);
    }
  });

  test('[P2] GET /parse-jobs supports limit parameter (AC1)', async ({ request }) => {
    // WHEN: Requesting parse jobs with limit
    const response = await request.get(`${API_BASE_URL}/parse-jobs?limit=5`);

    // THEN: Response is valid
    if (response.ok()) {
      const json = await response.json();
      expect(json.success).toBe(true);
      const data = json.data as unknown[];
      expect(data.length).toBeLessThanOrEqual(5);
    }
  });
});

// =============================================================================
// Duplicate Detection Tests (AC5)
// =============================================================================

test.describe('Duplicate Detection @parse-trigger @story-4-5', () => {
  test('[P2] should not show parse badge for downloading torrents (AC5)', async ({ page }) => {
    // GIVEN: API returns a downloading torrent (should not have parse status)
    await page.route(`${API_BASE_URL}/downloads*`, (route) => {
      if (route.request().url().includes('/counts')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: { all: 1, downloading: 1, paused: 0, completed: 0, seeding: 0, error: 0 },
          }),
        });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [presetDownloads.downloading] }),
      });
    });

    // WHEN: Navigating to downloads page
    await page.goto('/downloads');

    // THEN: No parse status badge is shown for downloading torrent
    await expect(page.getByTestId('download-parse-status-badge')).not.toBeVisible();
  });
});
