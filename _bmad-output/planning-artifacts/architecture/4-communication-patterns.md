# 4. Communication Patterns

## 4.1 State Management Patterns

**MANDATORY Rules:**

**Server State (TanStack Query):**
- ✅ **Use for:** All data from backend API
- ✅ **Pattern:** Define query keys with hierarchical structure
- ✅ **Examples:**
  ```typescript
  // Query keys
  const movieKeys = {
    all: ['movies'] as const,
    lists: () => [...movieKeys.all, 'list'] as const,
    list: (filters: string) => [...movieKeys.lists(), { filters }] as const,
    details: () => [...movieKeys.all, 'detail'] as const,
    detail: (id: string) => [...movieKeys.details(), id] as const,
  };

  // Usage
  const { data: movie } = useQuery({
    queryKey: movieKeys.detail(movieId),
    queryFn: () => fetchMovie(movieId),
  });
  ```

**Global Client State (Zustand if needed):**
- ✅ **Use for:** UI state, user preferences, auth state
- ✅ **Pattern:** Single store per domain
- ✅ **Example:**
  ```typescript
  // stores/authStore.ts
  interface AuthState {
    isAuthenticated: boolean;
    user: User | null;
    login: (credentials: Credentials) => Promise<void>;
    logout: () => void;
  }

  export const useAuthStore = create<AuthState>((set) => ({
    isAuthenticated: false,
    user: null,
    login: async (credentials) => { /* ... */ },
    logout: () => set({ isAuthenticated: false, user: null }),
  }));
  ```
- ❌ **Anti-pattern:** Using Zustand for server data (use TanStack Query)

**Local Component State (useState):**
- ✅ **Use for:** Form inputs, toggle states, local UI state
- ✅ **Pattern:** Keep state as close to usage as possible
- ❌ **Anti-pattern:** Lifting state unnecessarily high

**State Update Patterns:**
- ✅ **Immutable updates:** Always create new objects/arrays
- ✅ **Example:**
  ```typescript
  // Correct
  setMovies(prev => [...prev, newMovie]);
  setUser(prev => ({ ...prev, name: newName }));

  // Incorrect
  movies.push(newMovie); // Mutates state
  user.name = newName;   // Mutates state
  ```

---

## 4.2 Loading State Patterns

**MANDATORY Patterns:**

**TanStack Query States:**
- ✅ **Use built-in states:** `isLoading`, `isFetching`, `isError`
- ✅ **Pattern:**
  ```typescript
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['movie', id],
    queryFn: () => fetchMovie(id),
  });

  if (isLoading) return <LoadingSpinner />;
  if (isError) return <ErrorMessage error={error} />;
  return <MovieDetail movie={data} />;
  ```

**Loading UI Conventions:**
- ✅ **Initial load:** Full-page spinner or skeleton screen
- ✅ **Background refresh:** Subtle indicator (e.g., spinning icon in corner)
- ✅ **Pagination:** "Load More" button or skeleton items
- ❌ **Anti-pattern:** Blocking entire UI during background refresh

**Skeleton Screens:**
- ✅ **Use for:** Initial loads of content-heavy components
- ✅ **Example:** MovieCard skeleton with gray blocks matching layout

**Progress Indicators:**
- ✅ **AI Parsing (10s operation):** Progress bar or animated dots
- ✅ **File uploads:** Percentage-based progress bar
- ✅ **Quick operations (<1s):** No loading state (instant feedback)

---

## 4.3 SSE (Server-Sent Events) Communication Pattern

**MANDATORY for real-time updates (replaces polling):**

**SSE Event Format:**
```
id: {monotonic-event-id}
event: {event_type}
data: {json_payload}

```

**Event Types and Payloads:**
```typescript
// download_progress
event: download_progress
data: {"id": "abc123", "name": "Movie.mkv", "progress": 45.2, "speed": "12.5 MB/s", "eta": 120}

// scan_status
event: scan_status
data: {"status": "scanning", "current": "/media/movies", "found": 42, "matched": 38}

// subtitle_status
event: subtitle_status
data: {"mediaId": "123", "status": "downloading", "provider": "assrt", "language": "zh-Hant"}

// notification
event: notification
data: {"level": "info", "message": "Media scan complete: 5 new items found"}
```

**Client Connection:**
```typescript
const eventSource = new EventSource('/api/v1/events');

eventSource.addEventListener('download_progress', (e) => {
  const data = JSON.parse(e.data);
  // Update download progress in UI
});

// Reconnection with Last-Event-ID is automatic in EventSource API
```

**Last-Event-ID Support:**
- ✅ Server includes monotonic `id` field in every event
- ✅ On reconnect, browser sends `Last-Event-ID` header automatically
- ✅ Server replays missed events since that ID (buffered for up to 5 minutes)
- ❌ Do NOT rely on event replay for critical state — use REST API for initial state

**Backend Pattern:**
```go
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    clientCh := h.hub.Register(r.Context())
    defer h.hub.Deregister(clientCh)

    for {
        select {
        case event := <-clientCh:
            fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, event.JSON())
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

---
