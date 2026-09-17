import { describe, it, expect } from 'vitest';
import { ApiError } from '../../lib/apiError';
import type { TranscriptionEstimate } from '../../services/transcriptionService';
import {
  deriveGenerateCostView,
  FALLBACK_RUNTIME_LINE,
  type EstimateQueryState,
} from './generateCostView';

// Story dsr-6a AC #5 — every state the paid 生成字幕／重試 buttons can be in,
// and the one line that explains it (J9-D six states + the SM supplement).

function estimate(overrides: Partial<TranscriptionEstimate> = {}): TranscriptionEstimate {
  return {
    mediaId: 'm1',
    mediaType: 'movie',
    plan: 'full',
    asrAvailable: true,
    selfHostedAsr: false,
    translationConfigured: true,
    modelId: 'claude-sonnet-5',
    runtimeMinutes: 48,
    runtimeKnown: true,
    runtimeSource: 'ffprobe',
    estimatedUsd: 0.67,
    ...overrides,
  };
}

const ready = (overrides: Partial<TranscriptionEstimate> = {}): EstimateQueryState => ({
  data: estimate(overrides),
  isError: false,
  error: null,
});
const loading: EstimateQueryState = { data: undefined, isError: false, error: null };
const failed = (error: unknown): EstimateQueryState => ({ data: undefined, isError: true, error });

