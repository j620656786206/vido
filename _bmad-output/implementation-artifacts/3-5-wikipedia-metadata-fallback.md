# Story 3.5: Wikipedia Metadata Fallback

Status: ready-for-dev

## Story

As a **media collector with obscure content**,
I want **Wikipedia as a third fallback source**,
So that **even rare titles can have basic metadata**.

## Acceptance Criteria

1. **AC1: Search zh.wikipedia.org**
   - Given TMDb and Douban both fail
   - When Wikipedia fallback is triggered
   - Then it searches zh.wikipedia.org for the title
   - And extracts data from Infobox templates

2. **AC2: Infobox Data Extraction**
   - Given Wikipedia article has an Infobox
   - When parsing the page
   - Then it extracts: title, director, cast, year, genre, plot summary
   - And handles multiple Infobox template variations (NFR-I15)

3. **AC3: No Poster Handling**
   - Given Wikipedia has no poster images
   - When displaying the media
   - Then a default placeholder icon is shown
   - And the user is notified: "No poster available from Wikipedia"

4. **AC4: API Compliance**
   - Given Wikipedia API is called
   - When making requests
   - Then proper User-Agent is set (NFR-I13)
   - And rate limit of 1 request/second is respected (NFR-I14)

## Tasks / Subtasks

- [ ] Task 1: Create Wikipedia Client (AC: 4)
  - [ ] 1.1: Create `/apps/api/internal/wikipedia/client.go`
  - [ ] 1.2: Implement MediaWiki API client
  - [ ] 1.3: Set proper User-Agent header (NFR-I13)
  - [ ] 1.4: Add rate limiter (1 req/s) (NFR-I14)
  - [ ] 1.5: Write client tests

- [ ] Task 2: Implement Search Functionality (AC: 1)
  - [ ] 2.1: Create `/apps/api/internal/wikipedia/search.go`
  - [ ] 2.2: Use MediaWiki API `action=query&list=search`
  - [ ] 2.3: Search zh.wikipedia.org for Traditional Chinese articles
  - [ ] 2.4: Handle search result ranking
  - [ ] 2.5: Write search tests

- [ ] Task 3: Implement Infobox Parser (AC: 2)
  - [ ] 3.1: Create `/apps/api/internal/wikipedia/infobox.go`
  - [ ] 3.2: Parse `{{Infobox film}}` template
  - [ ] 3.3: Parse `{{Infobox television}}` template
  - [ ] 3.4: Parse `{{Infobox animanga/Header}}` template
  - [ ] 3.5: Handle Chinese Infobox variations (電影資訊框, 電視節目資訊框)
  - [ ] 3.6: Extract: title, director, cast, year, genre
  - [ ] 3.7: Write parser tests with sample wikitext

- [ ] Task 4: Implement Content Extraction (AC: 2)
  - [ ] 4.1: Create `/apps/api/internal/wikipedia/content.go`
  - [ ] 4.2: Use MediaWiki API `action=parse` for page content
  - [ ] 4.3: Extract plain text summary (first paragraph)
  - [ ] 4.4: Handle wiki markup cleanup
  - [ ] 4.5: Write content extraction tests

- [ ] Task 5: Handle Image Extraction (AC: 3)
  - [ ] 5.1: Check for poster/cover image in Infobox
  - [ ] 5.2: Use `action=query&prop=images` to get image list
  - [ ] 5.3: Use `action=query&prop=imageinfo` to get image URL
  - [ ] 5.4: Return placeholder indicator when no image found
  - [ ] 5.5: Write image extraction tests

- [ ] Task 6: Create Wikipedia Provider (AC: 1, 2, 3, 4)
  - [ ] 6.1: Update `/apps/api/internal/metadata/wikipedia_provider.go`
  - [ ] 6.2: Replace stub with actual implementation
  - [ ] 6.3: Map Wikipedia data to `MetadataItem` format
  - [ ] 6.4: Integrate with circuit breaker from Story 3.3
  - [ ] 6.5: Write provider tests

- [ ] Task 7: Caching Layer (AC: 4)
  - [ ] 7.1: Create `wikipedia_cache` table
  - [ ] 7.2: Cache successful fetches for 7 days
  - [ ] 7.3: Reduce API calls to Wikipedia
  - [ ] 7.4: Write cache tests

## Dev Notes

### Architecture Requirements

**FR18: Retrieve metadata from Wikipedia when TMDb and Douban fail**
- Uses MediaWiki API for structured data access

