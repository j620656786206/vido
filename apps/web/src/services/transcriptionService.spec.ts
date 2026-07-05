import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { transcriptionService } from './transcriptionService';

const mockFetch = vi.fn();
global.fetch = mockFetch;

beforeEach(() => {
  mockFetch.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('transcriptionService.startTranscription', () => {
  it('POSTs /movies/{id}/transcribe with translate=true ALWAYS and returns started on 202', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 202,
      json: () =>
        Promise.resolve({
          success: true,
          data: { job_id: 'job-9', message: '已開始轉錄' },
        }),
    });

    const outcome = await transcriptionService.startTranscription(42);

    expect(outcome).toEqual({
      status: 'started',
      result: { jobId: 'job-9', message: '已開始轉錄' },
    });
    const [url, options] = mockFetch.mock.calls[0];
    expect(url).toContain('/movies/42/transcribe?translate=true');
    expect(options).toEqual({ method: 'POST' });
  });

  it('maps 503 TRANSCRIPTION_DISABLED to {status: disabled} without throwing', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 503,
      json: () =>
        Promise.resolve({
          success: false,
          error: { code: 'TRANSCRIPTION_DISABLED', message: 'OpenAI API key not configured' },
        }),
    });

    await expect(transcriptionService.startTranscription(42)).resolves.toEqual({
      status: 'disabled',
    });
  });

  it('maps 409 TRANSCRIPTION_IN_PROGRESS to {status: inProgress} (SSE attach path)', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 409,
      json: () =>
        Promise.resolve({
          success: false,
          error: { code: 'TRANSCRIPTION_IN_PROGRESS', message: 'already running' },
        }),
    });

    await expect(transcriptionService.startTranscription(42)).resolves.toEqual({
      status: 'inProgress',
    });
  });

  it('a bare 503 WITHOUT the TRANSCRIPTION_DISABLED code (reverse-proxy outage) throws fail-soft, not 尚未設定', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 503,
      json: () => Promise.reject(new Error('not json')), // proxy HTML error page
    });

    await expect(transcriptionService.startTranscription(42)).rejects.toThrow(
      'API request failed: 503'
    );
  });

  it('a 409 WITHOUT the TRANSCRIPTION_IN_PROGRESS code throws instead of silently attaching', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 409,
      json: () =>
        Promise.resolve({
          success: false,
          error: { code: 'SOMETHING_ELSE', message: '衝突' },
        }),
    });

    await expect(transcriptionService.startTranscription(42)).rejects.toThrow('衝突');
  });

  it('throws the envelope message for other errors (404 → fail-soft + 重試)', async () => {
    mockFetch.mockResolvedValue({
      ok: false,
      status: 404,
      json: () =>
        Promise.resolve({
          success: false,
          error: { code: 'DB_NOT_FOUND', message: '找不到電影' },
        }),
    });

    await expect(transcriptionService.startTranscription(999)).rejects.toThrow('找不到電影');
  });
});
