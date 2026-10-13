import { act, fireEvent, screen } from '@testing-library/react';
import { expect } from 'vitest';

/**
 * An icon-only control names itself in the shared tooltip on keyboard focus
 * (J11-D rule 6, DESIGN.md §Tooltips) and never in a browser `title`.
 * Focuses `el`, waits for `text` to appear outside it, then blurs it again.
 */
export async function expectTooltipOnFocus(el: HTMLElement, text: string): Promise<void> {
  expect(el.hasAttribute('title')).toBe(false);
  await act(async () => {
    fireEvent.focus(el);
  });
  const shown = await screen.findAllByText(text);
  expect(shown.some((node) => !el.contains(node))).toBe(true);
  await act(async () => {
    fireEvent.blur(el);
  });
}
