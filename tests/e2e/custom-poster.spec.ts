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
    // poster-upload-b AC #2: versioned so a re-upload changes what the page renders.
    expect(posterPath).toMatch(new RegExp(`^/posters/${movieId}\\.jpg\\?v=\\d+$`));

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

  test('[P0] 修改資訊: pick an image — 取消 changes nothing; 儲存 swaps it in, and a second swap shows without reload (poster-upload-b)', async ({
    page,
  }) => {
    await page.goto(`/media/movie/${movieId}`);
    const tile = page.getByTestId('detail-poster-tile');
    await expect(page.getByTestId('action-edit-metadata')).toBeVisible({ timeout: 15000 });
    // A freshly created movie has no poster: the tile is a gradient, no <img>
    // (getAttribute on a missing element would wait out the action timeout).
    const posterSrc = async () =>
      (await tile.locator('img').count()) ? tile.locator('img').getAttribute('src') : null;
    const srcBefore = await posterSrc();

    const openAndPick = async () => {
      await page.getByTestId('action-edit-metadata').click();
      const dialog = page.getByRole('dialog', { name: '修改資訊' });
      await expect(dialog).toBeVisible();
      await dialog.getByTestId('poster-file-input').setInputFiles({
        name: 'poster.jpg',
        mimeType: 'image/jpeg',
        buffer: getValidJpegBuffer(),
      });
      await expect(dialog.getByText('新海報・尚未儲存')).toBeVisible();
      return dialog;
    };

    // 取消 → nothing uploaded, the page is unchanged.
    let dialog = await openAndPick();
    await dialog.getByRole('button', { name: '取消' }).click();
    await expect(dialog).toBeHidden();
    expect(await posterSrc()).toBe(srcBefore);

    // 儲存 → the detail page shows the uploaded poster, decoded.
    dialog = await openAndPick();
    await dialog.getByRole('button', { name: '儲存' }).click();
    await expect(dialog).toBeHidden({ timeout: 15000 });
    const img = tile.locator('img');
    await expect(img).toHaveAttribute(
      'src',
      new RegExp(`/api/v1/posters/${movieId}\\.jpg\\?v=\\d+$`)
    );
    await expect
      .poll(() => img.evaluate((el: HTMLImageElement) => el.naturalWidth))
      .toBeGreaterThan(0);
    const firstSrc = await img.getAttribute('src');

    // Again → a new src, no reload.
    dialog = await openAndPick();
    await dialog.getByRole('button', { name: '儲存' }).click();
    await expect(dialog).toBeHidden({ timeout: 15000 });
    await expect.poll(() => img.getAttribute('src')).not.toBe(firstSrc);
  });

  test('[P1] a series poster sent the way the web app sends it (multipart field) is really saved', async ({
    api,
    request,
  }) => {
    const created = await api.createSeries({
      title: `影集海報測試 ${Date.now()}`,
      first_air_date: faker.date.past({ years: 5 }).toISOString().split('T')[0],
      original_title: faker.lorem.words(3),
      genres: ['劇情'],
      overview: faker.lorem.paragraph(),
    });
    expect(created.success).toBe(true);
    const seriesId = created.data!.id;
    try {
      const res = await request.post(`${API_BASE_URL}/media/${seriesId}/poster`, {
        multipart: {
          mediaType: 'series',
          file: { name: 'p.jpg', mimeType: 'image/jpeg', buffer: getValidJpegBuffer() },
        },
      });
      expect(res.status()).toBe(200);
      const series = await api.getSeries(seriesId);
      const stored =
        (series.data as Record<string, unknown>).poster_path ??
        (series.data as Record<string, unknown>).posterPath;
      expect(stored).toMatch(new RegExp(`^/posters/${seriesId}\\.jpg\\?v=\\d+$`));
    } finally {
      await api.deleteSeries(seriesId);
    }
  });
});
