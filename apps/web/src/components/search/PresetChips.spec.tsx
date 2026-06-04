import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PresetChips } from './PresetChips';
import type { FilterPreset } from '../../services/filterPresetService';

// Mockable hook state.
const deleteMutateAsync = vi.fn();
let presetsData: FilterPreset[] | undefined;
let deletePending = false;

vi.mock('../../hooks/useFilterPresets', () => ({
  useFilterPresets: () => ({ data: presetsData }),
  useDeleteFilterPreset: () => ({ mutateAsync: deleteMutateAsync, isPending: deletePending }),
}));

const PRESETS: FilterPreset[] = [
  {
    id: 'p1',
    name: '2024年後韓劇',
    filters: JSON.stringify({ region: 'KR', year_gte: 2024 }),
    sortOrder: 0,
    createdAt: '2026-06-04T00:00:00Z',
  },
  {
    id: 'p2',
    name: '高評分動畫',
    filters: JSON.stringify({ genre: '16', rating_gte: 8 }),
    sortOrder: 1,
    createdAt: '2026-06-04T00:00:00Z',
  },
];

describe('PresetChips', () => {
  beforeEach(() => {
    deleteMutateAsync.mockReset();
    deletePending = false;
    presetsData = PRESETS;
  });

  it('renders nothing when there are no saved presets', () => {
    presetsData = [];
    const { container } = render(<PresetChips onApplyPreset={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders a chip per saved preset above the filter area (AC #2)', () => {
    render(<PresetChips onApplyPreset={vi.fn()} />);
    expect(screen.getByText('快速篩選:')).toBeInTheDocument();
    expect(screen.getByTestId('preset-chip-p1')).toHaveTextContent('2024年後韓劇');
    expect(screen.getByTestId('preset-chip-p2')).toHaveTextContent('高評分動畫');
  });

  it('clicking a chip applies its parsed filters (AC #3)', () => {
    const onApplyPreset = vi.fn();
    render(<PresetChips onApplyPreset={onApplyPreset} />);
    fireEvent.click(screen.getByTestId('preset-chip-p1'));
    expect(onApplyPreset).toHaveBeenCalledWith(
      expect.objectContaining({ region: 'KR', yearGte: 2024 })
    );
  });

  it('right-click opens a delete confirmation (AC #4)', () => {
    render(<PresetChips onApplyPreset={vi.fn()} />);
    expect(screen.queryByTestId('preset-delete-dialog')).not.toBeInTheDocument();
    fireEvent.contextMenu(screen.getByTestId('preset-chip-p2'));
    const dialog = screen.getByTestId('preset-delete-dialog');
    expect(dialog).toBeInTheDocument();
    expect(dialog).toHaveTextContent('高評分動畫');
  });

  it('confirming delete calls the delete mutation (AC #4)', async () => {
    deleteMutateAsync.mockResolvedValue(undefined);
    render(<PresetChips onApplyPreset={vi.fn()} />);
    fireEvent.contextMenu(screen.getByTestId('preset-chip-p1'));
    fireEvent.click(screen.getByTestId('preset-delete-confirm'));
    await waitFor(() => expect(deleteMutateAsync).toHaveBeenCalledWith('p1'));
    await waitFor(() =>
      expect(screen.queryByTestId('preset-delete-dialog')).not.toBeInTheDocument()
    );
  });

  it('cancelling delete does not call the mutation', () => {
    render(<PresetChips onApplyPreset={vi.fn()} />);
    fireEvent.contextMenu(screen.getByTestId('preset-chip-p1'));
    fireEvent.click(screen.getByTestId('preset-delete-cancel'));
    expect(deleteMutateAsync).not.toHaveBeenCalled();
    expect(screen.queryByTestId('preset-delete-dialog')).not.toBeInTheDocument();
  });

  it('long-press opens delete dialog without applying the preset (AC #4, Task 3.4)', () => {
    vi.useFakeTimers();
    try {
      const onApplyPreset = vi.fn();
      render(<PresetChips onApplyPreset={onApplyPreset} />);
      const chip = screen.getByTestId('preset-chip-p1');
      fireEvent.touchStart(chip);
      act(() => vi.advanceTimersByTime(500)); // long-press threshold
      fireEvent.touchEnd(chip);
      // The browser synthesizes a click after touchend — it must be swallowed.
      fireEvent.click(chip);
      expect(screen.getByTestId('preset-delete-dialog')).toBeInTheDocument();
      expect(onApplyPreset).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it('a short tap still applies the preset on touch devices (AC #3)', () => {
    vi.useFakeTimers();
    try {
      const onApplyPreset = vi.fn();
      render(<PresetChips onApplyPreset={onApplyPreset} />);
      const chip = screen.getByTestId('preset-chip-p1');
      fireEvent.touchStart(chip);
      act(() => vi.advanceTimersByTime(100)); // below the long-press threshold
      fireEvent.touchEnd(chip);
      fireEvent.click(chip);
      expect(onApplyPreset).toHaveBeenCalledWith(
        expect.objectContaining({ region: 'KR', yearGte: 2024 })
      );
      expect(screen.queryByTestId('preset-delete-dialog')).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it('falls back gracefully when a preset has corrupt filter JSON', () => {
    presetsData = [{ ...PRESETS[0], filters: '{bad json' }];
    const onApplyPreset = vi.fn();
    render(<PresetChips onApplyPreset={onApplyPreset} />);
    fireEvent.click(screen.getByTestId('preset-chip-p1'));
    // Empty/default filter set rather than a throw.
    expect(onApplyPreset).toHaveBeenCalledWith(
      expect.objectContaining({ genre: [], platform: [], sortBy: 'popularity' })
    );
  });
});
