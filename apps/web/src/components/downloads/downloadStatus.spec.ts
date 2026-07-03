import { describe, it, expect } from 'vitest';
import { getDownloadStatus } from './downloadStatus';
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
});
