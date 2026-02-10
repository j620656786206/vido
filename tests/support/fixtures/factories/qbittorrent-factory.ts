/**
 * qBittorrent Factory for Test Data Generation
 *
 * Creates qBittorrent configuration data for testing.
 * Auto-cleanup handled by fixture teardown.
 *
 * Pattern: Factory with overrides
 * @see tests/support/fixtures/factories/settings-factory.ts
 */

import { faker } from '@faker-js/faker';

// =============================================================================
// Types
// =============================================================================

export interface QBConfigData {
  host: string;
  username: string;
  password: string;
  basePath: string;
}

export type PartialQBConfigData = Partial<QBConfigData>;

export interface QBConfigResponseData {
  host: string;
  username: string;
  basePath: string;
  configured: boolean;
}

export interface QBVersionInfoData {
  appVersion: string;
  apiVersion: string;
}

// =============================================================================
// Factory Implementation
// =============================================================================

let configCounter = 0;

/**
 * Generate unique qBittorrent config data with optional overrides
 *
 * @example
 * const config = createQBConfigData();
 * const custom = createQBConfigData({ host: 'http://nas:8080', basePath: '/qbt' });
 */
export function createQBConfigData(overrides: PartialQBConfigData = {}): QBConfigData {
  configCounter++;
  const port = faker.number.int({ min: 8000, max: 9999 });

  return {
    host: `http://${faker.internet.ip()}:${port}`,
    username: faker.internet.username(),
    password: faker.internet.password({ length: 12 }),
    basePath: '',
    ...overrides,
  };
}

/**
 * Create config with reverse proxy base path (AC4)
 */
export function createQBReverseProxyConfig(overrides: PartialQBConfigData = {}): QBConfigData {
  return createQBConfigData({
    basePath: '/qbittorrent',
    ...overrides,
  });
}

/**
 * Reset the counter (useful between test files)
 */
export function resetQBFactory(): void {
  configCounter = 0;
}

// =============================================================================
// Preset Configurations for Common Scenarios
// =============================================================================

export const presetQBConfigs = {
  /** Standard local connection */
  local: createQBConfigData({
    host: 'http://localhost:8080',
    username: 'admin',
    password: 'adminadmin',
    basePath: '',
  }),

  /** NAS connection */
  nas: createQBConfigData({
    host: 'http://192.168.1.100:8080',
    username: 'admin',
    password: 'nas-password-123',
    basePath: '',
  }),

  /** Reverse proxy (AC4) */
  reverseProxy: createQBConfigData({
    host: 'https://nas.example.com',
    username: 'admin',
    password: 'proxy-password-456',
    basePath: '/qbittorrent',
  }),

  /** HTTPS connection */
  https: createQBConfigData({
    host: 'https://192.168.1.100:8443',
    username: 'admin',
    password: 'secure-password-789',
    basePath: '',
  }),
} as const;
