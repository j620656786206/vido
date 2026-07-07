import { describe, it, expect } from 'vitest';
import { deriveWorkspaceMode, modeShowsFeed } from './generationWorkspace';

const base = { probing: false, batchStatus: 'idle' as const, hasItems: false, singleJobCount: 0 };

describe('deriveWorkspaceMode (ux3-ai-2 AC 2/3 state matrix)', () => {
  it('probe in flight → loading', () => {
    expect(deriveWorkspaceMode({ ...base, probing: true })).toBe('loading');
  });

  it('idle batch + no single jobs → idle', () => {
    expect(deriveWorkspaceMode(base)).toBe('idle');
  });

  it('idle batch + single jobs in flight → single', () => {
    expect(deriveWorkspaceMode({ ...base, singleJobCount: 2 })).toBe('single');
  });

  it('running batch WITH items → running (full queue)', () => {
    expect(deriveWorkspaceMode({ ...base, batchStatus: 'running', hasItems: true })).toBe(
      'running'
    );
  });

  it('running batch WITHOUT items → attach (degraded, no fake queue)', () => {
    expect(deriveWorkspaceMode({ ...base, batchStatus: 'running', hasItems: false })).toBe(
      'attach'
    );
  });

  it.each(['budget_ceiling', 'complete', 'cancelled', 'error'] as const)(
    'terminal batch status %s maps straight through',
    (status) => {
      expect(deriveWorkspaceMode({ ...base, batchStatus: status })).toBe(status);
    }
  );

  it('a running batch takes precedence over stray single jobs (batch owns the queue)', () => {
    expect(
      deriveWorkspaceMode({ ...base, batchStatus: 'running', hasItems: true, singleJobCount: 3 })
    ).toBe('running');
  });
});

describe('modeShowsFeed', () => {
  it('hides the feed for loading + idle, shows it once anything is/was happening', () => {
    expect(modeShowsFeed('loading')).toBe(false);
    expect(modeShowsFeed('idle')).toBe(false);
    for (const m of [
      'running',
      'attach',
      'single',
      'budget_ceiling',
      'complete',
      'cancelled',
      'error',
    ] as const) {
      expect(modeShowsFeed(m)).toBe(true);
    }
  });
});
