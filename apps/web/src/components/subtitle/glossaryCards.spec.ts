import { describe, it, expect } from 'vitest';
import { groupGlossaryCards } from './glossaryCards';
import type { GlossaryTerm } from '../../services/glossaryService';

const t = (
  id: string,
  termSrc: string,
  termZh: string,
  extra: Partial<GlossaryTerm> = {}
): GlossaryTerm => ({
  id,
  mediaId: 'see',
  termSrc,
  termZh,
  language: 'zh-Hant',
  source: 'official_subtitle',
  confirmed: false,
  createdAt: '',
  updatedAt: '',
  ...extra,
});

describe('groupGlossaryCards (disc-2026-10-glossary-season-variants-card-b)', () => {
  // The prod list for 《末日光明》, 2026-10-18: Baba Voss had three rows.
  const list = [
    t('bv', 'Baba Voss', '巴霸沃斯'),
    t('bv1', 'Baba Voss', '巴巴禾斯', { season: 1 }),
    t('bv2', 'baba voss', '巴霸沃斯', { season: 2 }),
    t('h', 'Haniwa', '哈妮娃', { confirmed: true }),
    t('o2', 'Oloman', '奧路文', { season: 2 }),
    t('o1', 'Oloman', '歐羅曼', { season: 1 }),
  ];

  it('makes one card per source term, NOCASE, in list order', () => {
    const cards = groupGlossaryCards(list);
    expect(cards.map((c) => c.key)).toEqual(['baba voss', 'haniwa', 'oloman']);
    expect(cards[0].main?.id).toBe('bv');
    expect(cards[0].seasons.map((s) => s.id)).toEqual(['bv1', 'bv2']);
  });

  it('shows only the seasons spelled differently from the show-wide row', () => {
    const [baba, haniwa] = groupGlossaryCards(list);
    expect(baba.shownSeasons.map((s) => s.id)).toEqual(['bv1']);
    expect(haniwa.shownSeasons).toEqual([]);
  });

  it('an orphan keeps every season, in season order', () => {
    const oloman = groupGlossaryCards(list)[2];
    expect(oloman.main).toBeUndefined();
    expect(oloman.termSrc).toBe('Oloman');
    expect(oloman.shownSeasons.map((s) => s.id)).toEqual(['o1', 'o2']);
  });

  it('a card is unconfirmed when its row or a shown season line is; a hidden one does not count', () => {
    const cards = groupGlossaryCards([
      t('a', 'A', '甲', { confirmed: true }),
      t('a1', 'A', '乙', { season: 1 }),
      t('b', 'B', '丙', { confirmed: true }),
      t('b1', 'B', '丙', { season: 1 }),
      t('c', 'C', '丁', { confirmed: true }),
    ]);
    expect(cards.map((c) => c.unconfirmed)).toEqual([true, false, false]);
  });
});
