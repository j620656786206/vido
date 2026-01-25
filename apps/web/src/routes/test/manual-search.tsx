/**
 * Test Route: Manual Search E2E Testing (Story 3-7)
 *
 * This route provides test fixtures for E2E testing of the manual search
 * and apply metadata flow. Only available in development/test environments.
 *
 * Path: /test/manual-search
 *
 * Test Scenarios:
 * - Movie with parsed info (Fight Club)
 * - TV Show with parsed info (Breaking Bad)
 * - File with no parsed info (unknown file)
 * - Anime with partial parse (Demon Slayer)
 *
 * @internal This route is for testing purposes only
 */

import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { ParseFailureCard, type LocalMediaFile } from '../../components/library/ParseFailureCard';

// Protect test route in production
const isTestEnvironment =
  import.meta.env.DEV ||
  import.meta.env.MODE === 'test' ||
  (typeof window !== 'undefined' && window.location.hostname === 'localhost');

export const Route = createFileRoute('/test/manual-search')({
  component: TestManualSearchPage,
});

// Mock test data for different scenarios
const MOCK_FILES: LocalMediaFile[] = [
  {
    id: 'test-movie-001',
    filename: 'Fight.Club.1999.BluRay.1080p.x264-GROUP.mkv',
    path: '/media/movies/Fight.Club.1999.BluRay.1080p.x264-GROUP.mkv',
    size: 8_500_000_000,
    parsedInfo: {
      title: 'Fight Club',
      year: 1999,
      mediaType: 'movie',
    },
    metadataStatus: 'failed',
    fallbackStatus: {
      attempts: [
        { source: 'tmdb', success: false },
        { source: 'douban', success: false },
      ],
      totalDuration: 2500,
    },
  },
  {
    id: 'test-series-001',
    filename: 'Breaking.Bad.S01E01.Pilot.720p.BluRay.x264.mkv',
    path: '/media/series/Breaking Bad/S01E01.mkv',
    size: 1_200_000_000,
    parsedInfo: {
      title: 'Breaking Bad',
      year: 2008,
      mediaType: 'tv',
      season: 1,
      episode: 1,
    },
    metadataStatus: 'failed',
    fallbackStatus: {
      attempts: [
        { source: 'tmdb', success: false },
      ],
      totalDuration: 1500,
    },
  },
  {
    id: 'test-unknown-001',
    filename: 'random_video_file_2024.mp4',
    path: '/media/unsorted/random_video_file_2024.mp4',
    size: 700_000_000,
    metadataStatus: 'pending',
  },
  {
    id: 'test-anime-001',
    filename: '[SubGroup] Demon Slayer - 01 (1080p).mkv',
    path: '/media/anime/Demon Slayer/01.mkv',
    size: 500_000_000,
    parsedInfo: {
      title: 'Demon Slayer',
      mediaType: 'tv',
      episode: 1,
    },
    metadataStatus: 'failed',
    fallbackStatus: {
      attempts: [
        { source: 'tmdb', success: false },
        { source: 'douban', success: false },
        { source: 'wikipedia', success: false, skipped: true, skipReason: 'No Wikipedia provider configured' },
      ],
      totalDuration: 3200,
    },
  },
];

function TestManualSearchPage() {
  const [appliedItems, setAppliedItems] = useState<string[]>([]);

  // Block access in production
  if (!isTestEnvironment) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-white mb-4">Access Denied</h1>
          <p className="text-slate-400">This page is only available in test environments.</p>
        </div>
      </div>
    );
  }

  const handleMetadataApplied = (fileId: string) => {
    setAppliedItems((prev) => [...prev, fileId]);
  };

  const availableFiles = MOCK_FILES.filter((f) => !appliedItems.includes(f.id));
  const completedFiles = MOCK_FILES.filter((f) => appliedItems.includes(f.id));

  return (
    <div className="min-h-screen bg-slate-900 p-8" data-testid="test-manual-search-page">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-white mb-2">
            E2E Test: Manual Search Flow
          </h1>
          <p className="text-slate-400">
            This page provides test fixtures for E2E testing of Story 3-7
            (Manual Metadata Search and Selection)
          </p>
          <div className="mt-4 flex gap-4 text-sm">
            <span className="text-amber-400" data-testid="pending-count">
              Pending: {availableFiles.length}
            </span>
            <span className="text-green-400" data-testid="applied-count">
              Applied: {completedFiles.length}
            </span>
          </div>
        </div>

        {/* Test Scenario Info */}
        <div className="bg-slate-800/50 rounded-lg p-4 mb-8 text-sm">
          <h2 className="text-white font-medium mb-2">Test Scenarios:</h2>
          <ul className="text-slate-400 space-y-1 list-disc list-inside">
            <li><code>test-movie-001</code>: Movie with parsed info (Fight Club 1999)</li>
            <li><code>test-series-001</code>: TV Show with parsed info (Breaking Bad S01E01)</li>
            <li><code>test-unknown-001</code>: File with no parsed info</li>
            <li><code>test-anime-001</code>: Anime with partial parse (Demon Slayer)</li>
          </ul>
        </div>

        {/* Parse Failure Cards Grid */}
        {availableFiles.length > 0 ? (
          <div
            className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4"
            data-testid="parse-failure-grid"
          >
            {availableFiles.map((file) => (
              <ParseFailureCard
                key={file.id}
                file={file}
                onMetadataApplied={() => handleMetadataApplied(file.id)}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-12" data-testid="all-applied-message">
            <p className="text-green-400 text-lg font-medium">
              All metadata applied successfully!
            </p>
            <button
              onClick={() => setAppliedItems([])}
              className="mt-4 px-4 py-2 bg-slate-700 text-white rounded-lg hover:bg-slate-600 transition-colors"
              data-testid="reset-test-data"
            >
              Reset Test Data
            </button>
          </div>
        )}

        {/* Completed Items */}
        {completedFiles.length > 0 && availableFiles.length > 0 && (
          <div className="mt-12">
            <h2 className="text-lg font-medium text-white mb-4">
              Completed ({completedFiles.length})
            </h2>
            <div className="flex flex-wrap gap-2" data-testid="completed-items">
              {completedFiles.map((file) => (
                <span
                  key={file.id}
                  className="px-3 py-1 bg-green-900/30 text-green-400 rounded-full text-sm"
                  data-testid={`completed-${file.id}`}
                >
                  {file.parsedInfo?.title || file.filename}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default TestManualSearchPage;
