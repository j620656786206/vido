/**
 * Download Factory for Test Data Generation (Story 4.2)
 *
 * Creates download/torrent data for testing.
 * Pattern: Factory with overrides using faker.
 *
 * @see tests/support/fixtures/factories/qbittorrent-factory.ts
 */

import { faker } from '@faker-js/faker';

// =============================================================================
// Types
// =============================================================================

export type TorrentStatus =
  | 'downloading'
  | 'paused'
  | 'seeding'
  | 'completed'
  | 'stalled'
  | 'error'
  | 'queued'
  | 'checking';

export interface DownloadData {
  hash: string;
  name: string;
  size: number;
  progress: number;
  downloadSpeed: number;
  uploadSpeed: number;
  eta: number;
  status: TorrentStatus;
  addedOn: string;
  completedOn?: string;
  seeds: number;
  peers: number;
  downloaded: number;
  uploaded: number;
  ratio: number;
  savePath: string;
}

export interface DownloadDetailsData extends DownloadData {
  pieceSize: number;
  comment?: string;
  createdBy?: string;
  creationDate: string;
  totalWasted: number;
  timeElapsed: number;
  seedingTime: number;
  avgDownSpeed: number;
  avgUpSpeed: number;
}

export type PartialDownloadData = Partial<DownloadData>;
export type PartialDownloadDetailsData = Partial<DownloadDetailsData>;

// =============================================================================
// Factory Implementation
// =============================================================================

/**
 * Generate unique download data with optional overrides
 *
 * @example
 * const download = createDownloadData();
 * const custom = createDownloadData({ status: 'completed', progress: 1 });
 */
export function createDownloadData(overrides: PartialDownloadData = {}): DownloadData {
  const size = faker.number.int({ min: 500_000_000, max: 50_000_000_000 });
  const progress = faker.number.float({ min: 0, max: 1, fractionDigits: 2 });
  const downloaded = Math.floor(size * progress);

  return {
    hash: faker.string.hexadecimal({ length: 40, casing: 'lower', prefix: '' }),
    name: `[${faker.word.noun()}Sub] ${faker.lorem.words(3)} (${faker.date.recent().getFullYear()}) [1080p]`,
    size,
    progress,
    downloadSpeed: progress < 1 ? faker.number.int({ min: 0, max: 50_000_000 }) : 0,
    uploadSpeed: faker.number.int({ min: 0, max: 5_000_000 }),
    eta: progress < 1 ? faker.number.int({ min: 60, max: 86400 }) : 8640000,
    status: 'downloading',
    addedOn: faker.date.recent({ days: 7 }).toISOString(),
    seeds: faker.number.int({ min: 0, max: 100 }),
    peers: faker.number.int({ min: 0, max: 50 }),
    downloaded,
    uploaded: faker.number.int({ min: 0, max: downloaded }),
    ratio: faker.number.float({ min: 0, max: 5, fractionDigits: 2 }),
    savePath: `/downloads/${faker.helpers.arrayElement(['movies', 'series', 'anime'])}`,
    ...overrides,
  };
}

/**
 * Generate download details data with optional overrides
 */
export function createDownloadDetailsData(
  overrides: PartialDownloadDetailsData = {}
): DownloadDetailsData {
  const base = createDownloadData(overrides);

  return {
    ...base,
    pieceSize: faker.helpers.arrayElement([1_048_576, 2_097_152, 4_194_304]),
    comment: faker.helpers.maybe(() => faker.lorem.sentence(), { probability: 0.3 }),
    createdBy: `qBittorrent v${faker.number.int({ min: 4, max: 5 })}.${faker.number.int({ min: 0, max: 9 })}.${faker.number.int({ min: 0, max: 9 })}`,
    creationDate: faker.date.recent({ days: 30 }).toISOString(),
    totalWasted: faker.number.int({ min: 0, max: 10_000 }),
    timeElapsed: faker.number.int({ min: 60, max: 360_000 }),
    seedingTime: faker.number.int({ min: 0, max: 100_000 }),
    avgDownSpeed: faker.number.int({ min: 500_000, max: 30_000_000 }),
    avgUpSpeed: faker.number.int({ min: 50_000, max: 5_000_000 }),
    ...overrides,
  };
}

/**
 * Generate a list of downloads
 */
export function createDownloadList(count: number = 5): DownloadData[] {
  const statuses: TorrentStatus[] = ['downloading', 'seeding', 'completed', 'paused', 'stalled'];
  return Array.from({ length: count }, (_, i) =>
    createDownloadData({ status: statuses[i % statuses.length] })
  );
}

// =============================================================================
// Preset Downloads for Common Scenarios
// =============================================================================

export const presetDownloads = {
  /** Active download at 85% */
  downloading: createDownloadData({
    hash: 'a'.repeat(40),
    name: '[SubGroup] Movie Name (2026) [1080p].mkv',
    size: 4_294_967_296,
    progress: 0.85,
    downloadSpeed: 10_485_760,
    uploadSpeed: 524_288,
    eta: 600,
    status: 'downloading',
    seeds: 10,
    peers: 5,
    downloaded: 3_650_722_201,
    uploaded: 104_857_600,
    ratio: 0.03,
    savePath: '/downloads/movies',
  }),

  /** Completed download */
  completed: createDownloadData({
    hash: 'b'.repeat(40),
    name: 'Series S01 Complete [720p]',
    size: 8_589_934_592,
    progress: 1,
    downloadSpeed: 0,
    uploadSpeed: 262_144,
    eta: 8640000,
    status: 'completed',
    completedOn: '2026-01-15T12:00:00Z',
    seeds: 20,
    peers: 3,
    downloaded: 8_589_934_592,
    uploaded: 1_073_741_824,
    ratio: 0.125,
    savePath: '/downloads/series',
  }),

  /** Paused download */
  paused: createDownloadData({
    hash: 'c'.repeat(40),
    name: '[Fansub] Anime EP01 [1080p]',
    size: 1_073_741_824,
    progress: 0.3,
    downloadSpeed: 0,
    uploadSpeed: 0,
    eta: 8640000,
    status: 'paused',
    seeds: 0,
    peers: 0,
    savePath: '/downloads/anime',
  }),

  /** Seeding torrent */
  seeding: createDownloadData({
    hash: 'd'.repeat(40),
    name: 'Documentary (2025) [4K]',
    size: 15_032_385_536,
    progress: 1,
    downloadSpeed: 0,
    uploadSpeed: 1_048_576,
    eta: 8640000,
    status: 'seeding',
    seeds: 50,
    peers: 12,
    downloaded: 15_032_385_536,
    uploaded: 45_097_156_608,
    ratio: 3.0,
    savePath: '/downloads/movies',
  }),

  /** Error state */
  error: createDownloadData({
    hash: 'e'.repeat(40),
    name: 'Broken.Torrent.Missing.Files',
    size: 2_147_483_648,
    progress: 0.1,
    downloadSpeed: 0,
    uploadSpeed: 0,
    eta: 8640000,
    status: 'error',
    seeds: 0,
    peers: 0,
    savePath: '/downloads/movies',
  }),
} as const;
