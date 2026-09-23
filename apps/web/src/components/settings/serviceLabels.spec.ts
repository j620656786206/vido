import { describe, it, expect } from 'vitest';
import { SERVICE_LABELS, getServiceLabel, isBroken } from './serviceLabels';

describe('serviceLabels', () => {
  it.each([
    ['tmdb', 'TMDb API', 'TMDB', '中繼資料與海報'],
    ['douban', 'Douban Scraper', '豆瓣', '評分（選配）'],
    ['wikipedia', 'Wikipedia API', '維基百科', '中繼資料補充（選配）'],
    ['ai', 'AI Parser', 'AI 解析', '檔名解析與字幕翻譯'],
    ['qbittorrent', 'qBittorrent', 'qBittorrent', '下載器'],
  ])('%s → the Chinese name, never the backend displayName', (name, displayName, zh, role) => {
    expect(getServiceLabel({ name, displayName })).toMatchObject({ name: zh, role });
  });

  // Keyed by name, not displayName — a renamed display string must not un-translate the page.
  it('looks up by service.name even when displayName changes', () => {
    expect(getServiceLabel({ name: 'douban', displayName: 'Douban v2' }).name).toBe('豆瓣');
  });

  it('falls back to displayName with no role and no hint for a service it does not know', () => {
    const label = getServiceLabel({ name: 'sonarr', displayName: 'Sonarr' });
    expect(label).toEqual({ name: 'Sonarr', role: '' });
    expect(label.fixHint).toBeUndefined();
  });

  it('only qBittorrent and TMDB carry advice, each with a link to where it is fixed', () => {
    const withHints = Object.entries(SERVICE_LABELS)
      .filter(([, l]) => l.fixHint)
      .map(([k, l]) => [k, l.fixHint?.link?.to]);
    expect(withHints).toEqual([
      ['tmdb', '/settings/keys'],
      ['qbittorrent', '/settings/connection'],
    ]);
    // The link label must literally appear in the sentence it is cut out of.
    for (const l of Object.values(SERVICE_LABELS)) {
      if (l.fixHint?.link) expect(l.fixHint.text).toContain(`「${l.fixHint.link.label}」`);
    }
  });

  it('only error and disconnected count as broken', () => {
    expect(
      ['connected', 'rate_limited', 'error', 'disconnected', 'unconfigured'].filter((s) =>
        isBroken(s as never)
      )
    ).toEqual(['error', 'disconnected']);
  });
});
