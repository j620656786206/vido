/**
 * MediaLibrarySetupStep label-association coverage (retro-11-AI1b).
 * Asserts the jsx-a11y htmlFor/id fixes: the per-entry 資料夾路徑 and 類型
 * labels resolve to their indexed controls via getByLabelText.
 */
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MediaLibrarySetupStep } from './MediaLibrarySetupStep';

const libraries = [
  { id: 'a', path: '/media/movies', contentType: 'movie' as const },
  { id: 'b', path: '/media/series', contentType: 'series' as const },
];

describe('MediaLibrarySetupStep', () => {
  it('associates per-entry labels with their indexed controls (retro-11-AI1b htmlFor/id)', () => {
    render(
      <MediaLibrarySetupStep
        data={{ libraries }}
        onUpdate={vi.fn()}
        onNext={vi.fn()}
        onBack={vi.fn()}
        isFirst={false}
        isLast={false}
      />
    );

    const pathInputs = screen.getAllByLabelText('資料夾路徑');
    expect(pathInputs).toHaveLength(2);
    expect(pathInputs[0]).toBe(screen.getByTestId('library-path-0'));
    expect(pathInputs[1]).toBe(screen.getByTestId('library-path-1'));

    const typeSelects = screen.getAllByLabelText('類型');
    expect(typeSelects).toHaveLength(2);
    expect(typeSelects[0]).toBe(screen.getByTestId('library-type-0'));
    expect(typeSelects[1]).toBe(screen.getByTestId('library-type-1'));
  });
});
