/**
 * bugfix-custom-posters-served-and-not-cache — a poster the user uploads is
 * served, shown, and survives every cache clear.
 *
 * @tags @e2e @metadata-editor @custom-poster
 *
 * Before: the upload stored `/posters/<id>.jpg`, which nothing served and the
 * frontend glued onto the TMDb CDN (always broken), and 清除快取 deleted the file
 * because data/posters was counted as an "image cache".
 */
import { faker } from '@faker-js/faker';
import { test, expect } from '../support/fixtures';
import { getValidJpegBuffer } from '../support/fixtures/test-images';

const API_BASE_URL = process.env.API_URL || 'http://localhost:8080/api/v1';

test.describe('Custom poster @metadata-editor @custom-poster', () => {
  let movieId = '';

  test.beforeEach(async ({ api }) => {
    const res = await api.createMovie({
      title: `自訂海報測試 ${Date.now()}`,
      release_date: faker.date.past({ years: 10 }).toISOString().split('T')[0],
      original_title: faker.lorem.words(3),
      genres: ['劇情'],
      overview: faker.lorem.paragraph(),
    });
    expect(res.success).toBe(true);
    movieId = res.data!.id;
  });

  test.afterEach(async ({ api }) => {
    if (movieId) await api.deleteMovie(movieId);
    movieId = '';
  });

  test('[P0] an uploaded poster is served, shown on the detail page, and survives every cache clear', async ({
    api,
    page,
    request,
  }) => {
    const up = await api.uploadPoster(movieId, getValidJpegBuffer(), 'poster.jpg');
    expect(up.success).toBe(true);
    const posterPath = up.data!.poster_url as string;
    expect(posterPath).toBe(`/posters/${movieId}.jpg`);

    // Served by the API.
    const served = await request.get(`${API_BASE_URL}${posterPath}`);
    expect(served.status()).toBe(200);
    expect(served.headers()['content-type']).toContain('image/jpeg');

    // Shown on the detail page — actually decoded, not a broken image.
    await page.goto(`/media/movie/${movieId}`);
    const img = page.locator(`img[src$="/api/v1${posterPath}"]`).first();
    await expect(img).toBeVisible({ timeout: 15000 });
    await expect
      .poll(() => img.evaluate((el: HTMLImageElement) => el.naturalWidth))
      .toBeGreaterThan(0);

    // Every cache clear there is.
    expect(
      (await request.delete(`${API_BASE_URL}/settings/cache?older_than_days=30`)).status()
    ).toBe(200);
    expect((await request.delete(`${API_BASE_URL}/settings/cache`)).status()).toBe(200);
    expect((await request.delete(`${API_BASE_URL}/settings/cache/image`)).status()).toBe(400);

    // Still there.
    expect((await request.get(`${API_BASE_URL}${posterPath}`)).status()).toBe(200);
    await page.reload();
    await expect
      .poll(() => img.evaluate((el: HTMLImageElement) => el.naturalWidth))
      .toBeGreaterThan(0);
  });
});
