import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { CreditsSection } from './CreditsSection';
import type { CastMember, CrewMember, Creator } from '../../types/tmdb';

const mockDirector: CrewMember = {
  id: 1,
  name: '導演名',
  job: 'Director',
  department: 'Directing',
  profilePath: '/director.jpg',
};

const mockCast: CastMember[] = [
  { id: 1, name: '演員一', character: '角色一', profilePath: '/actor1.jpg', order: 0 },
  { id: 2, name: '演員二', character: '角色二', profilePath: '/actor2.jpg', order: 1 },
  { id: 3, name: '演員三', character: '角色三', profilePath: null, order: 2 },
  { id: 4, name: '演員四', character: '角色四', profilePath: '/actor4.jpg', order: 3 },
  { id: 5, name: '演員五', character: '角色五', profilePath: '/actor5.jpg', order: 4 },
  { id: 6, name: '演員六', character: '角色六', profilePath: '/actor6.jpg', order: 5 },
  { id: 7, name: '演員七', character: '角色七', profilePath: '/actor7.jpg', order: 6 },
];

const mockCreatedBy: Creator[] = [
  { id: 1, name: '創作者一', profilePath: '/creator1.jpg' },
  { id: 2, name: '創作者二', profilePath: null },
];

describe('CreditsSection', () => {
  it('should not render when no credits data', () => {
    const { container } = render(<CreditsSection />);
    expect(container.firstChild).toBeNull();
  });

  describe('Director', () => {
    it('should render director when provided', () => {
      render(<CreditsSection director={mockDirector} />);
      expect(screen.getByText('導演')).toBeInTheDocument();
      expect(screen.getByText('導演名')).toBeInTheDocument();
    });

    it('should render director profile image', () => {
      render(<CreditsSection director={mockDirector} />);
      const images = screen.getAllByRole('img');
      expect(images.some((img) => img.getAttribute('src')?.includes('/director.jpg'))).toBe(true);
    });
  });

  describe('Created By', () => {
    it('should render created by for TV shows', () => {
      render(<CreditsSection createdBy={mockCreatedBy} />);
      expect(screen.getByText('創作者')).toBeInTheDocument();
      expect(screen.getByText('創作者一')).toBeInTheDocument();
      expect(screen.getByText('創作者二')).toBeInTheDocument();
    });
  });

  describe('Cast', () => {
    it('should render cast section when cast provided', () => {
      render(<CreditsSection cast={mockCast} />);
      expect(screen.getByText('演員陣容')).toBeInTheDocument();
    });

    it('should render actor names', () => {
      render(<CreditsSection cast={mockCast.slice(0, 3)} />);
      expect(screen.getByText('演員一')).toBeInTheDocument();
      expect(screen.getByText('演員二')).toBeInTheDocument();
      expect(screen.getByText('演員三')).toBeInTheDocument();
    });

    it('should render character names', () => {
      render(<CreditsSection cast={mockCast.slice(0, 3)} />);
      expect(screen.getByText('角色一')).toBeInTheDocument();
      expect(screen.getByText('角色二')).toBeInTheDocument();
    });

    it('should limit cast to 6 members', () => {
      render(<CreditsSection cast={mockCast} />);
      const castGrid = screen.getByTestId('cast-grid');
      // Should only show 6 cast members
      expect(castGrid.children.length).toBe(6);
      expect(screen.getByText('演員六')).toBeInTheDocument();
      expect(screen.queryByText('演員七')).not.toBeInTheDocument();
    });

    it('should handle missing profile images gracefully', () => {
      render(<CreditsSection cast={[mockCast[2]]} />);
      // Should show fallback emoji instead of image
      expect(screen.getByText('👤')).toBeInTheDocument();
    });

    it('should render profile images for cast with profilePath', () => {
      render(<CreditsSection cast={[mockCast[0]]} />);
      const images = screen.getAllByRole('img');
      expect(images.some((img) => img.getAttribute('src')?.includes('/actor1.jpg'))).toBe(true);
    });
  });

  describe('Combined display', () => {
    it('should render director, created by, and cast together', () => {
      render(
        <CreditsSection
          director={mockDirector}
          createdBy={mockCreatedBy}
          cast={mockCast.slice(0, 3)}
        />
      );
      expect(screen.getByText('導演')).toBeInTheDocument();
      expect(screen.getByText('創作者')).toBeInTheDocument();
      expect(screen.getByText('演員陣容')).toBeInTheDocument();
    });
  });
});
