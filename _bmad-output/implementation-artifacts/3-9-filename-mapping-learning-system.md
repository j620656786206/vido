# Story 3.9: Filename Mapping Learning System

Status: ready-for-dev

## Story

As a **power user**,
I want the **system to learn from my corrections**,
So that **similar filenames are automatically matched in the future**.

## Acceptance Criteria

1. **AC1: Learn Pattern Prompt**
   - Given the user manually corrects a filename match
   - When the correction is saved
   - Then the system asks: "Learn this pattern for future files?"
   - And if confirmed, stores the pattern-to-metadata mapping

2. **AC2: Auto-Apply Learned Patterns**
   - Given a learned pattern exists
   - When a new file matches the pattern
   - Then the system automatically applies the learned mapping
   - And shows: "✓ 已套用你之前的設定" (UX-5)

3. **AC3: Manage Learned Patterns**
   - Given the user views settings
   - When checking learned patterns
   - Then they see: "已記住 15 個自訂規則"
   - And can view, edit, or delete learned patterns

4. **AC4: Fuzzy Pattern Matching**
   - Given a learned pattern like "[Leopard-Raws] Kimetsu no Yaiba"
   - When a new file "[Leopard-Raws] Kimetsu no Yaiba - 27" arrives
   - Then the pattern is recognized as a match
   - And the same metadata is applied

## Tasks / Subtasks

- [ ] Task 1: Create Filename Mappings Table (AC: 1)
  - [ ] 1.1: Create database migration for `filename_mappings` table
  - [ ] 1.2: Define columns: id, pattern, pattern_type, metadata_id, source, created_at
  - [ ] 1.3: Add indexes for pattern lookup
  - [ ] 1.4: Write migration tests

- [ ] Task 2: Create Pattern Extraction Logic (AC: 1, 4)
  - [ ] 2.1: Create `/apps/api/internal/learning/pattern.go`
  - [ ] 2.2: Extract fansub group pattern: `[GroupName]`
  - [ ] 2.3: Extract title pattern (normalize episode/season)
  - [ ] 2.4: Generate regex pattern from filename
  - [ ] 2.5: Write pattern extraction tests

- [ ] Task 3: Create Pattern Matching Engine (AC: 2, 4)
  - [ ] 3.1: Create `/apps/api/internal/learning/matcher.go`
  - [ ] 3.2: Implement exact match lookup
  - [ ] 3.3: Implement fuzzy matching with Levenshtein distance
  - [ ] 3.4: Implement regex pattern matching
  - [ ] 3.5: Return match confidence score
  - [ ] 3.6: Write matcher tests

- [ ] Task 4: Create Learning Repository (AC: 1, 3)
  - [ ] 4.1: Create `/apps/api/internal/repository/learning_repository.go`
  - [ ] 4.2: Implement `Save()` for new patterns
  - [ ] 4.3: Implement `FindByPattern()` for lookup
  - [ ] 4.4: Implement `List()` for management UI
  - [ ] 4.5: Implement `Delete()` for removal
  - [ ] 4.6: Write repository tests

- [ ] Task 5: Create Learning Service (AC: 1, 2, 3)
  - [ ] 5.1: Create `/apps/api/internal/services/learning_service.go`
  - [ ] 5.2: Define `LearningServiceInterface`
  - [ ] 5.3: Implement `LearnFromCorrection()` method
  - [ ] 5.4: Implement `FindMatchingPattern()` method
  - [ ] 5.5: Implement `GetPatternStats()` for UI display
  - [ ] 5.6: Write service tests

- [ ] Task 6: Integrate with Parser (AC: 2)
  - [ ] 6.1: Update parser service to check learned patterns first
  - [ ] 6.2: If pattern matches, skip AI/metadata search
  - [ ] 6.3: Set metadata source to "learned"
  - [ ] 6.4: Emit "pattern applied" event for UI feedback
  - [ ] 6.5: Write integration tests

- [ ] Task 7: Create Learn Pattern API (AC: 1)
  - [ ] 7.1: Create `POST /api/v1/learning/patterns` endpoint
  - [ ] 7.2: Accept filename, metadata ID, and confirmation
  - [ ] 7.3: Extract and store pattern
  - [ ] 7.4: Write handler tests

