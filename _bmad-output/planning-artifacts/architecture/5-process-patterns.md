# 5. Process Patterns

## 5.1 Error Handling Patterns

**MANDATORY Patterns:**

**Backend Error Flow:**
```
Error Occurs → Create AppError → Log with slog → Return ErrorResponse
```

**Example:**
```go
func (h *MovieHandler) GetMovie(c *gin.Context) {
    id := c.Param("id")

    movie, err := h.service.GetMovieByID(c.Request.Context(), id)
    if err != nil {
        // Convert to AppError if not already
        var appErr *AppError
        if !errors.As(err, &appErr) {
            appErr = NewInternalError(err)
        }

        // Log error with context
        slog.Error("Failed to get movie",
            "error_code", appErr.Code,
            "movie_id", id,
            "error", err,
        )

        // Return error response
        ErrorResponse(c, appErr)
        return
    }

    SuccessResponse(c, movie)
}
```

**Frontend Error Handling:**

**TanStack Query Error Handling:**
```typescript
const { data, error, isError } = useQuery({
  queryKey: ['movie', id],
  queryFn: () => fetchMovie(id),
  onError: (error: ApiError) => {
    // Display user-friendly toast
    toast.error(error.message, {
      description: error.suggestion,
    });

    // Log technical details
    console.error(`[${error.code}]`, error.details);
  },
});

if (isError) {
  return <ErrorMessage error={error} />;
}
```

**Global Error Boundary:**
```typescript
class ErrorBoundary extends React.Component<Props, State> {
  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Log to backend error tracking (future)
    console.error('React error boundary caught error', {
      error: error.message,
      componentStack: errorInfo.componentStack,
    });

    this.setState({ hasError: true });
  }

  render() {
    if (this.state.hasError) {
      return <ErrorFallback />;
    }
    return this.props.children;
  }
}
```

**401 Unauthorized Handling:**
```typescript
// Global query client config
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      onError: (error) => {
        if (error.status === 401) {
          authStore.logout();
          router.navigate('/login');
        }
      },
    },
  },
});
```

---

## 5.2 Retry Patterns

**Backend Retry (External APIs):**
- ✅ **Pattern:** Exponential backoff with max retries
- ✅ **Backoff sequence:** 1s → 2s → 4s → 8s
- ✅ **Max retries:**
  - TMDb API: 3 retries
  - AI providers: 2 retries (expensive)
  - qBittorrent: 5 retries
- ✅ **Retry conditions:** Network errors, timeouts, 5xx errors
- ❌ **Don't retry:** 4xx client errors (except 429 rate limit)

**Frontend Retry (TanStack Query):**
- ✅ **Pattern:** Automatic retry with exponential backoff
- ✅ **Config:**
  ```typescript
  const { data } = useQuery({
    queryKey: ['movie', id],
    queryFn: () => fetchMovie(id),
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
  });
  ```

---

## 5.3 Subtitle Pipeline Process Pattern

**MANDATORY Pipeline: Search → Score → Download → Convert → Place**

**Full Pipeline Flow:**
```
Input: MediaItem (title, year, resolution, language preferences)
    ↓
1. SEARCH (parallel)
    ├─→ Assrt API: Search(title, year)       → []SubtitleResult
    ├─→ Zimuku Scraper: Search(title, year)  → []SubtitleResult
    └─→ OpenSubtitles: Search(imdbID)        → []SubtitleResult
    ↓
    Merge all results (deduplicate by hash)
    ↓
2. SCORE (sequential)
    For each result:
      score = (language_match × 0.40)
            + (resolution_match × 0.20)
            + (source_trust × 0.20)
            + (group_match × 0.10)
            + (download_count_normalized × 0.10)
    ↓
    Sort by score descending
    ↓
3. DOWNLOAD (best match with fallback)
    Try top-scored result:
      → Download subtitle file (retry up to 3 times)
      → If failed, try next result
      → If all fail, return error
    ↓
4. CONVERT (OpenCC 簡繁轉換)
    Detect language via content analysis:
      → If zh-Hans and user wants zh-Hant:
          Run OpenCC s2twp (簡→繁 + 臺灣用語)
      → If zh-Hant and user wants zh-Hans:
          Run OpenCC tw2sp (繁→簡 + 大陸用語)
      → Otherwise: pass through unchanged
    ↓
5. PLACE (file output)
    Copy to media directory:
      → Normalize extension: .zh-Hant.srt, .zh-Hans.ass, .en.srt
      → Filename: {media_filename}.{lang_tag}.{ext}
      → Example: Movie.2024.1080p.zh-Hant.srt
    ↓
Output: Subtitle file placed next to media file
```

**Error Handling per Stage:**
- **Search failures:** Log and continue with results from other providers
- **Score:** Never fails (always produces a ranking, even if empty)
- **Download failures:** Try next best-scored result
- **Convert failures:** Fall back to unconverted subtitle (better than nothing)
- **Place failures:** Return error to user (disk permission issue)

**Content-based Language Detection (not filename-based):**
```go
func DetectLanguage(content []byte) LanguageTag {
    // Sample first 500 characters of subtitle text
    // Count CJK character ranges:
    //   U+4E00-U+9FFF (common CJK)
    //   U+3400-U+4DBF (CJK Extension A)
    // Detect simplified-only characters (e.g., 个, 这, 对)
    // Detect traditional-only characters (e.g., 個, 這, 對)
    // Return zh-Hans, zh-Hant, or zh (ambiguous)
}
```

---

## 5.4 Validation Patterns

**Backend Validation:**
- ✅ **Pattern:** Validate at handler layer, before service call
- ✅ **Use:** Gin's binding/validation tags
- ✅ **Example:**
  ```go
  type CreateMovieRequest struct {
      Title       string `json:"title" binding:"required,min=1,max=500"`
      ReleaseDate string `json:"release_date" binding:"required,datetime=2006-01-02"`
      TMDbID      int    `json:"tmdb_id" binding:"omitempty,min=1"`
  }

  func (h *MovieHandler) CreateMovie(c *gin.Context) {
      var req CreateMovieRequest
      if err := c.ShouldBindJSON(&req); err != nil {
          ErrorResponse(c, NewValidationError(err))
          return
      }
      // Proceed with validated request
  }
  ```

**Frontend Validation:**
- ✅ **Pattern:** Client-side validation for UX, server-side for security
- ✅ **Timing:** On blur and on submit
- ✅ **Feedback:** Inline error messages below fields
- ❌ **Anti-pattern:** Client-side only validation (security risk)

---
