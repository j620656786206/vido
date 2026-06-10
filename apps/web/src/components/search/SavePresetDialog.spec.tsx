import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { SavePresetDialog } from './SavePresetDialog';
import type { DiscoverFilters } from '../../lib/discoverFilters';

// Mock the TanStack Query mutation hook so the dialog is unit-testable.
const mutateAsync = vi.fn();
let isPending = false;
vi.mock('../../hooks/useFilterPresets', () => ({
  useCreateFilterPreset: () => ({ mutateAsync, isPending }),
}));

const filters: DiscoverFilters = {
  genre: [16],
  yearGte: 2023,
  yearLte: 2024,
  ratingGte: 7,
  platform: [],
  sortBy: 'popularity',
};

describe('SavePresetDialog', () => {
  beforeEach(() => {
    mutateAsync.mockReset();
    isPending = false;
  });

  it('renders the title, name field and a preview of active filters (AC #1)', () => {
    render(<SavePresetDialog filters={filters} onClose={vi.fn()} />);
    expect(screen.getByText('儲存篩選條件')).toBeInTheDocument();
    expect(screen.getByTestId('preset-name-input')).toBeInTheDocument();
    const preview = screen.getByTestId('save-preset-preview');
    expect(preview).toHaveTextContent('類型: 動畫');
    expect(preview).toHaveTextContent('年份: 2023-2024');
    expect(preview).toHaveTextContent('評分: 7+');
  });

  it('disables save when the name is empty', () => {
    render(<SavePresetDialog filters={filters} onClose={vi.fn()} />);
    expect(screen.getByTestId('save-preset-confirm')).toBeDisabled();
  });

  it('caps the name input at 30 chars (Task 2.3)', () => {
    render(<SavePresetDialog filters={filters} onClose={vi.fn()} />);
    const input = screen.getByTestId('preset-name-input') as HTMLInputElement;
    expect(input.maxLength).toBe(30);
  });

  it('focuses the name field on open via ref (retro-11-AI1b — replaces autoFocus)', () => {
    render(<SavePresetDialog filters={filters} onClose={vi.fn()} />);
    expect(screen.getByTestId('preset-name-input')).toHaveFocus();
  });

  it('associates the 預設名稱 label with the name input (retro-11-AI1b)', () => {
    render(<SavePresetDialog filters={filters} onClose={vi.fn()} />);
    expect(screen.getByLabelText('預設名稱')).toBe(screen.getByTestId('preset-name-input'));
  });

  it('saves the current filters serialized as a JSON string and closes (AC #1)', async () => {
    mutateAsync.mockResolvedValue({ id: 'p1' });
    const onClose = vi.fn();
    render(<SavePresetDialog filters={filters} onClose={onClose} />);

    fireEvent.change(screen.getByTestId('preset-name-input'), {
      target: { value: '高評分動畫' },
    });
    fireEvent.click(screen.getByTestId('save-preset-confirm'));

    await waitFor(() => expect(mutateAsync).toHaveBeenCalledTimes(1));
    const arg = mutateAsync.mock.calls[0][0];
    expect(arg.name).toBe('高評分動畫');
    // filters must be a JSON string in URL-param shape (not a nested object).
    expect(typeof arg.filters).toBe('string');
    expect(JSON.parse(arg.filters)).toMatchObject({
      genre: '16',
      year_gte: 2023,
      year_lte: 2024,
      rating_gte: 7,
    });
    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it('shows an error message when the mutation rejects', async () => {
    mutateAsync.mockRejectedValue(new Error('已達上限'));
    render(<SavePresetDialog filters={filters} onClose={vi.fn()} />);

    fireEvent.change(screen.getByTestId('preset-name-input'), { target: { value: '太多了' } });
    fireEvent.click(screen.getByTestId('save-preset-confirm'));

    expect(await screen.findByTestId('save-preset-error')).toHaveTextContent('已達上限');
  });

  it('cancel calls onClose without saving', () => {
    const onClose = vi.fn();
    render(<SavePresetDialog filters={filters} onClose={onClose} />);
    fireEvent.click(screen.getByTestId('save-preset-cancel'));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(mutateAsync).not.toHaveBeenCalled();
  });
});