describe('deriveGenerateCostView', () => {
  it('series: no generate route — disabled, points at the episode list, no amount', () => {
    const view = deriveGenerateCostView({ mediaType: 'series', estimate: loading });
    expect(view.cost).toEqual({ status: 'unavailable' });
    expect(view.helper).toEqual({
      text: '請於下方分集清單逐集生成',
      tone: 'secondary',
      settingsLink: false,
    });
  });

  it('① ready with a measured runtime: amount + the default line', () => {
    const view = deriveGenerateCostView({ mediaType: 'movie', estimate: ready() });
    expect(view.cost).toEqual({ status: 'ready', usd: 0.67, approximate: false });
    expect(view.helper).toEqual({
      text: '語音辨識＋AI 翻譯，約需數分鐘',
      tone: 'muted',
      settingsLink: false,
    });
    expect(view.retryNote).toBeNull();
  });

  it('② assumed runtime: ≈ on the amount and the J9-D ② line', () => {
    const view = deriveGenerateCostView({
      mediaType: 'episode',
      estimate: ready({ runtimeSource: 'fallback', runtimeKnown: false, runtimeMinutes: 45 }),
    });
    expect(view.cost).toEqual({ status: 'ready', usd: 0.67, approximate: true });
    expect(view.helper.text).toBe('片長未知（估 45 分）——實際費用依內容長度而定');
    expect(view.retryNote).toEqual({ text: FALLBACK_RUNTIME_LINE, settingsLink: false });
  });

  it('③ self-hosted ASR without a translation key: $0.00 explained by both halves', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: ready({
        selfHostedAsr: true,
        translationConfigured: false,
        estimatedUsd: 0,
        modelId: '',
      }),
    });
    expect(view.cost).toEqual({ status: 'ready', usd: 0, approximate: false });
    expect(view.helper).toEqual({
      text: '語音辨識：自架（不另計費）。僅能產生英文字幕——尚未設定翻譯金鑰',
      tone: 'muted',
      settingsLink: true,
    });
  });

  it('self-hosted ASR WITH translation keeps the default line (the amount is not zero)', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: ready({ selfHostedAsr: true, estimatedUsd: 0.39 }),
    });
    expect(view.helper.text).toBe('語音辨識＋AI 翻譯，約需數分鐘');
  });

  it('④ estimate on its way: skeleton, and the line follows the props (untranslated → cheap line)', () => {
    const movie = deriveGenerateCostView({ mediaType: 'movie', estimate: loading });
    expect(movie.cost).toEqual({ status: 'loading' });
    expect(movie.helper).toEqual({
      text: '語音辨識＋AI 翻譯，約需數分鐘',
      tone: 'muted',
      settingsLink: false,
    });

    const resumable = deriveGenerateCostView({
      mediaType: 'movie',
      subtitleStatus: 'untranslated',
      estimate: loading,
    });
    expect(resumable.helper.text).toBe('僅需翻譯，不再重跑語音辨識——這次很快也很便宜');
    expect(resumable.retryNote).toBeNull();
  });

  it('⑤ estimate failed: disabled with the reason, in the secondary tone', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: failed(new ApiError('boom', 500, 'INTERNAL_ERROR')),
    });
    expect(view.cost).toEqual({ status: 'unavailable' });
    expect(view.helper).toEqual({
      text: '暫時算不出費用，因此先不開放。重新整理或稍後再試。',
      tone: 'secondary',
      settingsLink: false,
    });
    expect(view.retryNote).toEqual({ text: view.helper.text, settingsLink: false });
  });

  it('the video file cannot be read: its own reason, not 「稍後再試」', () => {
    const view = deriveGenerateCostView({
      mediaType: 'episode',
      estimate: failed(new ApiError('找不到這一集的媒體檔案', 400, 'VALIDATION_REQUIRED_FIELD')),
    });
    expect(view.cost).toEqual({ status: 'unavailable' });
    expect(view.helper.text).toBe(
      '讀不到影片檔案，因此先不開放。請確認檔案還在，或重新掃描媒體庫。'
    );
  });

  it('⑥ speech recognition not configured: disabled before the click, with 前往設定', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: ready({ asrAvailable: false }),
    });
    expect(view.cost).toEqual({ status: 'unavailable' });
    expect(view.helper).toEqual({
      text: '生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。',
      tone: 'secondary',
      settingsLink: true,
    });
    expect(view.retryNote).toEqual({ text: view.helper.text, settingsLink: true });
  });

  it('a resumable row needs no ASR, so a missing ASR key does not block it', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: ready({ plan: 'translate_only', asrAvailable: false, estimatedUsd: 0.39 }),
    });
    expect(view.cost).toEqual({ status: 'ready', usd: 0.39, approximate: false });
    expect(view.helper.text).toBe('僅需翻譯，不再重跑語音辨識——這次很快也很便宜');
  });

  it('only translation is missing but there is no translation key: disabled (the click would produce nothing)', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: ready({ plan: 'translate_only', translationConfigured: false, estimatedUsd: 0 }),
    });
    expect(view.cost).toEqual({ status: 'unavailable' });
    expect(view.helper).toEqual({
      text: '尚未設定翻譯金鑰',
      tone: 'secondary',
      settingsLink: true,
    });
  });

  it('helper precedence: translate-only beats the missing-key line and the ≈ line', () => {
    const view = deriveGenerateCostView({
      mediaType: 'episode',
      estimate: ready({ plan: 'translate_only', runtimeSource: 'fallback', runtimeKnown: false }),
    });
    expect(view.cost).toEqual({ status: 'ready', usd: 0.67, approximate: true });
    expect(view.helper.text).toBe('僅需翻譯，不再重跑語音辨識——這次很快也很便宜');
    // The retry panels have no helper line, so there the ≈ is explained.
    expect(view.retryNote).toEqual({ text: FALLBACK_RUNTIME_LINE, settingsLink: false });
  });

  it('helper precedence: the missing-key line beats the ≈ line', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: ready({
        translationConfigured: false,
        runtimeSource: 'fallback',
        runtimeKnown: false,
      }),
    });
    expect(view.helper.text).toBe('僅能產生英文字幕——尚未設定翻譯金鑰');
    expect(view.helper.settingsLink).toBe(true);
  });

  it('the retry panels explain an English-only (or $0.00) retry too — a zero must say why', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: ready({
        selfHostedAsr: true,
        translationConfigured: false,
        estimatedUsd: 0,
        modelId: '',
      }),
    });
    expect(view.retryNote).toEqual({
      text: '語音辨識：自架（不另計費）。僅能產生英文字幕——尚未設定翻譯金鑰',
      settingsLink: true,
    });
  });

  it('a refetch that says the file is gone wins over the stale price (the click could only fail)', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: {
        data: estimate(),
        isError: true,
        error: new ApiError('Movie file not accessible', 400, 'VALIDATION_REQUIRED_FIELD'),
      },
    });
    expect(view.cost).toEqual({ status: 'unavailable' });
    expect(view.helper.text).toBe(
      '讀不到影片檔案，因此先不開放。請確認檔案還在，或重新掃描媒體庫。'
    );
  });

  it('data first: a failed BACKGROUND refetch keeps the price that is already on screen', () => {
    const view = deriveGenerateCostView({
      mediaType: 'movie',
      estimate: { data: estimate(), isError: true, error: new Error('network') },
    });
    expect(view.cost).toEqual({ status: 'ready', usd: 0.67, approximate: false });
  });
});
