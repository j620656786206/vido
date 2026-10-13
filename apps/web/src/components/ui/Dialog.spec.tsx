import { describe, it, expect } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { Dialog, DialogContent, DialogTitle } from './Dialog';

const CLOSE_TODAY =
  'absolute right-4 top-4 rounded-[var(--radius-sm)] text-[var(--text-muted)] opacity-70 transition-opacity hover:opacity-100 focus:outline-none';

function open(props: Partial<React.ComponentProps<typeof DialogContent>> = {}) {
  render(
    <Dialog open>
      <DialogContent aria-describedby={undefined} data-testid="content" {...props}>
        <DialogTitle>t</DialogTitle>
      </DialogContent>
    </Dialog>
  );
  return {
    content: screen.getByTestId('content'),
    close: screen.getByRole('button', { name: '關閉視窗' }),
  };
}

describe('ui/Dialog — closeClassName (dsr-6f-1)', () => {
  it('[guard] without it, the ✕ is byte-identical to what every dialog ships today', () => {
    expect(open().close.className).toBe(CLOSE_TODAY);
  });

  it('merges extra classes onto the ✕ only', () => {
    const { close, content } = open({ closeClassName: 'max-sm:h-11 max-sm:top-4' });
    const t = close.className.split(/\s+/);
    expect(t).toEqual(expect.arrayContaining(['max-sm:h-11', 'max-sm:top-4', 'right-4', 'top-4']));
    expect(content.className).not.toContain('max-sm:h-11');
  });

  it('never leaks the prop onto the DOM (DialogContent spreads ...props)', () => {
    const { content } = open({ closeClassName: 'max-sm:h-11' });
    expect(content.getAttribute('closeclassname')).toBeNull();
    expect(content.getAttribute('closeClassName')).toBeNull();
  });
});

// disc-2026-10-icon-only-native-title-audit — the ✕ is icon-only, so its name
// shows in the shared tooltip (J11-D rule 6), never only in a browser title.
describe('ui/Dialog — the ✕ names itself', () => {
  it('is called 關閉視窗, in Chinese, with no title attribute', () => {
    const { close } = open();
    expect(screen.getByRole('button', { name: '關閉視窗' })).toBe(close);
    expect(close.hasAttribute('title')).toBe(false);
  });

  it('keyboard focus shows the same sentence in the tooltip', async () => {
    const { close } = open();
    await act(async () => {
      fireEvent.focus(close);
    });
    const tips = await screen.findAllByText('關閉視窗');
    // the sr-only name plus the tooltip popup
    expect(tips.length).toBeGreaterThan(1);
  });
});
