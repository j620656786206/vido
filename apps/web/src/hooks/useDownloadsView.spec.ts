import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDownloadsView } from './useDownloadsView';

describe('useDownloadsView (ux3-4-4 AC1)', () => {
  beforeEach(() => localStorage.clear());

  it('defaults to list', () => {
    const { result } = renderHook(() => useDownloadsView());
    expect(result.current[0]).toBe('list');
  });

  it('persists the choice to localStorage', () => {
    const { result } = renderHook(() => useDownloadsView());
    act(() => result.current[1]('table'));
    expect(result.current[0]).toBe('table');
    expect(localStorage.getItem('vido:downloads:view')).toBe('table');
  });

  it('reads a persisted table preference on init', () => {
    localStorage.setItem('vido:downloads:view', 'table');
    const { result } = renderHook(() => useDownloadsView());
    expect(result.current[0]).toBe('table');
  });

  it('falls back to list for an invalid stored value', () => {
    localStorage.setItem('vido:downloads:view', 'garbage');
    const { result } = renderHook(() => useDownloadsView());
    expect(result.current[0]).toBe('list');
  });
});
