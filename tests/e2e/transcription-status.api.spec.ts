/**
 * Transcription status API (bugfix-dialog-reopen-shows-idle-during-run)
 *
 * GET /api/v1/{movies|episodes}/{id}/transcribe/status answers whether a
 * subtitle generation is running for one movie or episode, so the 管理字幕
 * dialog can show the progress when reopened mid-run. This is the real-shape
 * check (project-context Rule 28) for the body the web client parses
 * (transcriptionService.getTranscriptionStatus).
 *
 * Prerequisites:
 * - Backend running on port 8080: cd apps/api && go run ./cmd/api
 *
 * @tags @api @subtitle
 */

import { test, expect } from '../support/fixtures';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

// Nothing runs for an id that does not exist — the route never looks it up.
const UNKNOWN_UUID = '9ff0c000-dead-4bee-8f00-000000000999';

test.describe('Transcription status API @api', () => {
  for (const collection of ['movies', 'episodes'] as const) {
    test(`[P1] GET /${collection}/{id}/transcribe/status → {success, data:{in_progress:false}}`, async ({
      request,
    }) => {
      const response = await request.get(
        `${API_BASE_URL}/${collection}/${UNKNOWN_UUID}/transcribe/status`
      );

      expect(response.status()).toBe(200);
      const body = await response.json();
      expect(body).toEqual({ success: true, data: { in_progress: false } });
    });
  }
});
