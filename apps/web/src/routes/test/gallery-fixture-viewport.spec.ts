import { describe, it, expect } from 'vitest';
import { GALLERY_FIXTURES } from './-gallery.fixtures';

/**
 * dsr-6f-1 — per-fixture viewports.
 *
 * `width` boxes the component on a 1280px page; `viewport` resizes the page
 * itself. Setting both describes two different screens at once, and the visual
 * spec would silently photograph the wrong one.
 */
describe('gallery fixtures — viewport', () => {
  it('a fixture is either boxed (`width`) or re-viewported (`viewport`), never both', () => {
    const both = GALLERY_FIXTURES.filter((fx) => fx.width !== undefined && fx.viewport).map(
      (fx) => fx.id
    );
    expect(both).toEqual([]);
  });

  it('phone fixtures use the .pen canvas size, 390×844', () => {
    const phones = GALLERY_FIXTURES.filter((fx) => fx.viewport);
    expect(phones.length).toBeGreaterThan(0);
    for (const fx of phones) {
      expect(fx.viewport, fx.id).toEqual({ width: 390, height: 844 });
    }
  });
});
