/**
 * Settings Factory for Test Data Generation
 *
 * Creates settings data for testing the settings API.
 * Auto-cleanup handled by fixture teardown.
 *
 * Pattern: Factory with overrides
 * @see https://playwright.dev/docs/test-fixtures
 */

import { faker } from '@faker-js/faker';

// =============================================================================
// Types
// =============================================================================

export interface SettingData {
  key: string;
  value: string;
  type: 'string' | 'int' | 'bool';
}

export type PartialSettingData = Partial<SettingData>;

// =============================================================================
// Factory Implementation
// =============================================================================

let settingCounter = 0;

/**
 * Generate unique setting data with optional overrides
 *
 * @example
 * const setting = createSettingData();
 * const customSetting = createSettingData({ key: 'theme', value: 'dark', type: 'string' });
 */
export function createSettingData(overrides: PartialSettingData = {}): SettingData {
  settingCounter++;
  const types: ('string' | 'int' | 'bool')[] = ['string', 'int', 'bool'];
  const randomType = faker.helpers.arrayElement(types);

  let value: string;
  switch (randomType) {
    case 'int':
      value = String(faker.number.int({ min: 1, max: 1000 }));
      break;
    case 'bool':
      value = String(faker.datatype.boolean());
      break;
    default:
      value = faker.lorem.word();
  }

  return {
    key: `test_setting_${settingCounter}_${Date.now()}`,
    value,
    type: randomType,
    ...overrides,
  };
}

/**
 * Generate multiple settings for batch testing
 *
 * @example
 * const settings = createSettingsList(5);
 */
export function createSettingsList(count: number, overrides: PartialSettingData = {}): SettingData[] {
  return Array.from({ length: count }, () => createSettingData(overrides));
}

/**
 * Reset the counter (useful between test files)
 */
export function resetSettingsFactory(): void {
  settingCounter = 0;
}

// =============================================================================
// Preset Settings for Common Scenarios
// =============================================================================

export const presetSettings = {
  // Application settings
  theme: createSettingData({
    key: 'theme',
    value: 'dark',
    type: 'string',
  }),

  language: createSettingData({
    key: 'language',
    value: 'zh-TW',
    type: 'string',
  }),

  pageSize: createSettingData({
    key: 'page_size',
    value: '20',
    type: 'int',
  }),

  autoScan: createSettingData({
    key: 'auto_scan',
    value: 'true',
    type: 'bool',
  }),

  // TMDb settings
  tmdbApiKey: createSettingData({
    key: 'tmdb_api_key',
    value: 'test_api_key_123',
    type: 'string',
  }),

  tmdbLanguage: createSettingData({
    key: 'tmdb_language',
    value: 'zh-TW',
    type: 'string',
  }),

  // Cache settings
  cacheEnabled: createSettingData({
    key: 'cache_enabled',
    value: 'true',
    type: 'bool',
  }),

  cacheTtl: createSettingData({
    key: 'cache_ttl',
    value: '86400',
    type: 'int',
  }),
} as const;

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Create a string setting
 */
export function createStringSetting(key: string, value: string): SettingData {
  return createSettingData({ key, value, type: 'string' });
}

/**
 * Create an integer setting
 */
export function createIntSetting(key: string, value: number): SettingData {
  return createSettingData({ key, value: String(value), type: 'int' });
}

/**
 * Create a boolean setting
 */
export function createBoolSetting(key: string, value: boolean): SettingData {
  return createSettingData({ key, value: String(value), type: 'bool' });
}
