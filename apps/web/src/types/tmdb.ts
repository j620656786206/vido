export interface Movie {
  id: number;
  title: string;
  originalTitle: string;
  overview: string;
  releaseDate: string;
  posterPath: string | null;
  backdropPath: string | null;
  voteAverage: number;
  voteCount: number;
  genreIds: number[];
}

export interface TVShow {
  id: number;
  name: string;
  originalName: string;
  overview: string;
  firstAirDate: string;
  posterPath: string | null;
  backdropPath: string | null;
  voteAverage: number;
  voteCount: number;
  genreIds: number[];
}

export interface SearchResponse<T> {
  page: number;
  results: T[];
  totalPages: number;
  totalResults: number;
}

export type MovieSearchResponse = SearchResponse<Movie>;
export type TVShowSearchResponse = SearchResponse<TVShow>;

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    suggestion?: string;
  };
}

export type MediaType = 'movie' | 'tv';

export interface MediaItem {
  id: number;
  title: string;
  originalTitle: string;
  overview: string;
  releaseDate: string;
  posterPath: string | null;
  backdropPath: string | null;
  voteAverage: number;
  voteCount: number;
  genreIds: number[];
  mediaType: MediaType;
}

// Detail types for Movie and TV Show pages
export interface Genre {
  id: number;
  name: string;
}

export interface ProductionCountry {
  iso31661: string;
  name: string;
}

export interface SpokenLanguage {
  englishName: string;
  iso6391: string;
  name: string;
}

export interface MovieDetails {
  id: number;
  title: string;
  originalTitle: string;
  overview: string;
  releaseDate: string;
  posterPath: string | null;
  backdropPath: string | null;
  voteAverage: number;
  voteCount: number;
  popularity: number;
  genreIds: number[];
  originalLanguage: string;
  adult: boolean;
  video: boolean;
  runtime: number;
  budget: number;
  revenue: number;
  status: string;
  tagline: string;
  genres: Genre[];
  productionCountries: ProductionCountry[];
  spokenLanguages: SpokenLanguage[];
  imdbId: string;
  homepage: string | null;
}

export interface Network {
  id: number;
  name: string;
  logoPath: string | null;
}

export interface Creator {
  id: number;
  creditId: string;
  name: string;
  gender: number;
  profilePath: string | null;
}

export interface Season {
  id: number;
  name: string;
  overview: string;
  posterPath: string | null;
  seasonNumber: number;
  episodeCount: number;
  airDate: string | null;
}

export interface TVShowDetails {
  id: number;
  name: string;
  originalName: string;
  overview: string;
  firstAirDate: string;
  lastAirDate: string;
  posterPath: string | null;
  backdropPath: string | null;
  voteAverage: number;
  voteCount: number;
  popularity: number;
  genreIds: number[];
  originalLanguage: string;
  originCountry: string[];
  episodeRunTime: number[];
  numberOfSeasons: number;
  numberOfEpisodes: number;
  status: string;
  type: string;
  tagline: string;
  genres: Genre[];
  createdBy: Creator[];
  homepage: string | null;
  inProduction: boolean;
  languages: string[];
  networks: Network[];
  productionCountries: ProductionCountry[];
  seasons: Season[];
}

// Vido internal API types for Series/Season/Episode 3-tier architecture
export interface SeasonDetail {
  id: string;
  seriesId: string;
  tmdbId?: number;
  seasonNumber: number;
  name?: string;
  overview?: string;
  posterPath?: string;
  airDate?: string;
  episodeCount?: number;
  voteAverage?: number;
  createdAt: string;
  updatedAt: string;
}

export interface EpisodeDetail {
  id: string;
  seriesId: string;
  seasonId?: string;
  tmdbId?: number;
  seasonNumber: number;
  episodeNumber: number;
  title?: string;
  overview?: string;
  airDate?: string;
  runtime?: number;
  stillPath?: string;
  voteAverage?: number;
  filePath?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CastMember {
  id: number;
  name: string;
  character: string;
  profilePath: string | null;
  order: number;
}

export interface CrewMember {
  id: number;
  name: string;
  job: string;
  department: string;
  profilePath: string | null;
}

export interface Credits {
  id: number;
  cast: CastMember[];
  crew: CrewMember[];
}

// Story 10-2 — TMDb /videos endpoint response (trailers, teasers, etc.)
export interface Video {
  key: string;
  name: string;
  site: string;
  type: string;
  official: boolean;
  publishedAt: string;
}

export interface VideosResponse {
  id: number;
  results: Video[];
}

// Story 10-2 — Hero banner item (movie or TV) normalized for carousel display.
export interface HeroBannerItem {
  id: number;
  mediaType: MediaType;
  title: string;
  overview: string;
  backdropPath: string | null;
  releaseDate: string; // movie release_date OR series first_air_date
  voteAverage: number;
}
