import { describe, it, expect } from 'vitest';
import { getDownloadStatus, getDownloadTone } from './downloadStatus';
import type { TorrentStatus } from '../../services/downloadService';

describe('getDownloadStatus (DL-v2 §2.5 status→token)', () => {
  it('maps every torrent status to a zh-TW label + a token tint class', () => {
    const labels: Record<TorrentStatus, string> = {
      downloading: '下載中',
      paused: '已暫停',
      seeding: '做種',
      completed: '已完成',
      stalled: '停滯',
      error: '錯誤',
      queued: '佇列中',
      checking: '檢查中',
    };
    (Object.keys(labels) as TorrentStatus[]).forEach((status) => {
      const d = getDownloadStatus(status);
      expect(d.label).toBe(labels[status]);
      // token-only: a tint background var, never a raw hex (AC7)
      expect(d.className).toMatch(/bg-\[var\(--[a-z-]+\)\]/);
      expect(d.className).not.toMatch(/#[0-9a-fA-F]{3,6}/);
    });
  });

  it('uses AA-safe -text variants for the accent (downloading) and error pills (TC-2)', () => {
    expect(getDownloadStatus('downloading').className).toContain('text-[var(--accent-text)]');
    expect(getDownloadStatus('error').className).toContain('text-[var(--error-text)]');
  });

  it('a paused torrent is neutral, not 赭 — the user asked for the pause (dsr-4)', () => {
    const paused = getDownloadStatus('paused').className;
    expect(paused).toContain('bg-[var(--bg-tertiary)]');
    expect(paused).toContain('text-[var(--text-secondary)]');
    expect(paused).not.toContain('warning');
  });
});

describe('getDownloadTone (dsr-4 progress colour)', () => {
  it('gold only while bytes are moving down', () => {
    expect(getDownloadTone({ status: 'downloading', progress: 0.4 }).fill).toBe(
      'bg-[var(--accent-primary)]'
    );
    expect(getDownloadTone({ status: 'paused', progress: 0.4 }).fill).toBe(
      'bg-[var(--text-muted)]'
    );
    expect(getDownloadTone({ status: 'stalled', progress: 0.4 }).fill).toBe(
      'bg-[var(--text-muted)]'
    );
    expect(getDownloadTone({ status: 'checking', progress: 0.4 }).fill).toBe('bg-[var(--info)]');
    expect(getDownloadTone({ status: 'queued', progress: 0.4 }).fill).toBe(
      'bg-[var(--text-muted)]'
    );
  });

  it('seeding is 靛青, a finished torrent is 青碧, an errored one is 硃砂', () => {
    expect(getDownloadTone({ status: 'seeding', progress: 1 }).text).toBe(
      'text-[var(--info-text)]'
    );
    expect(getDownloadTone({ status: 'completed', progress: 1 }).text).toBe(
      'text-[var(--success-text)]'
    );
    expect(getDownloadTone({ status: 'paused', progress: 1 }).text).toBe(
      'text-[var(--success-text)]'
    );
    expect(getDownloadTone({ status: 'error', progress: 0.3 }).text).toBe(
      'text-[var(--error-text)]'
    );
  });
});
