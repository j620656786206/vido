import { describe, it, expect } from 'vitest';
import { formatSpeed, formatSize, formatETA, formatProgress, formatDate } from './formatters';

describe('formatSpeed', () => {
  it('formats zero', () => {
    expect(formatSpeed(0)).toBe('0 B/s');
  });

  it('formats bytes per second', () => {
    expect(formatSpeed(500)).toBe('500 B/s');
  });

  it('formats kilobytes per second', () => {
    expect(formatSpeed(1024)).toBe('1.0 KB/s');
    expect(formatSpeed(512000)).toBe('500.0 KB/s');
  });

  it('formats megabytes per second', () => {
    expect(formatSpeed(1048576)).toBe('1.0 MB/s');
    expect(formatSpeed(10485760)).toBe('10.0 MB/s');
  });

  it('formats gigabytes per second', () => {
    expect(formatSpeed(1073741824)).toBe('1.0 GB/s');
  });

  it('handles negative values', () => {
    expect(formatSpeed(-1)).toBe('0 B/s');
  });
});

describe('formatSize', () => {
  it('formats zero', () => {
    expect(formatSize(0)).toBe('0 B');
  });

  it('formats bytes', () => {
    expect(formatSize(500)).toBe('500 B');
  });

  it('formats kilobytes', () => {
    expect(formatSize(1024)).toBe('1.0 KB');
  });

  it('formats megabytes', () => {
    expect(formatSize(1048576)).toBe('1.0 MB');
  });

  it('formats gigabytes', () => {
    expect(formatSize(4294967296)).toBe('4.00 GB');
  });

  it('formats terabytes', () => {
    expect(formatSize(1099511627776)).toBe('1.00 TB');
  });
});

describe('formatETA', () => {
  it('formats infinity for negative values', () => {
    expect(formatETA(-1)).toBe('∞');
  });

  it('formats infinity for 8640000', () => {
    expect(formatETA(8640000)).toBe('∞');
  });

  it('formats zero', () => {
    expect(formatETA(0)).toBe('0s');
  });

  it('formats seconds', () => {
    expect(formatETA(45)).toBe('45s');
  });

  it('formats minutes', () => {
    expect(formatETA(125)).toBe('2m 5s');
  });

  it('formats hours', () => {
    expect(formatETA(3661)).toBe('1h 1m');
  });

  it('formats days', () => {
    expect(formatETA(90000)).toBe('1d 1h');
  });
});

describe('formatProgress', () => {
  it('formats zero progress', () => {
    expect(formatProgress(0)).toBe('0.0%');
  });

  it('formats partial progress', () => {
    expect(formatProgress(0.85)).toBe('85.0%');
  });

  it('formats complete', () => {
    expect(formatProgress(1)).toBe('100.0%');
  });

  it('formats precise progress', () => {
    expect(formatProgress(0.456)).toBe('45.6%');
  });
});

describe('formatDate', () => {
  it('formats ISO date string', () => {
    const result = formatDate('2026-01-15T10:30:00Z');
    expect(result).toBeTruthy();
    expect(typeof result).toBe('string');
  });
});
