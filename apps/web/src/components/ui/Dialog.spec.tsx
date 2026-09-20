import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
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
    close: screen.getByText('Close').closest('button') as HTMLElement,
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
