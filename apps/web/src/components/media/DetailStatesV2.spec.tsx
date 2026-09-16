import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { DetailSkeletonV2, DetailNotFoundV2, DetailLoadErrorV2 } from './DetailStatesV2';

describe('DetailStatesV2', () => {
  it('skeleton is busy and renders no spinner', () => {
    render(<DetailSkeletonV2 />);
    expect(screen.getByTestId('detail-skeleton')).toHaveAttribute('aria-busy', 'true');
  });

  it('not-found shows the message and calls onBack from both affordances', () => {
    const onBack = vi.fn();
    render(<DetailNotFoundV2 onBack={onBack} />);
    expect(screen.getByTestId('detail-not-found')).toHaveTextContent('找不到這部影片');
    // B6p-D copy — says WHERE it was removed from.
    expect(screen.getByTestId('detail-not-found')).toHaveTextContent(
      '這個項目可能已從媒體庫移除，或連結已失效。'
    );
    fireEvent.click(screen.getByTestId('detail-back'));
    fireEvent.click(screen.getByText('返回媒體庫'));
    expect(onBack).toHaveBeenCalledTimes(2);
  });

  // A TMDb title (Discover/Home) or a bad route was never in the library — saying it
  // "may have been removed from your library" would be false (review #3).
  it('not-found outside the library does not mention the library', () => {
    render(<DetailNotFoundV2 onBack={() => {}} inLibrary={false} />);
    const panel = screen.getByTestId('detail-not-found');
    expect(panel).toHaveTextContent('這個連結可能已失效。');
    expect(panel).not.toHaveTextContent('媒體庫移除');
  });

  // dsr-2 AC #8: a failed load is not a missing item.
  describe('DetailLoadErrorV2', () => {
    it('says it could not load, reassures about the files, and shows the code pill', () => {
      const onRetry = vi.fn();
      const onBack = vi.fn();
      render(
        <DetailLoadErrorV2 code="DB_QUERY_FAILED" reassureFiles onRetry={onRetry} onBack={onBack} />
      );
      const panel = screen.getByTestId('detail-load-error');
      expect(panel).toHaveTextContent('無法載入這部影片');
      expect(panel).not.toHaveTextContent('找不到');
      const reassure = screen.getByText('詳情資料查詢失敗，你的檔案沒有受影響。');
      // The sentence exists to calm, so it must not wear alarm red (dsr-1 CR #4).
      expect(reassure.className).toContain('text-[var(--text-secondary)]');
      expect(reassure.className).not.toContain('error');
      expect(screen.getByTestId('detail-load-error-code')).toHaveTextContent('DB_QUERY_FAILED');
      fireEvent.click(screen.getByTestId('detail-load-error-retry'));
      expect(onRetry).toHaveBeenCalledTimes(1);
      fireEvent.click(screen.getByTestId('detail-back'));
      fireEvent.click(screen.getByText('返回媒體庫'));
      expect(onBack).toHaveBeenCalledTimes(2);
    });

    // A TMDb-only title is not in the library, so "your files are fine" would mislead.
    it('omits the file reassurance and the code pill when not applicable', () => {
      render(<DetailLoadErrorV2 onRetry={() => {}} onBack={() => {}} />);
      expect(screen.queryByText(/你的檔案沒有受影響/)).not.toBeInTheDocument();
      expect(screen.queryByTestId('detail-load-error-code')).not.toBeInTheDocument();
      // Still says something — never a bare title + buttons (review #15).
      expect(screen.getByText('詳情資料暫時無法取得，請稍後再試。')).toBeInTheDocument();
    });

    // A retry that fails again must not look like a dead button (review #5).
    it('shows the retry in flight and blocks double-submits', () => {
      const onRetry = vi.fn();
      render(<DetailLoadErrorV2 onRetry={onRetry} onBack={() => {}} retrying />);
      const retry = screen.getByTestId('detail-load-error-retry');
      expect(retry).toBeDisabled();
      expect(retry).toHaveTextContent('重試中…');
      fireEvent.click(retry);
      expect(onRetry).not.toHaveBeenCalled();
    });
  });
});
