import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { SettingsPageHeader } from './SettingsPageHeader';

describe('SettingsPageHeader', () => {
  it('renders the page title as the one h1', () => {
    render(<SettingsPageHeader title="連線設定" description="說明" />);
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1).toHaveTextContent('連線設定');
  });

  // DESIGN.md §Responsive: the title steps down one rung on a phone (Title
  // 24→20); body copy does not. jsdom has no breakpoints, so this pins the
  // tokens; tests/e2e/settings-shell.spec.ts measures the computed sizes.
  it('steps the title down on a phone: 20px below sm, 24px from sm up', () => {
    render(<SettingsPageHeader title="連線設定" description="說明" />);
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1.className).toContain('text-xl');
    expect(h1.className).toContain('sm:text-2xl');
    expect(h1.className).toContain('font-bold');
  });

  it('keeps the description at 14px at every width — body copy never shrinks', () => {
    render(<SettingsPageHeader title="連線設定" description="設定 Vido 的連線方式。" />);
    const p = screen.getByText('設定 Vido 的連線方式。');
    expect(p.tagName).toBe('P');
    expect(p.className).toContain('text-sm');
    expect(p.className).not.toContain('text-xs');
    expect(p.className).not.toMatch(/sm:text-/);
  });

  // Desktop must not move a pixel: the ten hand-written headers used mb-2 / mb-6.
  it('keeps the spacing the hand-written headers had', () => {
    render(<SettingsPageHeader title="連線設定" description="說明" />);
    expect(screen.getByRole('heading', { level: 1 }).className).toContain('mb-2');
    expect(screen.getByText('說明').className).toContain('mb-6');
  });

  it('renders no paragraph when there is no description', () => {
    const { container } = render(<SettingsPageHeader title="外觀" />);
    expect(container.querySelector('p')).toBeNull();
    // Without a description the title carries the gap to the content itself.
    expect(screen.getByRole('heading', { level: 1 }).className).toContain('mb-6');
  });

  it('exposes an id on the title so a region can be labelled by it', () => {
    render(<SettingsPageHeader title="外觀" titleId="appearance-title" />);
    expect(screen.getByRole('heading', { level: 1 })).toHaveAttribute('id', 'appearance-title');
  });
});
