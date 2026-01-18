# Story 3.7: Manual Metadata Search and Selection

Status: ready-for-dev

## Story

As a **media collector**,
I want to **manually search and select the correct metadata**,
So that **I can fix misidentified or unfound titles**.

## Acceptance Criteria

1. **AC1: Manual Search Dialog**
   - Given automatic parsing fails
   - When the user clicks "Manual Search"
   - Then a search dialog opens
   - And they can enter a custom search query

2. **AC2: Search Results Display**
   - Given manual search returns results
   - When the user views the results
   - Then they see poster, title, year, and description preview
   - And they can select the correct match

3. **AC3: Selection and Application**
   - Given the user selects a match
   - When confirming the selection
   - Then the metadata is applied to the file
   - And the mapping is saved for learning (Story 3.9)

4. **AC4: Source Selection**
   - Given the manual search dialog is open
   - When the user wants to search specific sources
   - Then they can choose: TMDb, Douban, Wikipedia, or All
   - And results show the source indicator

## Tasks / Subtasks

- [ ] Task 1: Create Manual Search API Endpoint (AC: 1, 4)
  - [ ] 1.1: Create `/api/v1/metadata/manual-search` endpoint
  - [ ] 1.2: Accept query, mediaType, year (optional), source (optional)
  - [ ] 1.3: Return unified search results from selected sources
  - [ ] 1.4: Include source indicator in response
  - [ ] 1.5: Write handler tests

- [ ] Task 2: Create Manual Search Service (AC: 1, 4)
  - [ ] 2.1: Create `ManualSearchService` in services package
  - [ ] 2.2: Implement `Search()` with source selection
  - [ ] 2.3: Aggregate results from multiple sources if "All" selected
  - [ ] 2.4: Sort results by relevance
  - [ ] 2.5: Write service tests

- [ ] Task 3: Create Apply Metadata API Endpoint (AC: 3)
  - [ ] 3.1: Create `/api/v1/media/{id}/apply-metadata` endpoint
  - [ ] 3.2: Accept selected metadata item from search results
  - [ ] 3.3: Update media item in database
  - [ ] 3.4: Set metadata source to selected source
  - [ ] 3.5: Write handler tests

- [ ] Task 4: Create Manual Search Dialog Component (AC: 1, 2)
  - [ ] 4.1: Create `ManualSearchDialog.tsx` component
  - [ ] 4.2: Implement search input with debounce (300ms)
  - [ ] 4.3: Add source selector (TMDb, Douban, Wikipedia, All)
  - [ ] 4.4: Add media type toggle (Movie, TV)
  - [ ] 4.5: Add year filter (optional)
  - [ ] 4.6: Write component tests

- [ ] Task 5: Create Search Results Grid (AC: 2)
  - [ ] 5.1: Create `SearchResultsGrid.tsx` component
  - [ ] 5.2: Display poster, title, year, source badge
  - [ ] 5.3: Show description preview on hover/click
  - [ ] 5.4: Highlight selected item
  - [ ] 5.5: Handle empty results state
  - [ ] 5.6: Write component tests

- [ ] Task 6: Implement Selection and Confirmation (AC: 3)
  - [ ] 6.1: Add "Select" button for each result
  - [ ] 6.2: Show confirmation dialog with selected metadata preview
  - [ ] 6.3: Call apply metadata API on confirm
  - [ ] 6.4: Show success/error toast notification
  - [ ] 6.5: Trigger learning prompt (connects to Story 3.9)
  - [ ] 6.6: Write integration tests

- [ ] Task 7: Integrate with Failed Parse Flow (AC: 1)
  - [ ] 7.1: Add "Manual Search" button to parse failure state
  - [ ] 7.2: Pre-fill search query with parsed title
  - [ ] 7.3: Display fallback chain status (UX-4)
  - [ ] 7.4: Show "What you can do next" guidance

## Dev Notes

### Architecture Requirements

**FR20: Manual search and select metadata**
- Part of the graceful degradation chain

**FR26: Graceful degradation with manual option**
- Always provide manual option when automatic fails

**UX-4: Failure handling friendliness**
- Always show next step
- Explain reasons
- Multi-layer fallback visible

**NFR-R4: Graceful degradation with manual search option**

### API Design

**Manual Search Endpoint:**
```
POST /api/v1/metadata/manual-search
Content-Type: application/json

{
  "query": "Demon Slayer",
  "mediaType": "tv",        // "movie" | "tv"
  "year": 2019,             // optional
  "source": "all"           // "tmdb" | "douban" | "wikipedia" | "all"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "id": "tmdb-85937",
        "source": "tmdb",
        "title": "Demon Slayer: Kimetsu no Yaiba",
        "titleZhTW": "鬼滅之刃",
        "year": 2019,
        "mediaType": "tv",
        "overview": "It is the Taisho Period in Japan...",
        "posterUrl": "https://image.tmdb.org/...",
        "rating": 8.7,
        "confidence": 0.95
      },
      {
        "id": "douban-30277296",
        "source": "douban",
        "title": "鬼灭之刃",
        "titleZhTW": "鬼滅之刃",
        "year": 2019,
        "mediaType": "tv",
        "overview": "大正時期，少年炭治郎...",
        "posterUrl": "https://img.doubanio.com/...",
        "rating": 8.4,
        "confidence": 0.92
      }
    ],
    "totalCount": 2,
    "searchedSources": ["tmdb", "douban"]
  }
}
```

**Apply Metadata Endpoint:**
```
POST /api/v1/media/{id}/apply-metadata
Content-Type: application/json

{
  "selectedItem": {
    "id": "tmdb-85937",
    "source": "tmdb"
  },
  "learnPattern": true  // Optional: trigger learning system
}
```

### Frontend Component Design

