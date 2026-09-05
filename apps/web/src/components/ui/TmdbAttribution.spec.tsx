import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import {
  TmdbAttribution,
  TMDB_ATTRIBUTION_EN,
  TMDB_ATTRIBUTION_ZH,
  TMDB_HOME_URL,
  TMDB_LOGO_SRC,
} from './TmdbAttribution';

describe('TmdbAttribution', () => {
  it('quotes the §3 sentence verbatim in the full variant', () => {
    render(<TmdbAttribution />);
    // Written out here rather than compared to the exported constant: a test
    // that asserts `EN === EN` would pass through any future rewording, which
    // is the exact failure this compliance story exists to prevent.
    expect(
      screen.getByText(
        'This application uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.'
      )
    ).toBeInTheDocument();
    expect(TMDB_ATTRIBUTION_EN).toBe(
      'This application uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.'
    );
  });

  it('renders the zh-TW gloss alongside the English original, not instead of it', () => {
    render(<TmdbAttribution />);
    expect(screen.getByText(TMDB_ATTRIBUTION_ZH)).toBeInTheDocument();
    expect(screen.getByText(TMDB_ATTRIBUTION_EN)).toBeInTheDocument();
  });

  it('renders the official logo with an alt text and a safe outbound link', () => {
    render(<TmdbAttribution />);
    const logo = screen.getByAltText('TMDB') as HTMLImageElement;
    expect(logo.getAttribute('src')).toBe(TMDB_LOGO_SRC);

    const link = screen.getByTestId('tmdb-attribution-link');
    expect(link).toHaveAttribute('href', TMDB_HOME_URL);
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    expect(link).toHaveAttribute('target', '_blank');
  });

  it('falls back to a TMDB wordmark when the logo file cannot be loaded', () => {
    // Never a broken image (the ProviderLogo idiom) — and the link must keep an
    // accessible name, which the wordmark supplies.
    render(<TmdbAttribution />);
    fireEvent.error(screen.getByAltText('TMDB'));

    expect(screen.queryByAltText('TMDB')).not.toBeInTheDocument();
    expect(screen.getByTestId('tmdb-logo-fallback')).toHaveTextContent('TMDB');
    expect(screen.getByTestId('tmdb-attribution-link')).toHaveTextContent('TMDB');
  });

  it('inline variant reads 資料來源：TMDB and drops the long sentence', () => {
    render(<TmdbAttribution variant="inline" />);
    const el = screen.getByTestId('tmdb-attribution');
    expect(el).toHaveTextContent('資料來源：');
    expect(screen.getByAltText('TMDB')).toBeInTheDocument();
    // The detail-page line sits next to JustWatch's one-liner; the full notice
    // lives on the settings page.
    expect(screen.queryByText(TMDB_ATTRIBUTION_EN)).not.toBeInTheDocument();

    // With the image gone the line still reads as a whole sentence.
    fireEvent.error(screen.getByAltText('TMDB'));
    expect(screen.getByTestId('tmdb-attribution')).toHaveTextContent('資料來源：TMDB');
  });
});