**NFR-I13: Wikipedia proper User-Agent header**
```
User-Agent: Vido/1.0 (https://github.com/vido/vido; contact@example.com) Go-http-client/1.1
```

**NFR-I14: Wikipedia rate limit (1 req/s)**
- Must respect MediaWiki API rate limits

**NFR-I15: Wikipedia Infobox template handling**
- Support multiple Infobox variations

**NFR-P16: Wikipedia fallback retrieval <3s/query**

### MediaWiki API Endpoints

**Base URL:**
```
https://zh.wikipedia.org/w/api.php
```

**Search API:**
```
GET /w/api.php?action=query&list=search&srsearch={query}&format=json&utf8=1
```

**Parse API (get page content):**
```
GET /w/api.php?action=parse&page={title}&format=json&prop=wikitext|text
```

**Query API (get Infobox data):**
```
GET /w/api.php?action=query&titles={title}&prop=revisions&rvprop=content&format=json
```

**Image Info API:**
```
GET /w/api.php?action=query&titles=File:{filename}&prop=imageinfo&iiprop=url&format=json
```

### Infobox Template Variations

**Film Infobox (電影):**
```wikitext
{{Infobox film
| name           = 寄生上流
| original_name  = 기생충
| image          = Parasite (2019 film).png
| director       = [[奉俊昊]]
| producer       = 奉俊昊、郭信愛
| writer         = 奉俊昊、韓進元
| starring       = [[宋康昊]]、[[李善均]]
| music          = 鄭在日
| country        = {{KOR}}
| language       = 韓語
| released       = {{Film date|2019|5|21|坎城影展}}
| runtime        = 132分鐘
}}
```

**Television Infobox (電視節目):**
```wikitext
{{Infobox television
| show_name      = 魷魚遊戲
| image          =
| genre          = 驚悚、生存
| creator        = [[黃東赫]]
| starring       = [[李政宰]]、[[朴海秀]]
| country        = {{KOR}}
| language       = 韓語
| num_seasons    = 1
| num_episodes   = 9
| first_aired    = {{Start date|2021|9|17}}
}}
```

**Anime Infobox (動畫):**
```wikitext
{{Infobox animanga/Header
| name           = 鬼滅之刃
| image          =
| genre          = 動作、黑暗奇幻
}}
{{Infobox animanga/Anime
| director       = 外崎春雄
| studio         = [[ufotable]]
| first          = 2019年4月6日
| last           = 2019年9月28日
}}
```

### Infobox Field Mapping

| Infobox Field | MetadataItem Field | Notes |
|---------------|-------------------|-------|
| `name` / `show_name` | `Title` | Primary title |
| `original_name` | `OriginalTitle` | Original language title |
| `director` / `creator` | `Director` | May contain wiki links |
| `starring` | `Cast` | Array of actors |
| `released` / `first_aired` / `first` | `Year` | Extract year only |
| `genre` | `Genres` | Array of genres |
| `image` | `PosterURL` | If available |
| First paragraph | `Overview` | Plot summary |

### Wiki Markup Cleanup

```go
// Remove wiki links: [[Link|Text]] → Text
func cleanWikiLinks(text string) string {
    re := regexp.MustCompile(`\[\[(?:[^|\]]*\|)?([^\]]+)\]\]`)
    return re.ReplaceAllString(text, "$1")
}

// Remove templates: {{template}} → ""
func cleanTemplates(text string) string {
    // Nested template handling required
}

// Remove HTML tags
func cleanHTML(text string) string {
    re := regexp.MustCompile(`<[^>]+>`)
    return re.ReplaceAllString(text, "")
}
```

### Project Structure Notes

**New Directory to Create:**
```
/apps/api/internal/wikipedia/
├── client.go           # MediaWiki API client
├── client_test.go
├── search.go           # Search functionality
├── search_test.go
├── infobox.go          # Infobox parser
├── infobox_test.go
├── content.go          # Content extraction
├── content_test.go
├── types.go            # Wikipedia-specific types
└── testdata/           # Sample wikitext for tests
    ├── film_infobox.txt
    ├── tv_infobox.txt
    └── anime_infobox.txt
```

**Files to Modify:**
- `/apps/api/internal/metadata/wikipedia_provider.go` - Replace stub
- `/apps/api/internal/config/config.go` - Add Wikipedia settings

### Database Migration

