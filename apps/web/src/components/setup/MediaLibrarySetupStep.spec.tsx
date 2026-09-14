/**
 * MediaLibrarySetupStep coverage.
 * The label-association test is retro-11-AI1b (htmlFor/id fixes); dsr-13 kept
 * both accessible names while N3-D dropped the visible labels and replaced the
 * type <select> with a two-button toggle.
 */
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MediaLibrarySetupStep } from './MediaLibrarySetupStep';
import type { StepProps } from './SetupWizard';

const libraries = [
  { id: 'a', path: '/media/movies', contentType: 'movie' as const },
  { id: 'b', path: '/media/series', contentType: 'series' as const },
];

function renderStep(overrides?: Partial<StepProps>) {
  const props: StepProps = {
    data: { libraries },
    onUpdate: vi.fn(),
    onNext: vi.fn(),
    onBack: vi.fn(),
    isFirst: false,
    isLast: false,
    ...overrides,
  };
  render(<MediaLibrarySetupStep {...props} />);
  return props;
}

describe('MediaLibrarySetupStep', () => {
  it('associates per-entry labels with their indexed controls (retro-11-AI1b htmlFor/id)', () => {
    renderStep();

    const pathInputs = screen.getAllByLabelText('資料夾路徑');
    expect(pathInputs).toHaveLength(2);
    expect(pathInputs[0]).toBe(screen.getByTestId('library-path-0'));
    expect(pathInputs[1]).toBe(screen.getByTestId('library-path-1'));

    const typeGroups = screen.getAllByLabelText('類型');
    expect(typeGroups).toHaveLength(2);
    expect(typeGroups[0]).toBe(screen.getByTestId('library-type-0'));
    expect(typeGroups[1]).toBe(screen.getByTestId('library-type-1'));
  });

  it('picks the content type with a toggle, not a dropdown (N3-D)', () => {
    const { onUpdate } = renderStep();
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.getByTestId('library-type-0-movie')).toBeChecked();
    expect(screen.getByTestId('library-type-1-series')).toBeChecked();

    fireEvent.click(screen.getByTestId('library-type-0-series'));
    expect(onUpdate).toHaveBeenCalledWith({
      libraries: [{ ...libraries[0], contentType: 'series' }, libraries[1]],
    });
  });

  it('keeps each row’s toggle in its own radio group', () => {
    renderStep();
    const [first, second] = screen.getAllByRole('radiogroup');
    expect(first.querySelectorAll('input[type=radio]')).toHaveLength(2);
    expect(second.querySelectorAll('input[type=radio]')).toHaveLength(2);
    const name = (el: Element) => (el.querySelector('input') as HTMLInputElement).name;
    expect(name(first)).not.toBe(name(second));
  });

  it('removes a row', () => {
    const { onUpdate } = renderStep();
    fireEvent.click(screen.getByTestId('library-remove-0'));
    expect(onUpdate).toHaveBeenCalledWith({ libraries: [libraries[1]] });
  });

  it('has no remove button when there is a single library', () => {
    renderStep({ data: { libraries: [libraries[0]] } });
    expect(screen.queryByTestId('library-remove-0')).not.toBeInTheDocument();
  });

  it('adds an empty movie row', () => {
    const { onUpdate } = renderStep();
    fireEvent.click(screen.getByTestId('add-library-button'));
    const [{ libraries: next }] = vi.mocked(onUpdate).mock.calls[0] as [
      { libraries: typeof libraries },
    ];
    expect(next).toHaveLength(3);
    expect(next[2]).toMatchObject({ path: '', contentType: 'movie' });
  });

  it('disables 下一步 while any path is blank', () => {
    renderStep({
      data: { libraries: [libraries[0], { id: 'c', path: '  ', contentType: 'movie' }] },
    });
    expect(screen.getByTestId('next-button')).toBeDisabled();
  });
});
