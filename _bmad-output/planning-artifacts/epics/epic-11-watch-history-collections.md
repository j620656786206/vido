# Epic 11: Watch History & Collections

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can track personal watch history, see watch progress indicators, mark media as watched/unwatched, and create custom collections of media items.

## Story 11.1: Watch History Tracking

As a **user**,
I want to **track what I've watched**,
So that **I can remember my viewing history**.

**Acceptance Criteria:**

**Given** the user marks a movie as "Watched"
**When** saving the status
**Then** the watch date is recorded
**And** the movie appears in "Watch History"

**Given** the user marks a TV show episode as "Watched"
**When** saving the status
**Then** the episode is marked
**And** show progress percentage updates

**Given** the user opens Watch History page
**When** viewing the list
**Then** items are sorted by watch date (newest first)
**And** filtering by type (Movies/TV) is available

**Given** the user wants to track rewatches
**When** marking an already-watched item
**Then** option appears: "Mark as rewatched"
**And** rewatch count is incremented

**Technical Notes:**
- Implements FR43: Track personal watch history
- Watch events stored with timestamps
- Foundation for multi-user history (Epic 13)

---

## Story 11.2: Watch Progress Indicators

As a **user**,
I want to **see my watch progress**,
So that **I know what I've finished and what's remaining**.

**Acceptance Criteria:**

**Given** a TV series in the library
**When** viewing the library card
**Then** progress bar shows: X of Y episodes watched
**And** percentage is displayed: "60% complete"

**Given** a movie in the library
**When** viewing the library card
**Then** watched status shows: ✓ (checkmark) or empty
**And** watch date shows if watched

**Given** a series with unwatched episodes
**When** viewing the detail page
**Then** "Continue Watching" shows the next unwatched episode
**And** season progress is displayed per season

**Given** the user filters the library
**When** selecting "In Progress"
**Then** only partially-watched series are shown
**And** sorted by most recently watched

**Technical Notes:**
- Implements FR44: Display watch progress indicators
- Episode-level tracking for TV series
- Visual progress bars in UI

---

## Story 11.3: Mark as Watched/Unwatched

As a **user**,
I want to **easily mark items as watched or unwatched**,
So that **I can manage my watch status**.

**Acceptance Criteria:**

**Given** the user views a movie detail page
**When** clicking the "Mark as Watched" button
**Then** the movie status changes to watched
**And** current date is recorded
**And** button changes to "Mark as Unwatched"

**Given** a TV series detail page
**When** the user right-clicks an episode
**Then** context menu shows: "Mark Watched" / "Mark Unwatched"
**And** bulk options: "Mark season as watched", "Mark all as watched"

**Given** the user accidentally marks something
**When** clicking "Undo" within 5 seconds
**Then** the action is reversed
**And** history is corrected

**Given** the library list view
**When** the user selects multiple items
**Then** batch action: "Mark selected as watched"
**And** confirmation shows count affected

**Technical Notes:**
- Implements FR45: Mark media as watched/unwatched
- Undo support with 5-second window
- Batch operations for efficiency

---

## Story 11.4: Custom Collections

As a **media collector**,
I want to **create custom collections**,
So that **I can organize media by my own categories**.

**Acceptance Criteria:**

**Given** the user opens Collections page
**When** clicking "Create Collection"
**Then** a dialog asks for: Name, Description (optional), Cover image (optional)
**And** the collection is created empty

**Given** a collection exists
**When** the user views a media detail page
**Then** "Add to Collection" button is available
**And** clicking shows list of collections to choose from

**Given** the user views a collection
**When** opening the collection page
**Then** all items in the collection are displayed
**And** custom ordering is supported (drag-drop)

**Given** the user wants to organize collections
**When** editing a collection
**Then** they can: Rename, Change cover, Delete, Export as list
**And** "Smart Collections" option creates auto-updating rules

**Technical Notes:**
- Implements FR46: Create custom collections
- Manual ordering with drag-drop
- Smart collections with filter rules

---
