import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { glossaryService } from './glossaryService';

const mockFetch = vi.fn();
global.fetch = mockFetch;

const wireTerm = {
  id: 't1',
  media_id: '42',
  term_src: 'Demogorgon',
  term_zh: '魔王獸',
  language: 'zh-Hant',
  source: 'subtitle',
  confirmed: false,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
};

const camelTerm = {
  id: 't1',
  mediaId: '42',
  termSrc: 'Demogorgon',
  termZh: '魔王獸',
  language: 'zh-Hant',
  source: 'subtitle',
  confirmed: false,
  createdAt: '2026-07-01T00:00:00Z',
  updatedAt: '2026-07-01T00:00:00Z',
};

beforeEach(() => {
  mockFetch.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('glossaryService', () => {
  describe('listTerms', () => {
    it('GETs /media/{mediaId}/glossary and snakeToCamels the terms array', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { terms: [wireTerm] } }),
      });

      const result = await glossaryService.listTerms('42');

      expect(result).toEqual([camelTerm]);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/media/42/glossary'),
        undefined
      );
    });

    it('returns [] when the backend sends an empty terms list', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { terms: [] } }),
      });

      const result = await glossaryService.listTerms('42');
      expect(result).toEqual([]);
    });

    it('throws the envelope error message on failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'DB_NOT_FOUND', message: '找不到媒體' },
          }),
      });

      await expect(glossaryService.listTerms('42')).rejects.toThrow('找不到媒體');
    });
  });

  describe('addTerm', () => {
    it('POSTs a camelToSnake body and returns the created term camelCased', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        status: 201,
        json: () => Promise.resolve({ success: true, data: { ...wireTerm, source: 'manual' } }),
      });

      const result = await glossaryService.addTerm('42', {
        termSrc: 'Demogorgon',
        termZh: '魔王獸',
        source: 'manual',
        confirmed: true,
      });

      expect(result).toEqual({ ...camelTerm, source: 'manual' });
      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/media/42/glossary');
      expect(options.method).toBe('POST');
      expect(JSON.parse(options.body)).toEqual({
        term_src: 'Demogorgon',
        term_zh: '魔王獸',
        source: 'manual',
        confirmed: true,
      });
    });
  });

  describe('confirmAll', () => {
    it('POSTs /confirm-all and returns the confirmed count', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { confirmed: 6 } }),
      });

      const result = await glossaryService.confirmAll('42');

      expect(result).toEqual({ confirmed: 6 });
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/media/42/glossary/confirm-all'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('editTerm', () => {
    it('PUTs {term_zh, confirmed} and resolves void on 204 (no body parse)', async () => {
      mockFetch.mockResolvedValue({ ok: true, status: 204 });

      await expect(
        glossaryService.editTerm('42', 't1', { termZh: '魔神', confirmed: true })
      ).resolves.toBeUndefined();

      const [url, options] = mockFetch.mock.calls[0];
      expect(url).toContain('/media/42/glossary/t1');
      expect(options.method).toBe('PUT');
      expect(JSON.parse(options.body)).toEqual({ term_zh: '魔神', confirmed: true });
    });

    it('throws on 404 unknown termId', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'DB_NOT_FOUND', message: 'Glossary term not found' },
          }),
      });

      await expect(
        glossaryService.editTerm('42', 'missing', { termZh: 'x', confirmed: false })
      ).rejects.toThrow('Glossary term not found');
    });
  });

  describe('confirmTerm', () => {
    it('POSTs /{termId}/confirm and resolves void on 204', async () => {
      mockFetch.mockResolvedValue({ ok: true, status: 204 });

      await expect(glossaryService.confirmTerm('42', 't1')).resolves.toBeUndefined();
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/media/42/glossary/t1/confirm'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('deleteTerm', () => {
    it('DELETEs /{termId} and resolves void on 204', async () => {
      mockFetch.mockResolvedValue({ ok: true, status: 204 });

      await expect(glossaryService.deleteTerm('42', 't1')).resolves.toBeUndefined();
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/media/42/glossary/t1'),
        expect.objectContaining({ method: 'DELETE' })
      );
    });
  });
});
