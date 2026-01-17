import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { TVShowInfo } from './TVShowInfo';
import type { TVShowDetails } from '../../types/tmdb';

const mockTVShow: TVShowDetails = {
  id: 456,
  name: '測試影集',
  original_name: 'Test TV Show',
  overview: '這是一部測試影集',
  first_air_date: '2023-06-15',
  last_air_date: '2024-01-20',
  poster_path: '/poster.jpg',
  backdrop_path: '/backdrop.jpg',
  vote_average: 8.5,
  vote_count: 2000,
  episode_run_time: [45, 50],
  number_of_seasons: 3,
  number_of_episodes: 30,
  status: 'Returning Series',
  type: 'Scripted',
  tagline: '',
  genres: [{ id: 1, name: '劇情' }],
  created_by: [],
  networks: [
    { id: 1, name: 'Netflix', logo_path: null },
    { id: 2, name: 'HBO', logo_path: null },
  ],
  in_production: true,
  seasons: [],
};

describe('TVShowInfo', () => {
  it('should render section title', () => {
    render(<TVShowInfo show={mockTVShow} />);
    expect(screen.getByText('影集資訊')).toBeInTheDocument();
  });

  describe('Seasons and Episodes', () => {
    it('should render number of seasons', () => {
      render(<TVShowInfo show={mockTVShow} />);
      expect(screen.getByTestId('seasons-count')).toHaveTextContent('3 季');
    });

    it('should render number of episodes', () => {
      render(<TVShowInfo show={mockTVShow} />);
      expect(screen.getByTestId('episodes-count')).toHaveTextContent('30 集');
    });
  });

  describe('Status', () => {
    it('should translate "Returning Series" status', () => {
      render(<TVShowInfo show={mockTVShow} />);
      expect(screen.getByTestId('show-status')).toHaveTextContent('回歸中');
    });

    it('should translate "Ended" status', () => {
      const endedShow = { ...mockTVShow, status: 'Ended' };
      render(<TVShowInfo show={endedShow} />);
      expect(screen.getByTestId('show-status')).toHaveTextContent('已完結');
    });

    it('should translate "Canceled" status', () => {
      const canceledShow = { ...mockTVShow, status: 'Canceled' };
      render(<TVShowInfo show={canceledShow} />);
      expect(screen.getByTestId('show-status')).toHaveTextContent('已取消');
    });

    it('should show original status if not in translation map', () => {
      const unknownStatus = { ...mockTVShow, status: 'Unknown Status' };
      render(<TVShowInfo show={unknownStatus} />);
      expect(screen.getByTestId('show-status')).toHaveTextContent('Unknown Status');
    });
  });

  describe('Air Dates', () => {
    it('should render first air date in Traditional Chinese format', () => {
      render(<TVShowInfo show={mockTVShow} />);
      const firstAirDate = screen.getByTestId('first-air-date');
      // Date format: 2023年6月15日
      expect(firstAirDate).toHaveTextContent('2023');
      expect(firstAirDate).toHaveTextContent('6');
      expect(firstAirDate).toHaveTextContent('15');
    });

    it('should render last air date when available', () => {
      render(<TVShowInfo show={mockTVShow} />);
      const lastAirDate = screen.getByTestId('last-air-date');
      expect(lastAirDate).toHaveTextContent('2024');
    });

    it('should not render last air date when null', () => {
      const showNoLastDate = { ...mockTVShow, last_air_date: '' };
      render(<TVShowInfo show={showNoLastDate} />);
      expect(screen.queryByTestId('last-air-date')).not.toBeInTheDocument();
    });
  });

  describe('Networks', () => {
    it('should render networks', () => {
      render(<TVShowInfo show={mockTVShow} />);
      expect(screen.getByTestId('networks')).toHaveTextContent('Netflix, HBO');
    });

    it('should not render networks section when empty', () => {
      const showNoNetworks = { ...mockTVShow, networks: [] };
      render(<TVShowInfo show={showNoNetworks} />);
      expect(screen.queryByTestId('networks')).not.toBeInTheDocument();
    });
  });

  describe('Show Type', () => {
    it('should render show type', () => {
      render(<TVShowInfo show={mockTVShow} />);
      expect(screen.getByTestId('show-type')).toHaveTextContent('Scripted');
    });

    it('should not render type when empty', () => {
      const showNoType = { ...mockTVShow, type: '' };
      render(<TVShowInfo show={showNoType} />);
      expect(screen.queryByTestId('show-type')).not.toBeInTheDocument();
    });
  });
});