- [ ] Task 8: Create Patterns Management API (AC: 3)
  - [ ] 8.1: Create `GET /api/v1/learning/patterns` endpoint
  - [ ] 8.2: Create `DELETE /api/v1/learning/patterns/{id}` endpoint
  - [ ] 8.3: Include pattern stats in response
  - [ ] 8.4: Write handler tests

- [ ] Task 9: Create Learn Pattern UI (AC: 1, 2)
  - [ ] 9.1: Create `LearnPatternPrompt.tsx` component
  - [ ] 9.2: Show after manual metadata selection
  - [ ] 9.3: Display extracted pattern preview
  - [ ] 9.4: Add confirm/skip buttons
  - [ ] 9.5: Show success toast with UX-5 feedback

- [ ] Task 10: Create Patterns Management UI (AC: 3)
  - [ ] 10.1: Create `LearnedPatternsSettings.tsx` component
  - [ ] 10.2: Display pattern list with metadata preview
  - [ ] 10.3: Add delete button for each pattern
  - [ ] 10.4: Show "已記住 N 個自訂規則" count
  - [ ] 10.5: Write component tests

## Dev Notes

### Architecture Requirements

**FR24: Learn from user corrections**
- Pattern matching with fuzzy matching support

**UX-5: Learning system feedback**
- Show when system applies learned rules: "✓ 已套用你之前的設定"

### Database Schema

**Migration file: `XXX_create_filename_mappings_table.sql`**
```sql
CREATE TABLE IF NOT EXISTS filename_mappings (
    id TEXT PRIMARY KEY,
    pattern TEXT NOT NULL,              -- Extracted pattern
    pattern_type TEXT NOT NULL,         -- 'exact', 'regex', 'fuzzy'
    pattern_regex TEXT,                 -- Generated regex (if applicable)
    fansub_group TEXT,                  -- Extracted fansub group
    title_pattern TEXT,                 -- Title without episode numbers
    metadata_type TEXT NOT NULL,        -- 'movie' or 'series'
    metadata_id TEXT NOT NULL,          -- Reference to movie/series
    tmdb_id INTEGER,                    -- TMDb ID for quick lookup
    confidence REAL DEFAULT 1.0,        -- Match confidence threshold
    use_count INTEGER DEFAULT 0,        -- Times pattern was applied
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    FOREIGN KEY (metadata_id) REFERENCES movies(id) ON DELETE CASCADE
);

CREATE INDEX idx_filename_mappings_pattern ON filename_mappings(pattern);
CREATE INDEX idx_filename_mappings_fansub_group ON filename_mappings(fansub_group);
CREATE INDEX idx_filename_mappings_title_pattern ON filename_mappings(title_pattern);
```

### Pattern Extraction Logic

```go
// PatternExtractor extracts reusable patterns from filenames
type PatternExtractor struct{}

type ExtractedPattern struct {
    OriginalFilename string
    FansubGroup      string   // e.g., "Leopard-Raws"
    TitlePattern     string   // e.g., "Kimetsu no Yaiba"
    Regex            string   // Generated regex
    PatternType      string   // "fansub", "standard", "custom"
}

func (e *PatternExtractor) Extract(filename string) (*ExtractedPattern, error) {
    pattern := &ExtractedPattern{
        OriginalFilename: filename,
    }

    // 1. Extract fansub group: [GroupName] or 【組名】
    fansubRegex := regexp.MustCompile(`[\[【]([^\]】]+)[\]】]`)
    if matches := fansubRegex.FindStringSubmatch(filename); len(matches) > 1 {
        pattern.FansubGroup = matches[1]
    }

    // 2. Extract title (remove episode numbers, quality info)
    title := filename
    // Remove fansub group
    title = fansubRegex.ReplaceAllString(title, "")
    // Remove episode numbers (S01E01, - 01, 第01話, etc.)
    title = regexp.MustCompile(`(?i)S\d+E\d+|第?\d+[話话集]?| - \d+`).ReplaceAllString(title, "")
    // Remove quality info
    title = regexp.MustCompile(`(?i)\d+[pP]|BD|WEB|HDTV|x26[45]|HEVC|FLAC`).ReplaceAllString(title, "")
    // Remove extension
    title = regexp.MustCompile(`\.\w+$`).ReplaceAllString(title, "")
    // Clean up
    title = strings.TrimSpace(title)
    pattern.TitlePattern = title

    // 3. Generate regex for matching
    pattern.Regex = e.generateRegex(pattern)

    return pattern, nil
}

func (e *PatternExtractor) generateRegex(p *ExtractedPattern) string {
    var parts []string

    if p.FansubGroup != "" {
        // Match fansub group with flexible brackets
        parts = append(parts, fmt.Sprintf(`[\[【]%s[\]】]`, regexp.QuoteMeta(p.FansubGroup)))
    }

    if p.TitlePattern != "" {
        // Match title with flexible spacing
        parts = append(parts, regexp.QuoteMeta(p.TitlePattern))
    }

    // Allow any episode number, quality info, extension
    parts = append(parts, `.*`)

    return strings.Join(parts, `\s*`)
}
```

