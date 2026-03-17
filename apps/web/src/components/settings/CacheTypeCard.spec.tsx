import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { CacheTypeCard, formatBytes } from './CacheTypeCard';
import type { CacheTypeInfo } from '../../services/cacheService';

const mockCacheType: CacheTypeInfo = {
  type: 'ai',
  label: 'AI 解析快取',
  sizeBytes: 52428800,
  entryCount: 120,
};

describe('CacheTypeCard', () => {
  it('renders cache type label', () => {
    render(
      React.createElement(CacheTypeCard, {
        cacheType: mockCacheType,
        onClear: vi.fn(),
      })
    );
    expect(screen.getByTestId('cache-type-label')).toHaveTextContent('AI 解析快取');
  });

  it('renders cache size and count', () => {
    render(
      React.createElement(CacheTypeCard, {
        cacheType: mockCacheType,
        onClear: vi.fn(),
      })
    );
    expect(screen.getByTestId('cache-type-size')).toHaveTextContent('50.0 MB');
    expect(screen.getByTestId('cache-type-size')).toHaveTextContent('120 筆');
  });

  it('shows confirm button on first click', () => {
    render(
      React.createElement(CacheTypeCard, {
        cacheType: mockCacheType,
        onClear: vi.fn(),
      })
    );

    const clearBtn = screen.getByTestId('cache-clear-btn');
    expect(clearBtn).toHaveTextContent('清除');

    fireEvent.click(clearBtn);
    expect(clearBtn).toHaveTextContent('確認清除');
  });

  it('shows cancel button during confirmation', () => {
    render(
      React.createElement(CacheTypeCard, {
        cacheType: mockCacheType,
        onClear: vi.fn(),
      })
    );

    fireEvent.click(screen.getByTestId('cache-clear-btn'));
    expect(screen.getByTestId('cache-cancel-btn')).toBeInTheDocument();
  });

  it('cancels confirmation on cancel click', () => {
    render(
      React.createElement(CacheTypeCard, {
        cacheType: mockCacheType,
        onClear: vi.fn(),
      })
    );

    fireEvent.click(screen.getByTestId('cache-clear-btn'));
    fireEvent.click(screen.getByTestId('cache-cancel-btn'));
    expect(screen.getByTestId('cache-clear-btn')).toHaveTextContent('清除');
  });

  it('calls onClear on second click (confirm)', async () => {
    const onClear = vi.fn().mockResolvedValue(undefined);
    render(
      React.createElement(CacheTypeCard, {
        cacheType: mockCacheType,
        onClear,
      })
    );

    const clearBtn = screen.getByTestId('cache-clear-btn');
    fireEvent.click(clearBtn); // first click — enters confirm mode
    fireEvent.click(clearBtn); // second click — executes

    await waitFor(() => {
      expect(onClear).toHaveBeenCalledWith('ai');
    });
  });

  it('renders test id with cache type', () => {
    render(
      React.createElement(CacheTypeCard, {
        cacheType: mockCacheType,
        onClear: vi.fn(),
      })
    );
    expect(screen.getByTestId('cache-type-ai')).toBeInTheDocument();
  });
});

describe('formatBytes', () => {
  it('formats 0 bytes', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  it('formats bytes', () => {
    expect(formatBytes(500)).toBe('500 B');
  });

  it('formats kilobytes', () => {
    expect(formatBytes(1024)).toBe('1.0 KB');
  });

  it('formats megabytes', () => {
    expect(formatBytes(1048576)).toBe('1.0 MB');
  });

  it('formats gigabytes', () => {
    expect(formatBytes(1073741824)).toBe('1.0 GB');
  });
});
