import { act, fireEvent, render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { Tooltip } from './Tooltip';

// disc-2026-10-episode-list-subtitle-badge-b — `openOnPress` (J11-D rule 4).
// The real-browser behaviour is pinned by tests/e2e/episode-subtitle-badge;
// this guards the press logic against a Base UI bump (CR L5).

function renderTip() {
  render(
    <Tooltip content="有繁中字幕" delay={0} openOnPress>
      <button type="button">mark</button>
    </Tooltip>
  );
  return screen.getByRole('button', { name: 'mark' });
}

describe('Tooltip openOnPress', () => {
  it('a touch press opens it and the hover-close a tap ends with does not shut it', async () => {
    const button = renderTip();
    await act(async () => {
      fireEvent.pointerDown(button, { pointerType: 'touch' });
      fireEvent.click(button, { detail: 1 });
    });
    expect(await screen.findByText('有繁中字幕')).toBeInTheDocument();

    await act(async () => {
      fireEvent.pointerLeave(button, { pointerType: 'touch' });
      fireEvent.mouseLeave(button);
    });
    expect(screen.getByText('有繁中字幕')).toBeInTheDocument();
  });

  it('Esc closes a press-opened tooltip', async () => {
    const button = renderTip();
    await act(async () => {
      fireEvent.pointerDown(button, { pointerType: 'touch' });
      fireEvent.click(button, { detail: 1 });
    });
    expect(await screen.findByText('有繁中字幕')).toBeInTheDocument();
    await act(async () => {
      fireEvent.keyDown(document.activeElement ?? document.body, { key: 'Escape' });
    });
    await act(async () => {
      await new Promise((r) => setTimeout(r, 50));
    });
    expect(screen.queryByText('有繁中字幕')).not.toBeInTheDocument();
  });

  it('without openOnPress a click does not open it (existing callers unchanged)', async () => {
    render(
      <Tooltip content="側欄標籤">
        <button type="button">rail</button>
      </Tooltip>
    );
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'rail' }), { detail: 1 });
    });
    expect(screen.queryByText('側欄標籤')).not.toBeInTheDocument();
  });
});
