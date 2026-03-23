# Epic 10: Smart Recommendations & Discovery

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can receive smart recommendations based on genre, cast, and director, and see "similar titles" suggestions to discover new content.

## Story 10.1: Genre-Based Recommendations

As a **media collector**,
I want to **receive recommendations based on my genres**,
So that **I can discover similar content I might enjoy**.

**Acceptance Criteria:**

**Given** the user has media in their library
**When** the system analyzes their collection
**Then** it identifies the top genres by count
**And** generates recommendations based on genre overlap

**Given** recommendations are generated
**When** viewing the Dashboard
**Then** a "Recommended for You" section appears
**And** shows 6-12 recommendations with posters

**Given** a recommendation is displayed
**When** hovering over it
**Then** the reason is shown: "Because you like [Genre]"
**And** clicking opens the detail page

**Given** the user dismisses a recommendation
**When** clicking "Not Interested"
**Then** that title is hidden
**And** similar content is de-prioritized

**Technical Notes:**
- Implements FR9: Smart recommendations based on genre
- Uses TMDb similar/recommendations API
- Local caching for recommendations

---

## Story 10.2: Cast and Director Based Recommendations

As a **media collector**,
I want **recommendations based on actors and directors I follow**,
So that **I discover their other works**.

**Acceptance Criteria:**

**Given** the user's library has multiple works by the same director
**When** generating recommendations
**Then** other works by that director are suggested
**And** reason shows: "From director [Name]"

**Given** the user's library has multiple works with the same actor
**When** generating recommendations
**Then** other works featuring that actor are suggested
**And** reason shows: "[Actor Name] is in this"

**Given** recommendations are personalized
**When** viewing a specific media detail
**Then** "More from this director" section appears
**And** "More with [Lead Actor]" section appears

**Given** the user explicitly "follows" an actor/director
**When** new content becomes available
**Then** it's highlighted in recommendations
**And** optional notification is sent

**Technical Notes:**
- Implements FR9: Smart recommendations based on cast, director
- TMDb person credits API
- Follow feature stored in user preferences

---

## Story 10.3: Similar Titles Suggestions

As a **media collector**,
I want to **see similar titles for media I'm viewing**,
So that **I can find related content**.

**Acceptance Criteria:**

**Given** the user views a media detail page
**When** scrolling to the bottom
**Then** "Similar Titles" section shows 6-10 related items
**And** items are sourced from TMDb similar/recommendations API

**Given** similar titles are displayed
**When** one is already in the user's library
**Then** it shows "In Your Library" badge
**And** clicking goes to the library entry, not external

**Given** similar titles are displayed
**When** one is not in the library
**Then** it shows basic info (title, year, poster)
**And** clicking shows a mini-detail modal

**Given** the user is browsing similar titles
**When** they want to add one
**Then** "Add to Wishlist" button is available
**And** the wishlist can be exported for future reference

**Technical Notes:**
- Implements FR10: Similar titles suggestions
- TMDb similar movies/TV shows endpoint
- Local library cross-reference

---