### Pattern Matching Engine

```go
// PatternMatcher finds matching patterns for new files
type PatternMatcher struct {
    repo   repository.LearningRepositoryInterface
    logger *slog.Logger
}

type MatchResult struct {
    Pattern    *models.FilenameMapping
    Confidence float64
    MatchType  string // "exact", "regex", "fuzzy"
}

func (m *PatternMatcher) FindMatch(filename string) (*MatchResult, error) {
    // 1. Try exact match first
    if pattern, err := m.repo.FindByExactPattern(filename); err == nil && pattern != nil {
        return &MatchResult{Pattern: pattern, Confidence: 1.0, MatchType: "exact"}, nil
    }

    // 2. Extract pattern from filename
    extractor := &PatternExtractor{}
    extracted, _ := extractor.Extract(filename)

    // 3. Try fansub group + title match
    if extracted.FansubGroup != "" && extracted.TitlePattern != "" {
        patterns, _ := m.repo.FindByFansubAndTitle(extracted.FansubGroup, extracted.TitlePattern)
        if len(patterns) > 0 {
            return &MatchResult{Pattern: patterns[0], Confidence: 0.95, MatchType: "pattern"}, nil
        }
    }

    // 4. Try regex match
    allPatterns, _ := m.repo.ListWithRegex()
    for _, p := range allPatterns {
        if p.PatternRegex != "" {
            re, err := regexp.Compile(p.PatternRegex)
            if err == nil && re.MatchString(filename) {
                return &MatchResult{Pattern: p, Confidence: 0.9, MatchType: "regex"}, nil
            }
        }
    }

    // 5. Try fuzzy match (title only)
    if extracted.TitlePattern != "" {
        patterns, _ := m.repo.ListAll()
        for _, p := range patterns {
            similarity := fuzzyMatch(extracted.TitlePattern, p.TitlePattern)
            if similarity > 0.8 {
                return &MatchResult{Pattern: p, Confidence: similarity, MatchType: "fuzzy"}, nil
            }
        }
    }

    return nil, nil // No match found
}

func fuzzyMatch(s1, s2 string) float64 {
    distance := levenshtein.Distance(strings.ToLower(s1), strings.ToLower(s2))
    maxLen := max(len(s1), len(s2))
    if maxLen == 0 {
        return 1.0
    }
    return 1.0 - float64(distance)/float64(maxLen)
}
```

### API Design

**Learn Pattern Endpoint:**
```
POST /api/v1/learning/patterns
Content-Type: application/json

{
  "filename": "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
  "metadataId": "series-123",
  "metadataType": "series",
  "tmdbId": 85937
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "pattern-456",
    "pattern": "[Leopard-Raws] Kimetsu no Yaiba",
    "fansubGroup": "Leopard-Raws",
    "titlePattern": "Kimetsu no Yaiba",
    "patternType": "fansub",
    "message": "已學習此規則，未來相同格式的檔案將自動套用"
  }
}
```

