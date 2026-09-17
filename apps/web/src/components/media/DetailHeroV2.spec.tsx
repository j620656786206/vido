import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { DetailHeroV2 } from './DetailHeroV2';

describe('DetailHeroV2', () => {
  it('renders the title, original title, status badges, meta and actions', () => {
    render(
      <DetailHeroV2
        title="你的名字"
        originalTitle="君の名は。"
        posterPath={null}
        backdropPath={null}
        badges={[{ label: '已入庫', className: 'x' }, { label: '繁中', className: 'y' }, null]}
        meta={<span>2016</span>}
        actions={<button data-testid="cta">管理字幕</button>}
        onBack={() => {}}
      />
    );
    expect(screen.getByTestId('detail-hero-v2')).toHaveTextContent('你的名字');
    expect(screen.getByTestId('detail-hero-v2')).toHaveTextContent('君の名は。');
    const badges = screen.getAllByTestId('detail-status-badge');
    expect(badges).toHaveLength(2); // null is skipped (F3)
    expect(badges[0]).toHaveTextContent('已入庫');
    expect(screen.getByTestId('cta')).toBeInTheDocument();
  });

  it('calls onBack from the back affordance', () => {
    const onBack = vi.fn();
    render(<DetailHeroV2 title="X" posterPath={null} backdropPath={null} onBack={onBack} />);
    fireEvent.click(screen.getByTestId('detail-back'));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  // dsr-2 AC #7: a failed TMDb image must never leave a broken <img> — fall back to
  // the same filename-hash gradient the null-path branch already renders.
  it('swaps a failed backdrop image for the gradient fallback', () => {
    const { container } = render(
      <DetailHeroV2 title="你的名字" backdropPath="/bd.jpg" posterPath={null} onBack={() => {}} />
    );
    expect(screen.queryByTestId('detail-backdrop-fallback')).not.toBeInTheDocument();
    const img = container.querySelector('img[src$="/bd.jpg"]') as HTMLImageElement;
    expect(img).not.toBeNull();
    fireEvent.error(img);
    expect(screen.getByTestId('detail-backdrop-fallback')).toBeInTheDocument();
    expect(container.querySelector('img[src$="/bd.jpg"]')).toBeNull();
  });

  it('swaps a failed poster image for the initial-letter gradient tile', () => {
    render(
      <DetailHeroV2 title="你的名字" backdropPath={null} posterPath="/p.jpg" onBack={() => {}} />
    );
    expect(screen.queryByTestId('detail-poster-fallback')).not.toBeInTheDocument();
    fireEvent.error(screen.getByAltText('你的名字'));
    expect(screen.getByTestId('detail-poster-fallback')).toHaveTextContent('你');
    expect(screen.queryByAltText('你的名字')).not.toBeInTheDocument();
  });

  // When the next title is already cached, React reuses the SAME DetailHeroV2 instance
  // (no skeleton in between), so a failure flag must not leak onto its working images.
  it('shows the next title’s images again after a failure (state follows the URL)', () => {
    const { container, rerender } = render(
      <DetailHeroV2 title="A" backdropPath="/a-bd.jpg" posterPath="/a-p.jpg" onBack={() => {}} />
    );
    fireEvent.error(container.querySelector('img[src$="/a-bd.jpg"]') as HTMLImageElement);
    fireEvent.error(screen.getByAltText('A'));
    expect(screen.getByTestId('detail-backdrop-fallback')).toBeInTheDocument();
    expect(screen.getByTestId('detail-poster-fallback')).toBeInTheDocument();

    rerender(
      <DetailHeroV2 title="B" backdropPath="/b-bd.jpg" posterPath="/b-p.jpg" onBack={() => {}} />
    );
    expect(container.querySelector('img[src$="/b-bd.jpg"]')).not.toBeNull();
    expect(screen.getByAltText('B')).toBeInTheDocument();
    expect(screen.queryByTestId('detail-backdrop-fallback')).not.toBeInTheDocument();
    expect(screen.queryByTestId('detail-poster-fallback')).not.toBeInTheDocument();
  });

  // dsr-2 AC #6: the back button's hover pair must flip together — a light hover
  // ground under the fixed-light --text-on-scrim glyph vanished in 日巡.
  it('back button hover never pairs a theme-flipping ground with fixed scrim text', () => {
    render(<DetailHeroV2 title="X" posterPath={null} backdropPath={null} onBack={() => {}} />);
    const cls = screen.getByTestId('detail-back').className;
    expect(cls).not.toContain('hover:bg-[var(--bg-tertiary)]');
    expect(cls).toContain('hover:bg-[var(--bg-secondary)]');
    expect(cls).toContain('hover:text-[var(--text-primary)]');
  });

  // dsr-2 AC #6 / dsr-9: shadows are for floating layers only; the poster tile is not one.
  it('poster tile carries no shadow', () => {
    render(<DetailHeroV2 title="X" posterPath={null} backdropPath={null} onBack={() => {}} />);
    expect(screen.getByTestId('detail-poster-tile').className).not.toMatch(/shadow-/);
  });

  // dsr-2b-b AC #6: an unmatched item's title is its raw file name.
  it('the fallback initial skips a leading bracket', () => {
    render(
      <DetailHeroV2
        title="[FanSub] 未知電影"
        posterPath={null}
        backdropPath={null}
        onBack={() => {}}
      />
    );
    expect(screen.getByTestId('detail-poster-fallback')).toHaveTextContent(/^F$/);
  });
});
