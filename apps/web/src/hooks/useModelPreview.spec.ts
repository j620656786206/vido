import { describe, it, expect } from 'vitest';
import { previewFailureMessage } from './useModelPreview';
import { ModelPreviewError } from '../services/subtitleService';

describe('previewFailureMessage (sub-7-8c J10 ④ / C)', () => {
  it('maps the backend codes to the J10 copy and says 沒有扣款 only for a 4xx refusal', () => {
    expect(previewFailureMessage(new ModelPreviewError('x', 429, 'AI_PREVIEW_TOO_SOON'))).toEqual({
      message: '剛剛試跑過，稍後再試',
      unpaid: true,
      code: 'AI_PREVIEW_TOO_SOON',
    });
    expect(
      previewFailureMessage(new ModelPreviewError('x', 409, 'AI_NOT_CONFIGURED')).message
    ).toBe('先到 設定 → API 金鑰 存一組 Claude 金鑰');
    expect(
      previewFailureMessage(new ModelPreviewError('x', 400, 'VALIDATION_INVALID_FORMAT'))
    ).toEqual({
      message: '試跑失敗，沒有扣款',
      unpaid: true,
      code: 'VALIDATION_INVALID_FORMAT',
    });
    // A 5xx may have spent money mid-run: no 沒有扣款 promise.
    const server = previewFailureMessage(new ModelPreviewError('boom', 500, 'INTERNAL_ERROR'));
    expect(server.message).toBe('試跑失敗');
    expect(server.unpaid).toBe(false);
    expect(previewFailureMessage(new TypeError('offline')).message).toBe('試跑失敗');
  });
});
