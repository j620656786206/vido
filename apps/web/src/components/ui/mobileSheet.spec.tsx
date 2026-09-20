import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { MOBILE_SHEET_CONTENT, MOBILE_SHEET_CLOSE, SheetGrabber } from './mobileSheet';

/** Whole class TOKENS, never substrings: `aria-disabled:opacity-50` contains
 *  `disabled:opacity-50`, and that kind of accident is how an assertion ends up
 *  vacuous or order-dependent (dsr-6e-2 CR M1). */
const tokens = (s: string) => s.split(/\s+/).filter(Boolean);

describe('ui/mobileSheet — the phone bottom-sheet shell', () => {
  it('slides up ONLY below sm: — the sheet animation never leaks to the centred desktop dialog', () => {
    const t = tokens(MOBILE_SHEET_CONTENT);
    expect(t).toContain('max-sm:data-[state=open]:animate-sheet-enter');
    expect(t).toContain('max-sm:data-[state=closed]:animate-sheet-exit');
    // Desktop centres with `translate`; an unprefixed sheet animation animates
    // that same property and would throw the dialog off-centre.
    expect(t.filter((c) => c.includes('animate-sheet') && !c.startsWith('max-sm:'))).toEqual([]);
  });

  it('keeps the shell the four dialogs already shared, verbatim', () => {
    const t = tokens(MOBILE_SHEET_CONTENT);
    for (const c of [
      'bottom-0',
      'left-0',
      'right-0',
      'top-auto',
      'w-full',
      'max-w-none',
      'translate-x-0',
      'translate-y-0',
      'rounded-b-none',
      'rounded-t-[var(--radius-xl)]',
    ])
      expect(t).toContain(c);
  });

  it('the 44×44 close carries NO `top` — header heights differ per dialog', () => {
    const t = tokens(MOBILE_SHEET_CLOSE);
    expect(t).toContain('max-sm:h-11');
    expect(t).toContain('max-sm:w-11');
    expect(t).toContain('max-sm:right-1');
    expect(t.every((c) => c.startsWith('max-sm:'))).toBe(true);
    expect(t.filter((c) => /(^|:)top-/.test(c))).toEqual([]);
  });

  it('the grabber is phone-only, decorative, and forwards props to its wrapper', () => {
    const { getByTestId } = render(<SheetGrabber data-testid="grab" className="extra" />);
    const wrap = getByTestId('grab');
    expect(tokens(wrap.className)).toContain('sm:hidden');
    expect(tokens(wrap.className)).toContain('extra');
    const bar = wrap.firstElementChild as HTMLElement;
    expect(bar).toHaveAttribute('aria-hidden', 'true');
    expect(tokens(bar.className)).toEqual(
      expect.arrayContaining(['h-1', 'w-9', 'rounded-full', 'bg-[var(--bg-tertiary)]'])
    );
  });
});
