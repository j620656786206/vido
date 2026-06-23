import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  DiscoverGridSkeletonV2,
  DiscoverNoResultV2,
  DiscoverSectionErrorV2,
} from './DiscoverStatesV2';

describe('DiscoverStatesV2', () => {
  it('skeleton renders the requested number of placeholder cards (I6)', () => {
    render(<DiscoverGridSkeletonV2 count={6} />);
    const skeleton = screen.getByTestId('discover-grid-skeleton');
    expect(skeleton).toBeInTheDocument();
    expect(skeleton.children).toHaveLength(6);
  });

  it('no-result is distinct from empty: echoes active filters + wires clear (I7)', () => {
    const onClear = vi.fn();
    render(
      <DiscoverNoResultV2 activeLabels={['類型: 動作', '年份: 2020 起']} onClearFilters={onClear} />
    );
    expect(screen.getByText('找不到相符的結果')).toBeInTheDocument();
    expect(screen.getByTestId('discover-no-result-echo')).toHaveTextContent(
      '類型: 動作 · 年份: 2020 起'
    );
    fireEvent.click(screen.getByTestId('discover-no-result-clear'));
    expect(onClear).toHaveBeenCalledTimes(1);
  });

  it('no-result omits the echo when no filters are active', () => {
    render(<DiscoverNoResultV2 activeLabels={[]} onClearFilters={vi.fn()} />);
    expect(screen.queryByTestId('discover-no-result-echo')).toBeNull();
  });

  it('per-section error shows the message + code and wires retry (I8 fail-soft)', () => {
    const onRetry = vi.fn();
    render(
      <DiscoverSectionErrorV2
        message="電影結果暫時無法載入，其他結果不受影響"
        code="TMDB_TIMEOUT"
        onRetry={onRetry}
      />
    );
    expect(screen.getByText(/電影結果暫時無法載入/)).toBeInTheDocument();
    expect(screen.getByText(/TMDB_TIMEOUT/)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('discover-section-error-retry'));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
