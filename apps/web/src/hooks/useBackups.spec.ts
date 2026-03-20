import { describe, it, expect } from 'vitest';
import { backupKeys } from './useBackups';

describe('backupKeys', () => {
  it('[P2] returns correct query key hierarchy', () => {
    expect(backupKeys.all).toEqual(['backups']);
    expect(backupKeys.list()).toEqual(['backups', 'list']);
    expect(backupKeys.detail('b1')).toEqual(['backups', 'detail', 'b1']);
  });

  it('[P2] detail key extends all key', () => {
    const allKey = backupKeys.all;
    const detailKey = backupKeys.detail('b1');
    expect(detailKey.slice(0, allKey.length)).toEqual([...allKey]);
  });
});
