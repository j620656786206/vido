# 3. Format Patterns

## 3.1 API Response Formats

**MANDATORY Standard Response Wrapper:**

```typescript
// Success Response
interface ApiResponse<T> {
  success: true;
  data: T;
  meta?: {
    page?: number;
    per_page?: number;
    total_count?: number;
    has_more?: boolean;
  };
}

// Error Response
interface ApiErrorResponse {
  success: false;
  error: {
    code: string;          // Error code (e.g., "TMDB_TIMEOUT")
    message: string;       // User-friendly message (Traditional Chinese)
    suggestion?: string;   // Troubleshooting hint
    details?: string;      // Technical details (only in development)
  };
}
```

**Examples:**

```json
// GET /api/v1/movies/{id} - Success
{
  "success": true,
  "data": {
    "id": "abc123",
    "title": "範例電影",
    "release_date": "2024-01-15T00:00:00Z",
    "genres": ["動作", "科幻"],
    "tmdb_id": 12345
  }
}

// GET /api/v1/movies - Success with pagination
{
  "success": true,
  "data": [
    { "id": "abc123", "title": "電影 1" },
    { "id": "def456", "title": "電影 2" }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total_count": 150,
    "has_more": true
  }
}

// Error Response
{
  "success": false,
  "error": {
    "code": "TMDB_TIMEOUT",
    "message": "無法連線到 TMDb API，請稍後再試",
    "suggestion": "檢查網路連線或稍後重試。如果問題持續，請確認 TMDb API 狀態。"
  }
}
```

**Go Implementation Pattern:**

```go
// Success response helper
func SuccessResponse(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    data,
    })
}

// Success with meta
func SuccessResponseWithMeta(c *gin.Context, data interface{}, meta map[string]interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    data,
        "meta":    meta,
    })
}

// Error response helper
func ErrorResponse(c *gin.Context, err *AppError) {
    c.JSON(err.HTTPStatus, gin.H{
        "success": false,
        "error": gin.H{
            "code":       err.Code,
            "message":    err.Message,
            "suggestion": err.Suggestion,
        },
    })
}
```

**TypeScript Client Pattern:**

```typescript
// API service function
async function getMovie(id: string): Promise<Movie> {
  const response = await fetch(`/api/v1/movies/${id}`);
  const json: ApiResponse<Movie> | ApiErrorResponse = await response.json();

  if (!json.success) {
    throw new ApiError(json.error);
  }

  return json.data;
}
```

**Current Codebase Compliance:**
- ✅ **Shared types:** `ApiResponse<T>` interface exists in `libs/shared-types`
- ⚠️ **Backend:** No endpoints implemented yet, enforce during Phase 2-4
- ⚠️ **Frontend:** No API services exist yet, enforce during Phase 4

---

## 3.2 Date/Time Format Standards

**MANDATORY Rules:**

**API JSON Responses:**
- ✅ **Format:** ISO 8601 strings with timezone (UTC)
- ✅ **Example:** `"2024-01-15T14:30:00Z"`
- ❌ **Anti-pattern:** UNIX timestamps, `"2024-01-15"`, `"01/15/2024"`

**Database Storage:**
- ✅ **SQLite:** `TIMESTAMP` with `DEFAULT CURRENT_TIMESTAMP`
- ✅ **Format:** ISO 8601 string or Unix timestamp (consistent per column)
- ✅ **Examples:** `created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`

**Go Backend:**
- ✅ **Type:** `time.Time`
- ✅ **JSON marshaling:** Automatic ISO 8601 via `encoding/json`
- ✅ **Example:**
  ```go
  type Movie struct {
      ID          string    `json:"id"`
      ReleaseDate time.Time `json:"release_date"` // Marshals to ISO 8601
      CreatedAt   time.Time `json:"created_at"`
  }
  ```

**TypeScript Frontend:**
- ✅ **Type:** `string` (ISO 8601) in interfaces, `Date` object for manipulation
- ✅ **Example:**
  ```typescript
  interface Movie {
    releaseDate: string; // ISO 8601 string from API
  }

  // Usage
  const releaseDate = new Date(movie.releaseDate);
  const formatted = releaseDate.toLocaleDateString('zh-TW');
  ```

**Display Formatting:**
- ✅ **Pattern:** Use locale-aware formatting for display
- ✅ **Examples:**
  - `new Date(isoString).toLocaleDateString('zh-TW')` → `2024年1月15日`
  - Relative time: `「3 天前」`, `「2 小時前」`

