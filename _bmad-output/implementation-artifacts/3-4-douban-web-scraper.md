# Story 3.4: Douban Web Scraper

Status: ready-for-dev

## Story

As a **media collector with Asian content**,
I want **Douban as a fallback metadata source**,
So that **Chinese movies and shows not on TMDb can still be identified**.

## Acceptance Criteria

1. **AC1: Search and Extract Metadata**
   - Given TMDb search fails
   - When Douban scraper is triggered
   - Then it searches Douban for the title
   - And extracts: Chinese title, year, director, cast, rating, poster URL

2. **AC2: Anti-Scraping Handling**
   - Given Douban anti-scraping measures are encountered
   - When the scraper detects blocking
   - Then it implements exponential backoff
   - And falls back to Wikipedia if blocked

3. **AC3: Traditional Chinese Priority**
   - Given Douban returns results
   - When displaying metadata
   - Then Traditional Chinese is prioritized
   - And the source is clearly labeled

4. **AC4: Polite Scraping**
   - Given web scraping is performed
   - When making requests to Douban
   - Then respect robots.txt
   - And implement proper rate limiting (1 req/2s minimum)

## Tasks / Subtasks

- [ ] Task 1: Create Douban Client (AC: 1, 4)
  - [ ] 1.1: Create `/apps/api/internal/douban/client.go`
  - [ ] 1.2: Implement HTTP client with proper User-Agent
  - [ ] 1.3: Add request rate limiter (1 request per 2 seconds)
  - [ ] 1.4: Implement cookie/session handling for anti-scraping
  - [ ] 1.5: Write client tests with mocked responses

- [ ] Task 2: Implement Search Functionality (AC: 1)
  - [ ] 2.1: Create `/apps/api/internal/douban/search.go`
  - [ ] 2.2: Parse Douban search results page
  - [ ] 2.3: Extract movie/TV show links from search
  - [ ] 2.4: Handle pagination if needed
  - [ ] 2.5: Write search tests

- [ ] Task 3: Implement Detail Page Scraper (AC: 1, 3)
  - [ ] 3.1: Create `/apps/api/internal/douban/scraper.go`
  - [ ] 3.2: Extract Chinese title (繁體 if available, else 簡體)
  - [ ] 3.3: Extract year, director, cast, rating
  - [ ] 3.4: Extract poster URL
  - [ ] 3.5: Extract plot summary (劇情簡介)
  - [ ] 3.6: Write scraper tests with sample HTML

- [ ] Task 4: Implement Anti-Scraping Countermeasures (AC: 2)
  - [ ] 4.1: Detect blocking (403, CAPTCHA page, rate limit response)
  - [ ] 4.2: Implement exponential backoff (1s → 2s → 4s → 8s → 16s)
  - [ ] 4.3: Add request jitter (random delay 100-500ms)
  - [ ] 4.4: Rotate User-Agent strings
  - [ ] 4.5: Log anti-scraping events for monitoring

- [ ] Task 5: Create Douban Provider (AC: 1, 2, 3)
  - [ ] 5.1: Update `/apps/api/internal/metadata/douban_provider.go`
  - [ ] 5.2: Replace stub with actual implementation
  - [ ] 5.3: Map Douban results to `MetadataItem` format
  - [ ] 5.4: Integrate with circuit breaker from Story 3.3
  - [ ] 5.5: Write provider tests

- [ ] Task 6: Simplified/Traditional Chinese Conversion (AC: 3)
  - [ ] 6.1: Add OpenCC or similar library for S2T conversion
  - [ ] 6.2: Convert Simplified Chinese titles to Traditional
  - [ ] 6.3: Detect and preserve original Traditional Chinese
  - [ ] 6.4: Write conversion tests

- [ ] Task 7: Caching Layer (AC: 1, 4)
  - [ ] 7.1: Create `douban_cache` table for scraped results
  - [ ] 7.2: Cache successful scrapes for 7 days
  - [ ] 7.3: Reduce load on Douban servers
  - [ ] 7.4: Write cache tests

## Dev Notes

### Architecture Requirements

**FR17: Auto-switch to Douban when TMDb fails**
- Web scraping with proper rate limiting
- Respect robots.txt and implement polite scraping

**NFR-R5: Exponential backoff retry (1s → 2s → 4s → 8s)**
- Applied when anti-scraping measures are detected

### Douban Website Structure

**Search URL Pattern:**
```
https://search.douban.com/movie/subject_search?search_text={query}&cat=1002
```

**Movie Detail Page:**
```
https://movie.douban.com/subject/{id}/
```

**Key HTML Selectors:**
```css
/* Title */
#content h1 span[property="v:itemreviewed"]

/* Year */
#content h1 .year

/* Rating */
strong.rating_num

/* Director */
#info span:contains("导演") + span a

/* Cast */
#info span.actor span.attrs a

/* Poster */
#mainpic img

/* Plot Summary */
span[property="v:summary"]
```

### Sample HTML Structure

```html
<div id="content">
  <h1>
    <span property="v:itemreviewed">寄生上流</span>
    <span class="year">(2019)</span>
  </h1>
  <div id="interest_sectl">
    <strong class="rating_num" property="v:average">8.7</strong>
  </div>
  <div id="info">
    <span>导演</span>: <span class="attrs"><a href="...">奉俊昊</a></span>
    <span class="actor">
      <span class="attrs">
        <a href="...">宋康昊</a> / <a href="...">李善均</a>
      </span>
    </span>
  </div>
  <div id="mainpic">
    <img src="https://img.doubanio.com/view/photo/s_ratio_poster/public/p2561439800.jpg" />
  </div>
  <span property="v:summary">基澤一家四口...</span>
</div>
```

### Anti-Scraping Countermeasures

