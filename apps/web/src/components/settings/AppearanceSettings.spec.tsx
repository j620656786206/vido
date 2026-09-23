import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, it, expect } from 'vitest';
import { AppearanceSettings } from './AppearanceSettings';

describe('AppearanceSettings', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });
  afterEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  // dsr-3a: 外觀 was the one tab whose title was an 18px h2 — the heading
  // shrank as you stepped onto it from 連線設定.
  it('titles the page with the shared h1, not its own h2', () => {
    render(<AppearanceSettings />);
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1).toHaveTextContent('外觀');
    expect(h1.className).toContain('sm:text-2xl');
    expect(screen.queryByRole('heading', { level: 2 })).toBeNull();
  });

  it('says it is following the OS when nothing was chosen', () => {
    render(<AppearanceSettings />);
    expect(screen.getByText('目前跟隨系統設定；選了其中一個之後就不再跟隨。')).toBeInTheDocument();
  });

  it('says it is following the user once a choice is stored', () => {
    localStorage.setItem('vido:theme', 'light');
    render(<AppearanceSettings />);
    expect(screen.getByText('已依你的選擇顯示。')).toBeInTheDocument();
  });

  it('flips the description the moment a theme is picked', async () => {
    const user = userEvent.setup();
    render(<AppearanceSettings />);
    await user.click(screen.getByTestId('theme-option-light'));
    expect(screen.getByText('已依你的選擇顯示。')).toBeInTheDocument();
    expect(localStorage.getItem('vido:theme')).toBe('light');
    expect(screen.getByTestId('theme-option-light')).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByTestId('theme-option-dark')).toHaveAttribute('aria-checked', 'false');
  });

  it('names the radiogroup after the page heading', () => {
    render(<AppearanceSettings />);
    const group = screen.getByRole('radiogroup');
    expect(group).toHaveAccessibleName('外觀');
  });

  // C6-D qG91e: the two cards are 768px together — the settings form measure —
  // instead of stretching to the 1152px column.
  it('holds the two cards to the 768px form measure', () => {
    render(<AppearanceSettings />);
    expect(screen.getByRole('radiogroup').className).toContain('max-w-3xl');
  });
});
