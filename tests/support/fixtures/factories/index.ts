/**
 * Factory Exports
 *
 * Central export point for all test data factories.
 * Import from this file to access all factories:
 *
 * @example
 * import { createMovieData, createSeriesData, createSettingData } from '../support/fixtures/factories';
 */

// Movie Factory
export {
  createMovieData,
  createMovieList,
  resetMovieFactory,
  presetMovies,
  type MovieData,
  type PartialMovieData,
} from './movie-factory';

// Parser Factory
export {
  createParseTestCase,
  createBatchTestCases,
  getAllMovieFilenames,
  getAllTVFilenames,
  getAllFansubFilenames,
  movieFilenames,
  tvFilenames,
  fansubFilenames,
  edgeCaseFilenames,
  presetTestCases,
  type ParseTestCase,
} from './parser-factory';

// Series Factory
export {
  createSeriesData,
  createSeriesList,
  resetSeriesFactory,
  presetSeries,
  type SeriesData,
  type PartialSeriesData,
} from './series-factory';

// Settings Factory
export {
  createSettingData,
  createSettingsList,
  resetSettingsFactory,
  createStringSetting,
  createIntSetting,
  createBoolSetting,
  presetSettings,
  type SettingData,
  type PartialSettingData,
} from './settings-factory';

// Metadata Factory
export {
  createManualSearchRequest,
  createManualSearchResultItem,
  createManualSearchResultList,
  createUpdateMetadataRequest,
  resetMetadataFactory,
  presetSearchRequests,
  presetSearchResults,
  presetUpdateRequests,
  type ManualSearchRequestData,
  type ManualSearchResultItemData,
  type UpdateMetadataRequestData,
  type PartialManualSearchRequest,
  type PartialManualSearchResultItem,
  type PartialUpdateMetadataRequest,
} from './metadata-factory';

// qBittorrent Factory
export {
  createQBConfigData,
  createQBReverseProxyConfig,
  presetQBConfigs,
  type QBConfigData,
  type QBConfigResponseData,
  type QBVersionInfoData,
  type PartialQBConfigData,
} from './qbittorrent-factory';

// Download Factory (Story 4-2)
export {
  createDownloadData,
  createDownloadDetailsData,
  createDownloadList,
  presetDownloads,
  type DownloadData,
  type DownloadDetailsData,
  type PartialDownloadData,
  type PartialDownloadDetailsData,
} from './download-factory';

// Learning Factory
export {
  createLearnPatternRequest,
  createMoviePatternRequest,
  resetLearningFactory,
  presetPatternRequests,
  type CreatePatternRequestData,
  type LearnedPatternData,
  type PatternStatsData,
  type PatternListResponseData,
  type PartialCreatePatternRequest,
} from './learning-factory';