**Douban's Known Protections:**
1. Rate limiting (too many requests → 403)
2. CAPTCHA challenges
3. IP blocking
4. Cookie validation
5. JavaScript rendering requirements (some pages)

**Our Countermeasures:**
```go
type DoubanClient struct {
    httpClient    *http.Client
    rateLimiter   *rate.Limiter  // 1 req per 2s
    userAgents    []string        // Rotate UA
    retryConfig   RetryConfig
    logger        *slog.Logger
}

type RetryConfig struct {
    MaxRetries      int           // 5
    InitialDelay    time.Duration // 1s
    MaxDelay        time.Duration // 16s
    BackoffFactor   float64       // 2.0
    Jitter          time.Duration // 100-500ms random
}
```

### User-Agent Rotation

```go
var userAgents = []string{
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
}
```

### Simplified to Traditional Chinese Conversion

**Using OpenCC (recommended):**
```go
import "github.com/longbridgeapp/opencc"

func (c *DoubanClient) toTraditional(simplified string) (string, error) {
    converter, err := opencc.New("s2twp") // Simplified to Traditional (Taiwan)
    if err != nil {
        return simplified, err
    }
    return converter.Convert(simplified)
}
```

**Conversion Profiles:**
- `s2t`: Simplified to Traditional
- `s2tw`: Simplified to Traditional (Taiwan standard)
- `s2twp`: Simplified to Traditional (Taiwan + phrases)

### Project Structure Notes

**New Directory to Create:**
```
/apps/api/internal/douban/
├── client.go           # HTTP client with rate limiting
├── client_test.go
├── search.go           # Search functionality
├── search_test.go
├── scraper.go          # Detail page scraper
├── scraper_test.go
├── types.go            # Douban-specific types
└── testdata/           # Sample HTML for tests
    ├── search_results.html
    └── movie_detail.html
```

**Files to Modify:**
- `/apps/api/internal/metadata/douban_provider.go` - Replace stub
- `/apps/api/internal/config/config.go` - Add Douban settings

### Database Migration

**Migration file: `XXX_create_douban_cache_table.sql`**
```sql
CREATE TABLE IF NOT EXISTS douban_cache (
    id TEXT PRIMARY KEY,
    douban_id TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    title_traditional TEXT,
    year INTEGER,
    rating REAL,
    director TEXT,
    cast_json TEXT,
    poster_url TEXT,
    summary TEXT,
    scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_douban_cache_douban_id ON douban_cache(douban_id);
CREATE INDEX idx_douban_cache_title ON douban_cache(title);
CREATE INDEX idx_douban_cache_expires_at ON douban_cache(expires_at);
```

### Interface Implementation

```go
// DoubanProvider implements MetadataProvider
type DoubanProvider struct {
    client *DoubanClient
    cache  *DoubanCache
    logger *slog.Logger
}

func (p *DoubanProvider) Name() string {
    return "douban"
}

func (p *DoubanProvider) Source() models.MetadataSource {
    return models.MetadataSourceDouban
}

func (p *DoubanProvider) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
    // 1. Check cache first
    if cached := p.cache.Get(req.Query); cached != nil {
        return cached, nil
    }

    // 2. Search Douban
    searchResults, err := p.client.Search(ctx, req.Query)
    if err != nil {
        return nil, err
    }

    // 3. Scrape first result detail page
    if len(searchResults) > 0 {
        detail, err := p.client.ScrapeDetail(ctx, searchResults[0].ID)
        if err != nil {
            return nil, err
        }

        // 4. Convert to Traditional Chinese
        detail.TitleTraditional = p.toTraditional(detail.Title)

        // 5. Cache result
        p.cache.Set(req.Query, detail, 7*24*time.Hour)

        // 6. Return normalized result
        return p.toSearchResult(detail), nil
    }

    return &SearchResult{Items: []MetadataItem{}, Source: p.Source()}, nil
}
```

### Testing Strategy

**Unit Tests:**
1. HTML parsing tests with sample files
2. Rate limiter tests
3. Retry logic tests
4. S2T conversion tests

**Integration Tests:**
1. Full search → scrape flow (with mocks)
2. Cache hit/miss scenarios
3. Anti-scraping recovery

**Test Data:**
- Save sample Douban HTML pages in `testdata/`
- Mock HTTP responses for testing

**Coverage Targets:**
- Douban package: ≥80%

### Error Codes

Following project-context.md Rule 7:
- `DOUBAN_BLOCKED` - Anti-scraping measures detected
- `DOUBAN_NOT_FOUND` - No results found
- `DOUBAN_PARSE_ERROR` - HTML parsing failed
- `DOUBAN_RATE_LIMITED` - Rate limit exceeded
- `DOUBAN_TIMEOUT` - Request timeout

### Dependencies

**Go Libraries:**
- `github.com/PuerkitoBio/goquery` - HTML parsing
- `github.com/longbridgeapp/opencc` - S2T conversion
- `golang.org/x/time/rate` - Rate limiting

**Story Dependencies:**
- Story 3.3 (Fallback Chain) - Provides `MetadataProvider` interface

### Ethical Scraping Guidelines

1. **Respect robots.txt** - Check and comply
2. **Rate limit** - Max 1 request per 2 seconds
3. **Caching** - Cache results to reduce load
4. **User-Agent** - Use realistic browser UA
5. **No overloading** - Implement circuit breaker
6. **Purpose** - Personal use metadata fetching only

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.4]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR17]
- [Source: _bmad-output/implementation-artifacts/3-3-multi-source-metadata-fallback-chain.md]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.3 (Fallback Chain):**
- `MetadataProvider` interface defined
- `SearchRequest` and `SearchResult` types available
- Circuit breaker ready for integration
- Stub provider exists at `/apps/api/internal/metadata/douban_provider.go`

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
