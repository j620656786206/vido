import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { LocalizationSettings } from '../../services/subtitleLocalizationService';

const h = vi.hoisted(() => ({
  query: {
    data: undefined as LocalizationSettings | undefined,
    isLoading: false,
    isError: false,
    error: null as Error | null,
  },
  save: { mutate: vi.fn(), isPending: false, isError: false, error: null as Error | null },
}));

vi.mock('../../hooks/useSubtitleLocalization', () => ({
  useSubtitleLocalization: () => h.query,
  useSaveSubtitleLocalization: () => h.save,
}));

import { LocalizationLevelForm, LEVEL_SPECS } from './LocalizationLevelForm';

const LEVELS: LocalizationSettings['levels'] = ['literal', 'standard', 'ott'];

function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <LocalizationLevelForm />
    </QueryClientProvider>
  );
}

describe('LocalizationLevelForm (sub-7-4 AC #4)', () => {
  beforeEach(() => {
    h.query.data = { level: 'standard', source: 'default', levels: LEVELS };
    h.query.isLoading = false;
    h.query.isError = false;
    h.query.error = null;
    h.save.mutate.mockReset();
    h.save.isPending = false;
    h.save.isError = false;
    h.save.error = null;
  });

  it('renders three options, each with one sentence and one example', () => {
    renderForm();
    for (const spec of LEVEL_SPECS) {
      const option = screen.getByTestId(`localization-option-${spec.id}`);
      expect(option).toHaveTextContent(spec.label);
      expect(option).toHaveTextContent(spec.description);
      expect(option).toHaveTextContent(`例：${spec.example}`);
    }
    expect(screen.getAllByRole('radio')).toHaveLength(3);
  });

  it('checks the level in force', () => {
    h.query.data = { level: 'ott', source: 'settings', levels: LEVELS };
    renderForm();
    expect(screen.getByTestId('localization-option-ott')).toHaveAttribute('data-selected', 'true');
    expect(screen.getByTestId('localization-option-standard')).toHaveAttribute(
      'data-selected',
      'false'
    );
    expect(screen.getByRole('radio', { name: /OTT 風格/ })).toBeChecked();
  });

  it('saves on change', () => {
    renderForm();
    fireEvent.click(screen.getByRole('radio', { name: /直譯/ }));
    expect(h.save.mutate).toHaveBeenCalledWith('literal');
  });

  it('is honest about an env-sourced level', () => {
    h.query.data = { level: 'literal', source: 'env', levels: LEVELS };
    renderForm();
    expect(screen.getByTestId('localization-env-note')).toHaveTextContent(
      'SUBTITLE_LOCALIZATION_LEVEL'
    );
  });

  it('does not show the env note for a settings-sourced level', () => {
    h.query.data = { level: 'literal', source: 'settings', levels: LEVELS };
    renderForm();
    expect(screen.queryByTestId('localization-env-note')).toBeNull();
  });

  it('shows a load error and still renders the options', () => {
    h.query.data = undefined;
    h.query.isError = true;
    h.query.error = new Error('boom');
    renderForm();
    expect(screen.getByTestId('localization-load-error')).toHaveTextContent('boom');
    expect(screen.getAllByRole('radio')).toHaveLength(3);
  });

  it('shows a save error', () => {
    h.save.isError = true;
    h.save.error = new Error('db locked');
    renderForm();
    expect(screen.getByTestId('localization-save-error')).toHaveTextContent('db locked');
  });

  it('shows the loading state', () => {
    h.query.isLoading = true;
    renderForm();
    expect(screen.getByTestId('localization-loading')).toBeInTheDocument();
  });

  describe('dsr-3d', () => {
    it('has no outer card; each option is its own card, the chosen one gold', () => {
      renderForm();
      expect(screen.getByTestId('localization-form').className).not.toContain('bg-secondary');
      const chosen = screen.getByTestId('localization-option-standard');
      const other = screen.getByTestId('localization-option-literal');
      expect(chosen.className).toContain('bg-[var(--accent-subtle)]');
      expect(chosen.className).toContain('border-[var(--accent-primary)]');
      expect(chosen.className).toContain('rounded-[var(--radius-lg)]');
      expect(other.className).toContain('bg-[var(--bg-secondary)]');
      expect(other.className).toContain('border-[var(--border-subtle)]');
    });

    it('keeps the native radios: arrow keys move the choice', async () => {
      const user = userEvent.setup();
      renderForm();
      const standard = screen.getByRole('radio', { name: /台灣用語/ });
      expect(standard.className).toContain('sr-only');
      standard.focus();
      await user.keyboard('{ArrowDown}');
      expect(h.save.mutate).toHaveBeenCalledWith('ott');
    });

    it('description is 12px; the example is mono with「例：」only for screen readers', () => {
      renderForm();
      const option = screen.getByTestId('localization-option-literal');
      const spans = option.querySelectorAll(':scope > span');
      const description = [...spans].find((sp) => sp.textContent === LEVEL_SPECS[0].description)!;
      expect(description.className).toContain('text-xs');
      const example = [...spans].find((sp) => sp.textContent?.endsWith(LEVEL_SPECS[0].example))!;
      expect(example.className).toContain('font-mono');
      expect(example.querySelector('.sr-only')).toHaveTextContent('例：');
    });

    it('does not repeat the「only future subtitles」promise at the bottom (the intro already says it)', () => {
      renderForm();
      expect(screen.getByText(/已經翻好的不會動/)).toBeInTheDocument();
      expect(screen.queryByText(/已經翻好的不會重跑/)).toBeNull();
    });
  });

  it('dsr-3d: while saving, the radios stay focusable and a second change is ignored', () => {
    h.save.isPending = true;
    renderForm();
    const ott = screen.getByRole('radio', { name: /OTT 風格/ });
    expect(ott).not.toBeDisabled();
    fireEvent.click(ott);
    expect(h.save.mutate).not.toHaveBeenCalled();
  });
});
