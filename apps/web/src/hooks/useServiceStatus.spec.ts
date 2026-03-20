import { describe, it, expect } from 'vitest';
import { serviceStatusKeys } from './useServiceStatus';

describe('serviceStatusKeys', () => {
  it('[P2] returns correct query key hierarchy', () => {
    // GIVEN: serviceStatusKeys factory
    // WHEN: Accessing key levels
    // THEN: Keys follow correct hierarchy
    expect(serviceStatusKeys.all).toEqual(['serviceStatus']);
    expect(serviceStatusKeys.list()).toEqual(['serviceStatus', 'list']);
  });

  it('[P2] list key extends all key', () => {
    // GIVEN: all and list keys
    const allKey = serviceStatusKeys.all;
    const listKey = serviceStatusKeys.list();

    // THEN: list key starts with all key
    expect(listKey.slice(0, allKey.length)).toEqual([...allKey]);
  });
});
