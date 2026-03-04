# Pattern Examples

## Good Examples:

**✅ API Endpoint Implementation:**
```go
// /apps/api/internal/handlers/movie_handler.go

// @Summary Get movie by ID
// @Description Retrieve movie metadata from database
// @Tags movies
// @Accept json
// @Produce json
// @Param id path string true "Movie ID"
// @Success 200 {object} ApiResponse[Movie]
// @Failure 404 {object} ApiErrorResponse
// @Router /api/v1/movies/{id} [get]
func (h *MovieHandler) GetMovie(c *gin.Context) {
    id := c.Param("id")

    movie, err := h.service.GetMovieByID(c.Request.Context(), id)
    if err != nil {
        var appErr *AppError
        if !errors.As(err, &appErr) {
            appErr = NewInternalError(err)
        }

        slog.Error("Failed to get movie",
            "error_code", appErr.Code,
            "movie_id", id,
            "error", err,
        )

        ErrorResponse(c, appErr)
        return
    }

    SuccessResponse(c, movie)
}
```

**✅ Frontend Component with Query:**
```typescript
// /apps/web/src/components/library/MovieCard.tsx

import { useQuery } from '@tanstack/react-query';
import { movieService } from '../../services/movieService';

interface MovieCardProps {
  movieId: string;
}

export function MovieCard({ movieId }: MovieCardProps) {
  const { data: movie, isLoading, isError, error } = useQuery({
    queryKey: ['movies', 'detail', movieId],
    queryFn: () => movieService.getMovie(movieId),
  });

  if (isLoading) return <MovieCardSkeleton />;

  if (isError) {
    return (
      <ErrorMessage
        message={error.message}
        suggestion={error.suggestion}
      />
    );
  }

  return (
    <div className="movie-card p-4 rounded-lg shadow-md">
      <h3 className="text-xl font-bold">{movie.title}</h3>
      <p className="text-gray-600">
        {new Date(movie.release_date).getFullYear()}
      </p>
    </div>
  );
}
```

**✅ Database Migration:**
```sql
-- /apps/api/migrations/004_create_users_table.sql

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    pin_hash TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
```

---

## Anti-Patterns (Avoid):

**❌ Direct Repository Access from Handler:**
```go
// BAD: Handler directly accessing repository
func (h *MovieHandler) GetMovie(c *gin.Context) {
    id := c.Param("id")
    movie, err := h.repository.FindByID(id) // ❌ Skip service layer
    // ...
}

// GOOD: Handler calls service, service uses repository
func (h *MovieHandler) GetMovie(c *gin.Context) {
    id := c.Param("id")
    movie, err := h.service.GetMovieByID(c.Request.Context(), id) // ✅
    // ...
}
```

**❌ Using Zustand for Server Data:**
```typescript
// BAD: Using Zustand for API data
const useMovieStore = create((set) => ({
  movie: null,
  fetchMovie: async (id: string) => {
    const movie = await fetchMovie(id);
    set({ movie });
  },
}));

// GOOD: Use TanStack Query for server data
const { data: movie } = useQuery({
  queryKey: ['movie', id],
  queryFn: () => fetchMovie(id),
});
```

**❌ Inconsistent Error Format:**
```json
// BAD: Non-standard error format
{
  "error": "Movie not found"
}

// GOOD: Standard error format
{
  "success": false,
  "error": {
    "code": "DB_NOT_FOUND",
    "message": "找不到指定的電影",
    "suggestion": "請確認電影 ID 是否正確，或嘗試搜尋其他電影。"
  }
}
```

**❌ Wrong Naming Conventions:**
```typescript
// BAD: Mixed naming conventions
interface Movie {
  id: string;
  movieTitle: string;    // ❌ camelCase
  ReleaseDate: string;   // ❌ PascalCase
  tmdb_id: number;       // ✅ Correct
}

// GOOD: Consistent snake_case
interface Movie {
  id: string;
  title: string;
  release_date: string;
  tmdb_id: number;
}
```

---
