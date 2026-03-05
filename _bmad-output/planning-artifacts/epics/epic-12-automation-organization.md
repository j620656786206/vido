# Epic 12: Automation & Organization

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** The system can monitor watch folders to detect new files, automatically trigger parsing, rename files based on patterns, move files to organized structure, and execute automation tasks in background queue.

## Story 12.1: Watch Folder Monitoring

As a **media collector**,
I want **folders monitored for new files**,
So that **new downloads are automatically detected**.

**Acceptance Criteria:**

**Given** the user opens Settings > Automation
**When** adding a watch folder
**Then** they can browse/enter a folder path
**And** select file extensions to watch (e.g., .mkv, .mp4, .avi)

**Given** a watch folder is configured
**When** a new file appears in that folder
**Then** it is detected within 30 seconds
**And** appears in "Pending Processing" queue

**Given** file detection occurs
**When** the file is still being written
**Then** the system waits until file size is stable
**And** processes only after write is complete

**Given** multiple watch folders are configured
**When** viewing the automation dashboard
**Then** all watched folders are listed with status
**And** file counts per folder are shown

**Technical Notes:**
- Implements FR81: Monitor watch folders
- inotify/fsnotify for file detection
- File stability check before processing

---

## Story 12.2: Automatic Parsing Trigger

As a **media collector**,
I want **new files automatically parsed**,
So that **metadata is retrieved without manual intervention**.

**Acceptance Criteria:**

**Given** a new file is detected in a watch folder
**When** the file is stable and ready
**Then** parsing is automatically triggered
**And** uses the same logic as manual parsing (Epic 3)

**Given** automatic parsing succeeds
**When** metadata is retrieved
**Then** the file is added to the library
**And** status shows "Auto-processed"

**Given** automatic parsing fails
**When** no metadata is found
**Then** the file is marked as "Needs Review"
**And** notification alerts the user (if enabled)

**Given** many files arrive simultaneously
**When** queue builds up
**Then** files are processed in order (oldest first)
**And** processing rate respects API limits

**Technical Notes:**
- Implements FR82: Auto-trigger parsing
- Reuses parsing logic from Epic 3
- Queue management (ARCH-4)

---

## Story 12.3: Automatic File Renaming

As a **media collector**,
I want **files renamed based on patterns**,
So that **my library has consistent naming**.

**Acceptance Criteria:**

**Given** the user configures rename patterns in Settings
**When** setting up the pattern
**Then** variables are available:
- `{title}`, `{year}`, `{quality}`, `{codec}`
- `{season}`, `{episode}` (for TV)
- Example: `{title} ({year}) - {quality}.{ext}`

**Given** a file is successfully parsed
**When** rename automation is enabled
**Then** the file is renamed according to the pattern
**And** original name is logged for reference

**Given** rename would cause conflict
**When** target filename exists
**Then** a suffix is added: `(1)`, `(2)`
**And** user is notified of the conflict

**Given** the user wants to preview
**When** configuring patterns
**Then** a "Preview" shows example renames
**And** dry-run mode tests without actual changes

**Technical Notes:**
- Implements FR83: Auto-rename files
- Pattern template system
- Conflict resolution with suffixes

---

## Story 12.4: Automatic File Organization

As a **media collector**,
I want **files moved to organized folders**,
So that **my library structure is consistent**.

**Acceptance Criteria:**

**Given** the user configures organization in Settings
**When** setting up folder structure
**Then** patterns are available:
- Movies: `/media/Movies/{title} ({year})/`
- TV: `/media/TV/{title}/Season {season}/`

**Given** a file is successfully parsed
**When** organization automation is enabled
**Then** the file is moved to the target folder
**And** parent folders are created if needed

**Given** the source and target are different drives
**When** moving the file
**Then** file is copied first, then source deleted
**And** integrity is verified before deleting source

**Given** the user has custom requirements
**When** configuring advanced rules
**Then** genre-based folders are supported
**And** year-based folders: `/media/Movies/2024/`

**Technical Notes:**
- Implements FR84: Auto-move files to organized structure
- Cross-filesystem move support
- Verify integrity before deleting source

---

## Story 12.5: Background Task Queue

As a **system administrator**,
I want **automation tasks processed in background**,
So that **the UI remains responsive**.

**Acceptance Criteria:**

**Given** automation tasks are triggered
**When** they enter the queue
**Then** each task has: ID, Type, Status, Progress, Created time

**Given** the user opens Automation > Queue
**When** viewing the queue
**Then** all pending and running tasks are listed
**And** completed tasks show for 24 hours

**Given** a task is running
**When** viewing its details
**Then** progress percentage is shown
**And** current step is described

**Given** a task fails
**When** viewing the queue
**Then** status shows "Failed" with error message
**And** "Retry" button is available

**Technical Notes:**
- Implements FR85: Background processing queue
- Implements ARCH-4: Background Task Queue
- Persistent queue survives restarts

---

## Story 12.6: Automation Rules Configuration

As a **media collector**,
I want to **configure automation rules**,
So that **I can customize how files are processed**.

**Acceptance Criteria:**

**Given** the user opens Settings > Automation Rules
**When** creating a new rule
**Then** they configure:
- Watch folder(s)
- File filters (extension, size, name pattern)
- Actions: Parse, Rename, Move, Notify

**Given** multiple rules exist
**When** a file matches multiple rules
**Then** rules are applied in priority order
**And** first match wins (or all match option)

**Given** a rule is configured
**When** the user wants to test
**Then** "Test Rule" shows what would happen
**And** no actual changes are made

**Given** the user wants presets
**When** selecting from templates
**Then** common patterns are available:
- "Movies to /media/Movies/"
- "TV Shows by Season"
- "Anime with fansub naming"

**Technical Notes:**
- Implements FR86: Configure automation rules
- Rule priority system
- Dry-run test mode

---
