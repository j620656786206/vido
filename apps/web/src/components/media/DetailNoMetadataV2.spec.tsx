import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { DetailNoMetadataV2 } from './DetailNoMetadataV2';
import { ApiError } from '../../lib/apiError';

function renderBlock(over: Partial<React.ComponentProps<typeof DetailNoMetadataV2>> = {}) {
  const props = {
    variant: 'failed' as const,
    mediaType: 'movie' as const,
    onManualMatch: vi.fn(),
    onRematch: vi.fn(),
    rematching: false,
    ...over,
  };
  render(<DetailNoMetadataV2 {...props} />);
  return props;
}

describe('DetailNoMetadataV2', () => {
  describe('failed (比對失敗)', () => {
    it('says no metadata was found and offers manual match first, re-match second', () => {
      const props = renderBlock();
      const block = screen.getByTestId('detail-no-metadata');
      expect(block).toHaveAttribute('data-variant', 'failed');
      expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('沒有找到這部電影的資料');
      expect(block).toHaveTextContent('自動比對沒有找到符合的作品。你可以自己選對的那一部。');
      expect(block).not.toHaveAttribute('role', 'alert');

      fireEvent.click(screen.getByTestId('no-metadata-manual-match'));
      expect(props.onManualMatch).toHaveBeenCalledTimes(1);
      fireEvent.click(screen.getByTestId('no-metadata-rematch'));
      expect(props.onRematch).toHaveBeenCalledTimes(1);
    });

    it('is honest that re-matching mostly helps after a temporary failure', () => {
      renderBlock();
      expect(screen.getByTestId('detail-no-metadata')).toHaveTextContent(
        '上次如果是網路或 TMDb 暫時出錯，可以再比對一次。'
      );
      expect(screen.getByTestId('detail-no-metadata')).toHaveTextContent(
        '也可以用上方的「修改資訊」自己填片名與年份。'
      );
    });

    it('names the series variant', () => {
      renderBlock({ mediaType: 'series' });
      expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('沒有找到這部影集的資料');
    });

    it('only the manual-match button is the solid accent (one primary per screen)', () => {
      renderBlock();
      expect(screen.getByTestId('no-metadata-manual-match').className).toMatch(
        /bg-\[var\(--accent-primary\)\]/
      );
      expect(screen.getByTestId('no-metadata-rematch').className).not.toMatch(/--accent-primary/);
    });
  });

  describe('pending (資料整理中)', () => {
    it('says the data is still being sorted and offers only 立即比對, with nothing spinning', () => {
      const props = renderBlock({ variant: 'pending' });
      const block = screen.getByTestId('detail-no-metadata');
      expect(block).toHaveAttribute('data-variant', 'pending');
      expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('這部片的資料還在整理');
      expect(block).toHaveTextContent(
        '新加入的檔案會在下次掃描後自動比對。不想等的話，可以現在就比對。'
      );
      expect(screen.queryByTestId('no-metadata-manual-match')).not.toBeInTheDocument();
      expect(block.querySelector('.animate-spin')).toBeNull();

      const now = screen.getByTestId('no-metadata-rematch');
      expect(now).toHaveTextContent('立即比對');
      expect(now.className).toMatch(/bg-\[var\(--accent-primary\)\]/);
      fireEvent.click(now);
      expect(props.onRematch).toHaveBeenCalledTimes(1);
    });
  });

  it('while a re-match runs the button says 比對中…, ignores presses, keeps focus, and only then spins', () => {
    const props = renderBlock({ variant: 'pending', rematching: true });
    const btn = screen.getByTestId('no-metadata-rematch');
    // aria-disabled, not disabled: a disabled button would drop keyboard focus.
    expect(btn).toHaveAttribute('aria-disabled', 'true');
    expect(btn).not.toBeDisabled();
    btn.focus();
    fireEvent.click(btn);
    expect(props.onRematch).not.toHaveBeenCalled();
    expect(btn).toHaveFocus();
    expect(btn).toHaveTextContent('比對中…');
    expect(btn.querySelector('.animate-spin')).not.toBeNull();
  });

  it('a re-match that ran and still found nothing says so', () => {
    renderBlock({ lastRematch: 'still-failed' });
    expect(screen.getByRole('status')).toHaveTextContent('重新比對完成，還是沒有找到。');
  });

  describe('re-match errors', () => {
    it.each([
      [
        new ApiError('busy', 409, 'ENRICHMENT_ALREADY_RUNNING'),
        '媒體庫正在比對其他檔案，請稍後再試。',
        false,
      ],
      [new ApiError('slow', 504, 'METADATA_TIMEOUT'), '比對花太久，已經停止。請稍後再試。', false],
      [new ApiError('db', 500, 'DB_QUERY_FAILED'), '比對失敗，請稍後再試。', true],
    ])('%s → a sentence the user can act on', (error, sentence, showsCode) => {
      renderBlock({ rematchError: error });
      const alert = screen.getByRole('alert');
      expect(alert).toHaveTextContent(sentence);
      const pill = screen.queryByTestId('no-metadata-error-code');
      if (showsCode) expect(pill).toHaveTextContent('DB_QUERY_FAILED');
      else expect(pill).toBeNull();
      expect(alert).not.toHaveTextContent('db');
    });

    it('an error without a code still reads as a sentence, never the raw message', () => {
      renderBlock({ rematchError: new Error('Failed to fetch') });
      expect(screen.getByRole('alert')).toHaveTextContent('比對失敗，請稍後再試。');
      expect(screen.getByRole('alert')).not.toHaveTextContent('Failed to fetch');
    });
  });
});