**List Patterns Endpoint:**
```
GET /api/v1/learning/patterns
```

**Response:**
```json
{
  "success": true,
  "data": {
    "patterns": [
      {
        "id": "pattern-456",
        "pattern": "[Leopard-Raws] Kimetsu no Yaiba",
        "metadataTitle": "鬼滅之刃",
        "useCount": 12,
        "lastUsedAt": "2026-01-18T10:00:00Z"
      }
    ],
    "totalCount": 15,
    "stats": {
      "totalPatterns": 15,
      "totalApplied": 48,
      "mostUsedPattern": "[Leopard-Raws]"
    }
  }
}
```

### Frontend Components

**LearnPatternPrompt.tsx:**
```tsx
interface LearnPatternPromptProps {
  filename: string;
  extractedPattern: ExtractedPattern;
  onConfirm: () => void;
  onSkip: () => void;
}

const LearnPatternPrompt: React.FC<LearnPatternPromptProps> = ({
  filename,
  extractedPattern,
  onConfirm,
  onSkip,
}) => {
  return (
    <Alert className="mt-4">
      <LightbulbIcon className="h-4 w-4" />
      <AlertTitle>學習此規則？</AlertTitle>
      <AlertDescription>
        <p className="text-sm text-gray-600 mb-2">
          系統偵測到以下規則，是否記住以便未來自動套用？
        </p>
        <div className="bg-gray-100 rounded p-2 text-sm font-mono mb-3">
          {extractedPattern.fansubGroup && (
            <span className="text-blue-600">[{extractedPattern.fansubGroup}]</span>
          )}
          {' '}
          <span className="text-green-600">{extractedPattern.titlePattern}</span>
        </div>
        <div className="flex gap-2">
          <Button size="sm" onClick={onConfirm}>
            記住此規則
          </Button>
          <Button size="sm" variant="ghost" onClick={onSkip}>
            這次不用
          </Button>
        </div>
      </AlertDescription>
    </Alert>
  );
};
```

**PatternAppliedToast (UX-5):**
```tsx
const showPatternAppliedToast = (patternTitle: string) => {
  toast.success(
    <div className="flex items-center gap-2">
      <CheckCircleIcon className="h-5 w-5 text-green-500" />
      <span>✓ 已套用你之前的設定</span>
    </div>,
    {
      description: `已自動匹配「${patternTitle}」`,
      duration: 3000,
    }
  );
};
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/learning/
├── pattern.go          # Pattern extraction
├── pattern_test.go
├── matcher.go          # Pattern matching
└── matcher_test.go

/apps/api/internal/repository/
└── learning_repository.go

/apps/api/internal/services/
└── learning_service.go

/apps/api/internal/handlers/
└── learning_handler.go
```

**Frontend Files to Create:**
```
/apps/web/src/components/learning/
├── LearnPatternPrompt.tsx
├── LearnPatternPrompt.spec.tsx
├── LearnedPatternsSettings.tsx
├── LearnedPatternsSettings.spec.tsx
└── index.ts
```

### Testing Strategy

**Backend Tests:**
1. Pattern extraction tests (various filename formats)
2. Pattern matching tests (exact, regex, fuzzy)
3. Repository CRUD tests
4. Service integration tests

**Frontend Tests:**
1. LearnPatternPrompt interaction tests
2. Settings page tests
3. Toast notification tests

**Coverage Targets:**
- Learning package: ≥80%
- Learning service: ≥80%
- Frontend components: ≥70%

### Error Codes

Following project-context.md Rule 7:
- `LEARNING_PATTERN_EXISTS` - Pattern already exists
- `LEARNING_EXTRACTION_FAILED` - Failed to extract pattern
- `LEARNING_SAVE_FAILED` - Failed to save pattern

### Dependencies

**Go Libraries:**
- `github.com/agnivade/levenshtein` - Fuzzy matching

**Story Dependencies:**
- Story 3.7 (Manual Search) - Triggers learning after selection
- Story 3.8 (Metadata Editor) - Can trigger learning after edit

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.9]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR24]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#UX-5]
- [Source: project-context.md#Rule-4-Layered-Architecture]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
