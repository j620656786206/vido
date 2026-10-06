import { useRef, useState } from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest';
import { useFilterRailTransition, RAIL_MOTION_ATTR } from './useFilterRailTransition';

/**
 * disc-2026-10-filter-rail-toggle-no-motion (I11-D, `Qaz1x`).
 *
 * jsdom has no `document.startViewTransition`, so every branch that matters is
 * driven by a stand-in installed per test. The stand-in calls the update
 * callback synchronously (a real browser calls it a frame later, after it has
 * captured the old picture) and hands back a `finished` promise the test settles
 * by hand — that is the only way to look at `data-rail-motion` WHILE the
 * transition is "running".
 */

type Deferred = { promise: Promise<void>; resolve: () => void };
function deferred(): Deferred {
  let resolve!: () => void;
  const promise = new Promise<void>((r) => (resolve = r));
  return { promise, resolve };
}

interface Seen {
  collapsedInCallback: boolean | null;
  focusedInCallback: string | null;
  motionInCallback: string | null;
}

/** Installs a startViewTransition stand-in; each call gets its own `finished`. */
function installViewTransitions() {
  const finishes: Deferred[] = [];
  const seen: Seen[] = [];
  const spy = vi.fn((cb?: () => void) => {
    cb?.();
    seen.push({
      collapsedInCallback: screen.queryByTestId('rail') === null,
      focusedInCallback: document.activeElement?.getAttribute('data-testid') ?? null,
      motionInCallback: document.documentElement.getAttribute(RAIL_MOTION_ATTR),
    });
    const finished = deferred();
    finishes.push(finished);
    return {
      finished: finished.promise,
      ready: Promise.resolve(),
      updateCallbackDone: Promise.resolve(),
      skipTransition: vi.fn(),
    };
  });
  Object.defineProperty(document, 'startViewTransition', {
    value: spy,
    configurable: true,
    writable: true,
  });
  return { spy, finishes, seen };
}

function stubReducedMotion(matches: boolean) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn((q: string) => ({
      matches,
      media: q,
      addEventListener() {},
      removeEventListener() {},
    })) as unknown as typeof window.matchMedia
  );
}

const STORAGE_KEY = 'test:rail-collapsed';

/**
 * The same shape both pages use: state seeded from localStorage on mount, the
 * rail mounted/unmounted on that state, a toolbar 篩選 button only while
 * collapsed — and the hook doing the toggling.
 */
function Harness() {
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(STORAGE_KEY) === '1');
  const expandRef = useRef<HTMLButtonElement>(null);
  const collapseRef = useRef<HTMLButtonElement>(null);
  const { collapse, expand } = useFilterRailTransition({
    setRailCollapsed: setCollapsed,
    getCollapsedFocusTarget: () => expandRef.current,
    getExpandedFocusTarget: () => collapseRef.current,
  });
  return (
    <div>
      {!collapsed && (
        <aside data-testid="rail">
          <button ref={collapseRef} data-testid="collapse" onClick={collapse}>
            收合篩選
          </button>
        </aside>
      )}
      {collapsed && (
        <button ref={expandRef} data-testid="expand" onClick={expand}>
          篩選
        </button>
      )}
    </div>
  );
}

beforeEach(() => {
  localStorage.clear();
  stubReducedMotion(false);
});

afterEach(() => {
  delete (document as { startViewTransition?: unknown }).startViewTransition;
  document.documentElement.removeAttribute(RAIL_MOTION_ATTR);
  vi.unstubAllGlobals();
});

