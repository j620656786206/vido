// Implements: <utility — no .pen counterpart>
/**
 * Groups the glossary list into cards (disc-2026-10-glossary-season-variants-card-b,
 * F6s-D-v2 mHJar): one card per source term (case-insensitive, like the
 * NOCASE unique index), holding the show-wide row and its per-season drawers.
 *
 * ⚖️ 2026-10-18: every season follows its own official spelling. A season
 * line shows only where it DIFFERS from the show-wide spelling — a season
 * that spells it the same says nothing new (F6s Note ②).
 */
import type { GlossaryTerm } from '../../services/glossaryService';

export interface GlossaryCard {
  /** Lower-cased source term — stable React key. */
  key: string;
  termSrc: string;
  /** The show-wide row; absent for an orphan (only season drawers left). */
  main?: GlossaryTerm;
  /** Every season drawer of this term, by season. */
  seasons: GlossaryTerm[];
  /** The season lines the card shows: those spelled differently from `main`. */
  shownSeasons: GlossaryTerm[];
  /** 未確認 on the card: the show-wide row or a shown season line is unconfirmed. */
  unconfirmed: boolean;
}

export function groupGlossaryCards(list: GlossaryTerm[]): GlossaryCard[] {
  const byKey = new Map<string, GlossaryCard>();
  for (const term of list) {
    const key = term.termSrc.toLowerCase();
    let card = byKey.get(key);
    if (!card) {
      card = { key, termSrc: term.termSrc, seasons: [], shownSeasons: [], unconfirmed: false };
      byKey.set(key, card);
    }
    if (term.season === undefined) {
      card.main = term;
      card.termSrc = term.termSrc;
    } else {
      card.seasons.push(term);
    }
  }
  const cards = [...byKey.values()];
  for (const card of cards) {
    card.seasons.sort((a, b) => (a.season ?? 0) - (b.season ?? 0));
    card.shownSeasons = card.main
      ? card.seasons.filter((s) => s.termZh !== card.main?.termZh)
      : card.seasons;
    card.unconfirmed =
      (card.main !== undefined && !card.main.confirmed) ||
      card.shownSeasons.some((s) => !s.confirmed);
  }
  return cards;
}
