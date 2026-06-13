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
});