---

## 3.3 Error Code System

**MANDATORY Error Code Format:** `{SOURCE}_{ERROR_TYPE}`

**Error Code Categories:**

**TMDb Errors (`TMDB_*`):**
- `TMDB_TIMEOUT` - API request timeout
- `TMDB_RATE_LIMIT` - Rate limit exceeded
- `TMDB_NOT_FOUND` - Movie/TV show not found
- `TMDB_AUTH_FAILED` - Invalid API key
- `TMDB_NETWORK_ERROR` - Network connectivity issue

**AI Provider Errors (`AI_*`):**
- `AI_TIMEOUT` - AI parsing timeout (>10s)
- `AI_QUOTA_EXCEEDED` - User's API quota exhausted
- `AI_INVALID_RESPONSE` - Unparseable AI response
- `AI_PROVIDER_ERROR` - Generic provider error

**qBittorrent Errors (`QBIT_*`):**
- `QBIT_CONNECTION_FAILED` - Cannot connect to qBittorrent
- `QBIT_AUTH_FAILED` - Invalid credentials
- `QBIT_TORRENT_NOT_FOUND` - Torrent not found

**Database Errors (`DB_*`):**
- `DB_CONNECTION_FAILED` - Database connection error
- `DB_QUERY_FAILED` - Query execution error
- `DB_CONSTRAINT_VIOLATION` - Unique constraint violation
- `DB_NOT_FOUND` - Record not found

**Authentication Errors (`AUTH_*`):**
- `AUTH_INVALID_CREDENTIALS` - Wrong password/PIN
- `AUTH_TOKEN_EXPIRED` - JWT expired
- `AUTH_TOKEN_INVALID` - Malformed JWT
- `AUTH_UNAUTHORIZED` - No valid authentication

**Validation Errors (`VALIDATION_*`):**
- `VALIDATION_REQUIRED_FIELD` - Required field missing
- `VALIDATION_INVALID_FORMAT` - Invalid data format
- `VALIDATION_OUT_OF_RANGE` - Value out of acceptable range

**Implementation Example:**

```go
// /apps/api/internal/errors/app_error.go
type AppError struct {
    Code       string
    Message    string
    Details    string
    Suggestion string
    HTTPStatus int
    Err        error
}

func NewTMDbTimeoutError(err error) *AppError {
    return &AppError{
        Code:       "TMDB_TIMEOUT",
        Message:    "無法連線到 TMDb API，請稍後再試",
        Details:    fmt.Sprintf("TMDb API request timed out: %v", err),
        Suggestion: "檢查網路連線或稍後重試。如果問題持續，請確認 TMDb API 狀態。",
        HTTPStatus: http.StatusGatewayTimeout,
        Err:        err,
    }
}
```

---

## 3.4 JSON Field Naming

**MANDATORY Rules:**

**API JSON (External Interface):**
- ✅ **Format:** `snake_case`
- ✅ **Examples:** `tmdb_id`, `release_date`, `created_at`, `user_id`
- ❌ **Anti-pattern:** `tmdbId`, `releaseDate`, `createdAt`

**Go Struct Tags:**
- ✅ **Pattern:** Use `json` tags with `snake_case`
- ✅ **Example:**
  ```go
  type Movie struct {
      ID          string    `json:"id"`
      Title       string    `json:"title"`
      ReleaseDate time.Time `json:"release_date"`
      TMDbID      int       `json:"tmdb_id"`
  }
  ```

**TypeScript Interfaces (Matching API):**
- ✅ **Pattern:** `snake_case` for fields from API
- ✅ **Example:**
  ```typescript
  interface Movie {
    id: string;
    title: string;
    release_date: string;
    tmdb_id?: number;
  }
  ```
- ⚠️ **Note:** Match backend exactly to avoid transformation bugs

**Internal TypeScript Code:**
- ✅ **Pattern:** `camelCase` for internal variables AFTER transformation
- ✅ **Example:**
  ```typescript
  // API response
  const apiMovie: Movie = await fetchMovie(id);

  // Internal usage (if transformation needed)
  const releaseYear = new Date(apiMovie.release_date).getFullYear();
  ```

**Current Codebase Compliance:**
- ✅ **Shared types:** Use `snake_case` matching Go backend
- ⚠️ **Enforcement:** Ensure all new code follows `snake_case` for JSON fields

---
