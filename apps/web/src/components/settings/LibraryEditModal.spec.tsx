/**
 * LibraryEditModal label-association coverage (retro-11-AI1b).
 * Asserts the jsx-a11y htmlFor/id fixes: every visible form label resolves to
 * its control via getByLabelText, and icon-only buttons carry accessible names.
 */
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { LibraryEditModal } from './LibraryEditModal';

const mutation = { mutateAsync: vi.fn().mockResolvedValue({}), isPending: false };

vi.mock('../../hooks/useMediaLibrary', () => ({
  useMediaLibraries: vi.fn(() => ({
    data: {
      libraries: [
        {
          id: 'lib-1',
          name: '我的電影',
          contentType: 'movie',
          paths: [{ id: 'path-1', path: '/media/movies' }],
        },
      ],
    },
  })),
  useCreateLibrary: () => mutation,
  useUpdateLibrary: () => mutation,
  useAddLibraryPath: () => mutation,
  useRemoveLibraryPath: () => mutation,
}));

describe('LibraryEditModal', () => {
  beforeEach(() => {
    mutation.mutateAsync.mockClear();
  });

  it('associates every form label with its control (retro-11-AI1b htmlFor/id)', () => {
    render(<LibraryEditModal onClose={vi.fn()} />);

    expect(screen.getByLabelText('名稱')).toBe(screen.getByTestId('library-name-input'));
    expect(screen.getByLabelText('類型')).toBe(screen.getByTestId('library-type-select'));
    expect(screen.getByLabelText('資料夾路徑')).toBe(screen.getByTestId('library-path-input'));
  });

  it('gives icon-only buttons accessible names (retro-11-AI1b)', () => {
    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(screen.getByRole('button', { name: '關閉' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '移除路徑 /media/movies' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '新增路徑' })).toBeInTheDocument();
  });
});