**Migration file: `XXX_create_wikipedia_cache_table.sql`**
```sql
CREATE TABLE IF NOT EXISTS wikipedia_cache (
    id TEXT PRIMARY KEY,
    page_title TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    original_title TEXT,
    year INTEGER,
    director TEXT,
    cast_json TEXT,
    genres_json TEXT,
    summary TEXT,
    image_url TEXT,
    fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_wikipedia_cache_page_title ON wikipedia_cache(page_title);
CREATE INDEX idx_wikipedia_cache_title ON wikipedia_cache(title);
CREATE INDEX idx_wikipedia_cache_expires_at ON wikipedia_cache(expires_at);
```

### Interface Implementation

```go
// WikipediaProvider implements MetadataProvider
type WikipediaProvider struct {
    client *WikipediaClient
    cache  *WikipediaCache
    logger *slog.Logger
}

func (p *WikipediaProvider) Name() string {
    return "wikipedia"
}

func (p *WikipediaProvider) Source() models.MetadataSource {
    return models.MetadataSourceWikipedia
}

func (p *WikipediaProvider) Search(ctx context.Context, req *SearchRequest) (*SearchResult, error) {
    // 1. Check cache first
    if cached := p.cache.Get(req.Query); cached != nil {
        return cached, nil
    }

    // 2. Search Wikipedia
    searchResults, err := p.client.Search(ctx, req.Query)
    if err != nil {
        return nil, err
    }

    if len(searchResults) == 0 {
        return &SearchResult{Items: []MetadataItem{}, Source: p.Source()}, nil
    }

    // 3. Get page content and parse Infobox
    pageTitle := searchResults[0].Title
    content, err := p.client.GetPageContent(ctx, pageTitle)
    if err != nil {
        return nil, err
    }

    // 4. Parse Infobox
    infobox, err := p.parseInfobox(content.Wikitext)
    if err != nil {
        // Infobox parsing failed, but we still have basic info
        slog.Warn("Infobox parsing failed", "page", pageTitle, "error", err)
    }

    // 5. Get image URL if available
    var imageURL string
    if infobox != nil && infobox.Image != "" {
        imageURL, _ = p.client.GetImageURL(ctx, infobox.Image)
    }

    // 6. Build metadata item
    item := p.buildMetadataItem(pageTitle, infobox, content.Extract, imageURL)

    // 7. Cache result
    p.cache.Set(req.Query, item, 7*24*time.Hour)

    return &SearchResult{
        Items:  []MetadataItem{item},
        Source: p.Source(),
    }, nil
}
```

### Testing Strategy

**Unit Tests:**
1. MediaWiki API response parsing
2. Infobox template parsing (multiple variations)
3. Wiki markup cleanup
4. Rate limiter tests

**Integration Tests:**
1. Full search → parse flow (with mocks)
2. Cache hit/miss scenarios
3. No image handling

**Test Data:**
- Save sample wikitext in `testdata/`
- Mock API responses

**Coverage Targets:**
- Wikipedia package: ≥80%

### Error Codes

Following project-context.md Rule 7:
- `WIKIPEDIA_NOT_FOUND` - No article found
- `WIKIPEDIA_NO_INFOBOX` - Article has no Infobox
- `WIKIPEDIA_PARSE_ERROR` - Infobox parsing failed
- `WIKIPEDIA_RATE_LIMITED` - Rate limit exceeded
- `WIKIPEDIA_TIMEOUT` - Request timeout

### Dependencies

**Story Dependencies:**
- Story 3.3 (Fallback Chain) - Provides `MetadataProvider` interface

### User-Agent Requirement

Per MediaWiki API guidelines:
```
User-Agent: ApplicationName/Version (Contact; Description)
```

Example:
```
Vido/1.0 (https://github.com/vido; alexyu@example.com) Go-http-client/1.1
```

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.5]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR18]
- [Source: _bmad-output/implementation-artifacts/3-3-multi-source-metadata-fallback-chain.md]
- [MediaWiki API Documentation](https://www.mediawiki.org/wiki/API:Main_page)
- [project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.3 (Fallback Chain):**
- `MetadataProvider` interface defined
- `SearchRequest` and `SearchResult` types available
- Circuit breaker ready for integration
- Stub provider exists at `/apps/api/internal/metadata/wikipedia_provider.go`

**From Story 3.4 (Douban Scraper):**
- Rate limiting pattern established
- Cache table pattern can be reused

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