describe('useFilterRailTransition', () => {
  it('AC #1: collapse runs inside ONE view transition — state flushed and focus moved inside the callback', async () => {
    const vt = installViewTransitions();
    render(<Harness />);
    await userEvent.click(screen.getByTestId('collapse'));

    expect(vt.spy).toHaveBeenCalledTimes(1);
    // flushSync: the browser photographs the NEW picture when the callback returns,
    // so the rail must already be gone and focus already moved by then.
    expect(vt.seen[0]).toEqual({
      collapsedInCallback: true,
      focusedInCallback: 'expand',
      motionInCallback: 'collapse',
    });
  });

  it('AC #2: expand runs inside a view transition tagged "expand" and lands on the collapse button', async () => {
    localStorage.setItem(STORAGE_KEY, '1');
    const vt = installViewTransitions();
    render(<Harness />);
    await userEvent.click(screen.getByTestId('expand'));

    expect(vt.spy).toHaveBeenCalledTimes(1);
    expect(vt.seen[0]).toEqual({
      collapsedInCallback: false,
      focusedInCallback: 'collapse',
      motionInCallback: 'expand',
    });
  });

  it('data-rail-motion lives exactly as long as the transition, then comes off', async () => {
    const vt = installViewTransitions();
    render(<Harness />);
    await userEvent.click(screen.getByTestId('collapse'));

    expect(document.documentElement.getAttribute(RAIL_MOTION_ATTR)).toBe('collapse');
    vt.finishes[0].resolve();
    await vt.finishes[0].promise;
    await Promise.resolve();
    await Promise.resolve();
    expect(document.documentElement.hasAttribute(RAIL_MOTION_ATTR)).toBe(false);
  });

  it('a second click mid-transition: the FIRST finishing does not strip the attribute from the second', async () => {
    const vt = installViewTransitions();
    render(<Harness />);
    await userEvent.click(screen.getByTestId('collapse'));
    await userEvent.click(screen.getByTestId('expand'));
    expect(vt.spy).toHaveBeenCalledTimes(2);
    expect(document.documentElement.getAttribute(RAIL_MOTION_ATTR)).toBe('expand');

    // The browser skips the first one; its `finished` settles while the second runs.
    vt.finishes[0].resolve();
    await vt.finishes[0].promise;
    await Promise.resolve();
    await Promise.resolve();
    expect(document.documentElement.getAttribute(RAIL_MOTION_ATTR)).toBe('expand');

    vt.finishes[1].resolve();
    await vt.finishes[1].promise;
    await Promise.resolve();
    await Promise.resolve();
    expect(document.documentElement.hasAttribute(RAIL_MOTION_ATTR)).toBe(false);
  });

  it('a transition whose update callback failed still takes the attribute off', async () => {
    const spy = vi.fn((cb?: () => void) => {
      cb?.();
      return {
        finished: Promise.reject(new Error('update callback failed')),
        ready: Promise.resolve(),
        updateCallbackDone: Promise.resolve(),
        skipTransition: vi.fn(),
      };
    });
    Object.defineProperty(document, 'startViewTransition', {
      value: spy,
      configurable: true,
      writable: true,
    });
    render(<Harness />);
    await userEvent.click(screen.getByTestId('collapse'));
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
    expect(document.documentElement.hasAttribute(RAIL_MOTION_ATTR)).toBe(false);
  });

  it('AC #4: reduced motion → never calls startViewTransition; state and focus still switch synchronously', async () => {
    stubReducedMotion(true);
    const vt = installViewTransitions();
    render(<Harness />);
    await userEvent.click(screen.getByTestId('collapse'));

    expect(vt.spy).not.toHaveBeenCalled();
    expect(screen.queryByTestId('rail')).toBeNull();
    expect(document.activeElement).toBe(screen.getByTestId('expand'));
    expect(document.documentElement.hasAttribute(RAIL_MOTION_ATTR)).toBe(false);

    await userEvent.click(screen.getByTestId('expand'));
    expect(vt.spy).not.toHaveBeenCalled();
    expect(document.activeElement).toBe(screen.getByTestId('collapse'));
  });

  it('AC #5: no View Transitions API → no throw; state and focus switch synchronously', async () => {
    expect(typeof (document as { startViewTransition?: unknown }).startViewTransition).not.toBe(
      'function'
    );
    render(<Harness />);
    await userEvent.click(screen.getByTestId('collapse'));
    expect(screen.queryByTestId('rail')).toBeNull();
    expect(document.activeElement).toBe(screen.getByTestId('expand'));
    expect(document.documentElement.hasAttribute(RAIL_MOTION_ATTR)).toBe(false);

    await userEvent.click(screen.getByTestId('expand'));
    expect(screen.getByTestId('rail')).toBeInTheDocument();
    expect(document.activeElement).toBe(screen.getByTestId('collapse'));
  });

  it('AC #5: mounting with a remembered collapsed state never starts a transition', () => {
    localStorage.setItem(STORAGE_KEY, '1');
    const vt = installViewTransitions();
    const { unmount } = render(<Harness />);
    expect(screen.getByTestId('expand')).toBeInTheDocument();
    unmount();
    render(<Harness />);
    expect(vt.spy).not.toHaveBeenCalled();
    expect(document.documentElement.hasAttribute(RAIL_MOTION_ATTR)).toBe(false);
  });

  it('a missing focus target is not an error', async () => {
    function NoTarget() {
      const [collapsed, setCollapsed] = useState(false);
      const { collapse } = useFilterRailTransition({
        setRailCollapsed: setCollapsed,
        getCollapsedFocusTarget: () => null,
        getExpandedFocusTarget: () => null,
      });
      return collapsed ? (
        <p data-testid="gone">gone</p>
      ) : (
        <button data-testid="collapse" onClick={collapse} />
      );
    }
    render(<NoTarget />);
    await userEvent.click(screen.getByTestId('collapse'));
    expect(screen.getByTestId('gone')).toBeInTheDocument();
  });
});
