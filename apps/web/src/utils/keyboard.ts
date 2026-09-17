import type { KeyboardEvent } from 'react';

/**
 * True when this key press belongs to an input method (注音, 倉頡, pinyin…)
 * rather than to the page: the Enter that picks a candidate or the Esc that
 * drops one. Handlers that submit on Enter or cancel on Esc must ignore it,
 * or they fire with half-typed text (dsr-6c AC #7).
 *
 * Chrome and Firefox flag the event with `isComposing`. Safari reports the
 * candidate-committing Enter with `isComposing: false` but `keyCode` 229.
 */
export function isImeComposing(e: KeyboardEvent): boolean {
  return e.nativeEvent.isComposing || e.keyCode === 229;
}