**ManualSearchDialog.tsx:**
```tsx
interface ManualSearchDialogProps {
  isOpen: boolean;
  onClose: () => void;
  initialQuery?: string;       // Pre-filled from parse attempt
  mediaId: string;             // Media item to update
  fallbackStatus?: FallbackStatus;  // Show what was tried
  onSuccess: (metadata: MetadataItem) => void;
}

const ManualSearchDialog: React.FC<ManualSearchDialogProps> = ({
  isOpen,
  onClose,
  initialQuery,
  mediaId,
  fallbackStatus,
  onSuccess,
}) => {
  const [query, setQuery] = useState(initialQuery || '');
  const [source, setSource] = useState<'all' | 'tmdb' | 'douban' | 'wikipedia'>('all');
  const [mediaType, setMediaType] = useState<'movie' | 'tv'>('movie');
  const [selectedItem, setSelectedItem] = useState<SearchResultItem | null>(null);

  const { data, isLoading } = useManualSearch({ query, source, mediaType });

  // ... rest of component
};
```

**SearchResultsGrid.tsx:**
```tsx
interface SearchResultsGridProps {
  results: SearchResultItem[];
  selectedId: string | null;
  onSelect: (item: SearchResultItem) => void;
  isLoading: boolean;
}

const SearchResultsGrid: React.FC<SearchResultsGridProps> = ({
  results,
  selectedId,
  onSelect,
  isLoading,
}) => {
  if (isLoading) {
    return <SearchResultsSkeleton />;
  }

  if (results.length === 0) {
    return (
      <EmptyState
        icon={<SearchIcon />}
        title="找不到結果"
        description="試試其他關鍵字或選擇不同的來源"
      />
    );
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
      {results.map((item) => (
        <SearchResultCard
          key={item.id}
          item={item}
          isSelected={item.id === selectedId}
          onSelect={() => onSelect(item)}
        />
      ))}
    </div>
  );
};
```

### UX-4 Integration: Failure Handling Friendliness

**Fallback Status Display:**
```tsx
const FallbackStatusDisplay: React.FC<{ status: FallbackStatus }> = ({ status }) => {
  return (
    <div className="bg-gray-100 rounded-lg p-4 mb-4">
      <h4 className="font-medium text-gray-700 mb-2">已嘗試的來源：</h4>
      <div className="flex items-center gap-2">
        {status.attempts.map((attempt, index) => (
          <React.Fragment key={attempt.source}>
            <span className={`flex items-center gap-1 ${
              attempt.success ? 'text-green-600' : 'text-red-500'
            }`}>
              {attempt.source}
              {attempt.success ? <CheckIcon /> : <XIcon />}
            </span>
            {index < status.attempts.length - 1 && (
              <ArrowRightIcon className="text-gray-400" />
            )}
          </React.Fragment>
        ))}
      </div>
      <p className="text-sm text-gray-600 mt-2">
        {status.attempts.every(a => !a.success)
          ? "所有自動來源都無法找到匹配，請手動搜尋。"
          : ""}
      </p>
    </div>
  );
};
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/handlers/
└── manual_search_handler.go

/apps/api/internal/services/
└── manual_search_service.go
```

**Frontend Files to Create:**
```
/apps/web/src/components/search/
├── ManualSearchDialog.tsx
├── ManualSearchDialog.spec.tsx
├── SearchResultsGrid.tsx
├── SearchResultsGrid.spec.tsx
├── SearchResultCard.tsx
├── SearchResultCard.spec.tsx
├── FallbackStatusDisplay.tsx
└── index.ts
```

### TanStack Query Integration

```typescript
// hooks/useManualSearch.ts
export const useManualSearch = (params: ManualSearchParams) => {
  return useQuery({
    queryKey: ['manual-search', params],
    queryFn: () => metadataService.manualSearch(params),
    enabled: params.query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
};

// hooks/useApplyMetadata.ts
export const useApplyMetadata = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ mediaId, selectedItem, learnPattern }: ApplyMetadataParams) =>
      metadataService.applyMetadata(mediaId, selectedItem, learnPattern),
    onSuccess: (data, variables) => {
      // Invalidate media queries to refresh UI
      queryClient.invalidateQueries({ queryKey: ['media', variables.mediaId] });
      queryClient.invalidateQueries({ queryKey: ['library'] });
    },
  });
};
```

### Testing Strategy

**Backend Tests:**
1. Manual search handler tests
2. Multi-source aggregation tests
3. Apply metadata handler tests

**Frontend Tests:**
1. ManualSearchDialog component tests
2. SearchResultsGrid rendering tests
3. Selection and confirmation flow tests
4. Empty state tests

**E2E Tests:**
1. Full manual search flow
2. Apply metadata and verify update

**Coverage Targets:**
- Backend handlers: ≥70%
- Backend services: ≥80%
- Frontend components: ≥70%

### Error Codes

Following project-context.md Rule 7:
- `MANUAL_SEARCH_NO_RESULTS` - No results found for query
- `MANUAL_SEARCH_INVALID_SOURCE` - Invalid source specified
- `APPLY_METADATA_FAILED` - Failed to apply metadata
- `APPLY_METADATA_NOT_FOUND` - Media item not found

### Dependencies

**Story Dependencies:**
- Story 3.3 (Fallback Chain) - Provides unified search across sources
- Story 3.9 (Learning System) - Triggers pattern learning on selection

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.7]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR20]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#UX-4]
- [Source: project-context.md#Rule-5-TanStack-Query]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.3 (Fallback Chain):**
- `MetadataProvider` interface available for all sources
- `SearchResult` and `MetadataItem` types defined
- Can reuse orchestrator for multi-source search

**From Epic 2 (Search UI):**
- Search component patterns established
- Grid view layout patterns available
- Media card design established

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
