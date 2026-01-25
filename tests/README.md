# Vido E2E Testing

End-to-end testing infrastructure using [Playwright](https://playwright.dev/).

## Quick Start

```bash
# Install Playwright and browsers
npm install -D @playwright/test
npx playwright install

# Run all tests
npm run test:e2e

# Run tests with UI mode (recommended for development)
npm run test:e2e:ui

# Generate tests by recording actions
npm run test:e2e:codegen
```

## Available Scripts

| Script                     | Description                    |
| -------------------------- | ------------------------------ |
| `npm run test:e2e`         | Run all E2E tests (headless)   |
| `npm run test:e2e:ui`      | Open Playwright UI mode        |
| `npm run test:e2e:headed`  | Run tests with visible browser |
| `npm run test:e2e:debug`   | Run tests in debug mode        |
| `npm run test:e2e:codegen` | Open test recorder             |
| `npm run test:e2e:report`  | View HTML test report          |

## Project Structure

```
tests/
├── e2e/                          # Test files
│   ├── home.spec.ts              # Home page tests
│   ├── search.spec.ts            # Media search E2E tests
│   ├── media-detail.spec.ts      # Media detail page E2E tests
│   ├── api.spec.ts               # General API tests
│   ├── health.api.spec.ts        # Health check API tests
│   ├── movies.api.spec.ts        # Movies CRUD API tests
│   ├── series.api.spec.ts        # TV Series CRUD API tests
│   ├── parser.api.spec.ts        # Filename Parser API tests
│   └── settings.api.spec.ts      # Settings API tests
│
├── support/                      # Test infrastructure
│   ├── fixtures/                 # Playwright fixtures
│   │   ├── index.ts              # Main fixture exports
│   │   └── factories/            # Test data factories
│   │       ├── movie-factory.ts  # Movie data generator
│   │       └── parser-factory.ts # Parser test data generator
│   │
│   ├── helpers/                  # Utility functions
│   │   └── api-helpers.ts        # API request helpers
│   │
│   └── page-objects/             # Page Object Models (optional)
│
└── README.md                     # This file
```

## Writing Tests

### Basic Test

```typescript
import { test, expect } from '../support/fixtures';

test('user can search for movies', async ({ page }) => {
  await page.goto('/');

  const searchInput = page.getByPlaceholder(/search|搜尋/i);
  await searchInput.fill('Inception');
  await searchInput.press('Enter');

  await expect(page.getByText('Inception')).toBeVisible();
});
```

### Using API Fixtures

```typescript
import { test, expect } from '../support/fixtures';

test('API returns movie data', async ({ api }) => {
  const response = await api.searchMovies('Inception');

  expect(response.success).toBe(true);
  expect(response.data?.results.length).toBeGreaterThan(0);
});
```

### Using Data Factories

```typescript
import { test, expect } from '../support/fixtures';
import { createMovieData, presetMovies } from '../support/fixtures/factories/movie-factory';

test('displays movie details', async ({ page }) => {
  const movie = createMovieData({ title: 'Test Movie' });
  // Use movie data for assertions or mocking
});
```

## Test Tags

Use tags for selective test execution:

```typescript
test('critical flow @smoke', async ({ page }) => { ... });
test('edge case @regression', async ({ page }) => { ... });
test('API test @api', async ({ api }) => { ... });
```

Run tagged tests:

```bash
# Run only smoke tests
npx playwright test --grep @smoke

# Run everything except slow tests
npx playwright test --grep-invert @slow

# Run API tests only
npx playwright test --grep @api
```

## Priority Tags

All tests use priority tags for selective execution based on risk level:

| Tag    | Description                       | When to Run    |
| ------ | --------------------------------- | -------------- |
| `[P0]` | Critical paths, must always work  | Every commit   |
| `[P1]` | High priority, important features | PR to main     |
| `[P2]` | Medium priority, edge cases       | Nightly builds |
| `[P3]` | Low priority, nice-to-have        | On-demand      |

### Running by Priority

```bash
# Run critical tests only (P0)
npx playwright test --grep '\[P0\]'

# Run P0 + P1 tests (pre-merge)
npx playwright test --grep '\[P0\]|\[P1\]'

# Run all except low priority
npx playwright test --grep-invert '\[P3\]'
```

### Priority Tagging Convention

Include priority in test name:

```typescript
test('[P0] should login with valid credentials', async ({ page }) => { ... });
test('[P1] should display error for invalid input', async ({ page }) => { ... });
test('[P2] should handle edge case scenario', async ({ page }) => { ... });
```

## Debugging

### Trace Viewer

When a test fails, Playwright captures a trace. View it with:

```bash
npx playwright show-trace test-results/path-to-trace.zip
```

### Debug Mode

```bash
# Step through test with Playwright Inspector
npm run test:e2e:debug

# Or add this to a specific test
test.only('debug this test', async ({ page }) => {
  await page.pause(); // Breakpoint
  // ...
});
```

### Codegen (Record Tests)

```bash
# Record your actions and generate test code
npm run test:e2e:codegen http://localhost:5173
```

## Configuration

### Environment Variables

Set in `.env` or environment:

| Variable   | Default                        | Description                             |
| ---------- | ------------------------------ | --------------------------------------- |
| `TEST_ENV` | `local`                        | Environment: local, staging, production |
| `BASE_URL` | `http://localhost:5173`        | Frontend URL                            |
| `API_URL`  | `http://localhost:8080/api/v1` | Backend API URL                         |

### Timeout Standards

| Type       | Duration | Use Case          |
| ---------- | -------- | ----------------- |
| Action     | 15s      | click, fill, etc. |
| Navigation | 30s      | page.goto, reload |
| Expect     | 10s      | all assertions    |
| Test       | 60s      | entire test       |

### Browser Projects

Tests run on multiple browsers:

```bash
# Run specific browser
npx playwright test --project=chromium
npx playwright test --project=firefox
npx playwright test --project=webkit

# Run mobile emulation
npx playwright test --project=mobile-chrome
npx playwright test --project=mobile-safari
```

## CI Integration

### GitHub Actions

Add to `.github/workflows/e2e.yml`:

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version-file: '.nvmrc'

      - name: Install dependencies
        run: npm ci

      - name: Install Playwright browsers
        run: npx playwright install --with-deps

      - name: Run E2E tests
        run: npm run test:e2e
        env:
          CI: true
          TEST_ENV: staging

      - name: Upload test results
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: playwright-report/
          retention-days: 30
```

## Best Practices

### Selectors

Prefer `data-testid` attributes:

```typescript
// ✅ Good - stable selector
await page.click('[data-testid="submit-button"]');

// ❌ Avoid - brittle selectors
await page.click('.btn.btn-primary.submit');
await page.click('button:nth-child(3)');
```

### Test Isolation

Each test should be independent:

```typescript
// ✅ Good - each test sets up its own state
test('user can login', async ({ page, api }) => {
  // Setup via API (fast!)
  await api.createUser({ email: 'test@example.com' });

  // Test
  await page.goto('/login');
  // ...
});

// ❌ Avoid - depending on previous test state
test('user sees dashboard after login', async ({ page }) => {
  // Assumes user is already logged in from previous test
});
```

### API for Setup

Use API calls instead of UI actions for test setup:

```typescript
// ✅ Fast - API setup
test('user can view profile', async ({ page, api }) => {
  const user = await api.createUser({ name: 'Test User' });
  await api.login(user.email, user.password);
  await page.goto('/profile');
});

// ❌ Slow - UI setup
test('user can view profile', async ({ page }) => {
  await page.goto('/register');
  await page.fill('#email', 'test@example.com');
  // ... many more UI actions
});
```

## Resources

- [Playwright Documentation](https://playwright.dev/docs/intro)
- [Playwright Best Practices](https://playwright.dev/docs/best-practices)
- [Trace Viewer](https://playwright.dev/docs/trace-viewer)
- [Test Generator](https://playwright.dev/docs/codegen)
