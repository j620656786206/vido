import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { MetadataSourceBadge } from './MetadataSourceBadge';

describe('MetadataSourceBadge', () => {
  it('renders TMDb badge with correct icon and label', () => {
    render(<MetadataSourceBadge source="tmdb" />);
    const badge = screen.getByTestId('metadata-source-badge');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent('🎬');
    expect(badge).toHaveTextContent('TMDb');
  });

  it('renders Douban badge', () => {
    render(<MetadataSourceBadge source="douban" />);
    expect(screen.getByTestId('metadata-source-badge')).toHaveTextContent('豆瓣');
  });

  it('renders Wikipedia badge', () => {
    render(<MetadataSourceBadge source="wikipedia" />);
    expect(screen.getByTestId('metadata-source-badge')).toHaveTextContent('Wikipedia');
  });

  it('renders AI badge', () => {
    render(<MetadataSourceBadge source="ai" />);
    expect(screen.getByTestId('metadata-source-badge')).toHaveTextContent('AI 解析');
  });

  it('renders Manual badge', () => {
    render(<MetadataSourceBadge source="manual" />);
    expect(screen.getByTestId('metadata-source-badge')).toHaveTextContent('手動輸入');
  });

  it('shows tooltip with source and fetch date', () => {
    render(<MetadataSourceBadge source="tmdb" fetchDate="2026-01-10T00:00:00Z" />);
    const badge = screen.getByTestId('metadata-source-badge');
    expect(badge.getAttribute('title')).toContain('資料來源：TMDb');
    expect(badge.getAttribute('title')).toContain('取得');
  });

  it('shows tooltip without date when no fetchDate', () => {
    render(<MetadataSourceBadge source="ai" />);
    const badge = screen.getByTestId('metadata-source-badge');
    expect(badge.getAttribute('title')).toBe('資料來源：AI 解析');
  });

  it('returns null for unknown source', () => {
    const { container } = render(<MetadataSourceBadge source="unknown" />);
    expect(container.firstChild).toBeNull();
  });
});
