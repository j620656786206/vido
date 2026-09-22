import { describe, it, expect, vi } from 'vitest';
import { useRef, useState } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { Sheet } from './Sheet';

/** Whole class TOKENS, never substrings: `pb-[max(1rem,…)]` contains `p`-ish
 *  fragments, and a substring check is how an assertion ends up vacuous. */
const tokens = (el: Element) => (el.getAttribute('class') ?? '').split(/\s+/).filter(Boolean);

/** Controlled wrapper so Escape → onOpenChange(false) actually closes it. */
function Harness(props: Omit<React.ComponentProps<typeof Sheet>, 'open' | 'onOpenChange'>) {
  const [open, setOpen] = useState(true);
  return (
    <Sheet open={open} onOpenChange={setOpen} {...props}>
      <button type="button">inside</button>
    </Sheet>
  );
}

describe('ui/Sheet — the props dsr-4b-1 adds', () => {
  it('testId replaces the testid', () => {
    render(<Harness testId="download-sort-sheet" />);
    expect(screen.getByTestId('download-sort-sheet')).toBeInTheDocument();
    expect(screen.queryByTestId('bottom-sheet')).not.toBeInTheDocument();
  });

  it('className merges through cn(): p-0 really drops p-4 (and the safe-area pb with it)', () => {
    render(<Harness className="p-0 pt-2 pb-[max(1.25rem,env(safe-area-inset-bottom))]" />);
    const t = tokens(screen.getByTestId('bottom-sheet'));
    expect(t).not.toContain('p-4');
    expect(t).not.toContain('pb-[max(1rem,env(safe-area-inset-bottom))]');
    expect(t).toContain('p-0');
    expect(t).toContain('pt-2');
    expect(t).toContain('pb-[max(1.25rem,env(safe-area-inset-bottom))]');
    // Untouched base classes survive the merge.
    expect(t).toContain('fixed');
    expect(t).toContain('z-[71]');
  });

  it('titleClassName merges into the visible title', () => {
    render(<Harness title="排序" titleClassName="mb-1 px-5" />);
    const title = screen.getByText('排序');
    const t = tokens(title);
    expect(t).toContain('mb-1');
    expect(t).toContain('px-5');
    expect(t).not.toContain('mb-3');
    expect(t).toContain('font-semibold');
  });

  it('finalFocus (ref) sends focus to that element after the sheet closes', async () => {
    function WithTarget() {
      const target = useRef<HTMLButtonElement>(null);
      const [open, setOpen] = useState(true);
      return (
        <>
          <button type="button" ref={target}>
            target
          </button>
          <Sheet open={open} onOpenChange={setOpen} finalFocus={target} ariaLabel="測試">
            <button type="button">inside</button>
          </Sheet>
        </>
      );
    }
    render(<WithTarget />);
    await waitFor(() => expect(screen.getByText('inside')).toHaveFocus());
    fireEvent.keyDown(screen.getByText('inside'), { key: 'Escape' });
    await waitFor(() => expect(screen.queryByText('inside')).not.toBeInTheDocument());
    await waitFor(() => expect(screen.getByText('target')).toHaveFocus());
  });

  it('description renders as the dialog description (aria-describedby), with its class merged', () => {
    render(
      <Harness title="排序" description="選擇下載清單的排序方式" descriptionClassName="px-5 pb-3" />
    );
    expect(screen.getByRole('dialog')).toHaveAccessibleDescription('選擇下載清單的排序方式');
    const t = tokens(screen.getByText('選擇下載清單的排序方式'));
    expect(t).toEqual(expect.arrayContaining(['px-5', 'pb-3', 'text-sm']));
  });

  it('onOpenChangeComplete is forwarded to the root', async () => {
    const onOpenChangeComplete = vi.fn();
    render(<Harness onOpenChangeComplete={onOpenChangeComplete} />);
    await waitFor(() => expect(screen.getByText('inside')).toHaveFocus());
    fireEvent.keyDown(screen.getByText('inside'), { key: 'Escape' });
    await waitFor(() => expect(onOpenChangeComplete).toHaveBeenCalledWith(false));
  });
});

describe('ui/Sheet — defaults keep the two existing callers unchanged', () => {
  it('without new props the popup and title classes are byte-for-byte the pre-4b-1 strings', () => {
    // Pinned verbatim from ui/Sheet.tsx before dsr-4b-1: MobileMoreSheet and
    // LibraryFilterSheetV2 pass none of the new props, so cn() must hand these back
    // untouched — a tailwind-merge upgrade that reclassifies a token fails here.
    render(<Harness title="更多" />);
    expect(screen.getByTestId('bottom-sheet').getAttribute('class')).toBe(
      'fixed inset-x-0 bottom-0 z-[71] max-h-[85vh] overflow-y-auto rounded-t-[var(--radius-xl)] border-t border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-[var(--shadow-xl)] transition-transform duration-[var(--motion-move)] data-[ending-style]:translate-y-full data-[starting-style]:translate-y-full'
    );
    expect(screen.getByText('更多').getAttribute('class')).toBe(
      'mb-3 text-base font-semibold text-[var(--text-primary)]'
    );
    // …and no description element appears unless one is asked for.
    expect(screen.getByRole('dialog')).not.toHaveAttribute('aria-describedby');
  });

  it('without a title there is an sr-only 選單 heading', () => {
    render(<Harness />);
    const heading = screen.getByText('選單');
    expect(tokens(heading)).toContain('sr-only');
    expect(screen.getByRole('dialog')).toHaveAccessibleName('選單');
  });
});
