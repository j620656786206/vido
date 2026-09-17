import { describe, it, expect } from 'vitest';
import type { KeyboardEvent as ReactKeyboardEvent } from 'react';
import { isImeComposing } from './keyboard';

// dsr-6c AC #7: an Enter or Esc that only picks (or drops) an input-method
// candidate must not submit or cancel. Browsers disagree on how they say so.
type KeyInit = NonNullable<ConstructorParameters<typeof KeyboardEvent>[1]> & { keyCode?: number };

function keyEvent(init: KeyInit): ReactKeyboardEvent {
  const native = new KeyboardEvent('keydown', init);
  return {
    key: native.key,
    keyCode: init.keyCode ?? 0,
    nativeEvent: native,
  } as unknown as ReactKeyboardEvent;
}

describe('isImeComposing', () => {
  it('is true while the native event says it is composing (Chrome, Firefox)', () => {
    expect(isImeComposing(keyEvent({ key: 'Enter', isComposing: true }))).toBe(true);
  });

  it('is true for keyCode 229 even when isComposing is false (Safari commits a candidate this way)', () => {
    expect(isImeComposing(keyEvent({ key: 'Enter', isComposing: false, keyCode: 229 }))).toBe(true);
  });

  it('is false for a plain Enter', () => {
    expect(isImeComposing(keyEvent({ key: 'Enter', keyCode: 13 }))).toBe(false);
  });

  it('is false for a plain Escape', () => {
    expect(isImeComposing(keyEvent({ key: 'Escape', keyCode: 27 }))).toBe(false);
  });
});
