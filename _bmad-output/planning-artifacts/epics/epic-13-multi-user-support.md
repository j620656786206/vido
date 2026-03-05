# Epic 13: Multi-User Support

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** The system supports multiple user accounts with admin/user permission management, and each user has their own personal watch history and preference settings.

## Story 13.1: Multiple User Accounts

As a **household admin**,
I want to **create multiple user accounts**,
So that **each family member has their own profile**.

**Acceptance Criteria:**

**Given** the admin opens Settings > Users
**When** clicking "Add User"
**Then** they enter: Username, Password, Display Name
**And** the new user account is created

**Given** multiple users exist
**When** opening the login page
**Then** user selection is available
**And** each user logs in with their own password

**Given** a user is created
**When** they first log in
**Then** they see an empty watch history
**And** default preferences are applied

**Given** user limit is reached (5 users, NFR-SC4)
**When** trying to add another
**Then** message shows: "Maximum users reached"
**And** suggests removing inactive users

**Technical Notes:**
- Implements FR71: Support multiple user accounts
- Implements NFR-SC4: Support 5 concurrent sessions
- Each user gets separate data tables

---

## Story 13.2: Admin Permission Management

As a **system admin**,
I want to **manage user permissions**,
So that **I control who can modify system settings**.

**Acceptance Criteria:**

**Given** user roles are: Admin, User
**When** an Admin views Settings > Users
**Then** they see all users and their roles
**And** can change roles (except last admin)

**Given** a User role account
**When** accessing settings
**Then** they cannot see: Users, Backup, Automation Rules
**And** they can see: Their profile, Display preferences

**Given** Admin role account
**When** accessing any setting
**Then** full access is granted
**And** system-wide changes are allowed

**Given** the last admin account
**When** trying to change role to User
**Then** action is blocked
**And** message: "At least one admin required"

**Technical Notes:**
- Implements FR72: Manage user permissions
- Role-based access control (RBAC)
- Admin-only sections enforced in UI and API

---

## Story 13.3: Personal Watch History Per User

As a **household member**,
I want **my own watch history**,
So that **my viewing doesn't affect others**.

**Acceptance Criteria:**

**Given** User A marks a movie as watched
**When** User B logs in
**Then** that movie shows as unwatched for User B
**And** watch histories are completely separate

**Given** the user views their dashboard
**When** seeing "Continue Watching" section
**Then** only their own in-progress shows appear
**And** recommendations are based on their history

**Given** the admin wants to view all history
**When** accessing admin dashboard
**Then** aggregate statistics are available
**And** individual user histories remain private

**Given** a user is deleted
**When** admin removes the account
**Then** that user's watch history is deleted
**And** library content is unaffected

**Technical Notes:**
- Implements FR73: Personal watch history per user
- User-scoped database queries
- Privacy maintained between users

---

## Story 13.4: Personal Preference Settings

As a **household member**,
I want **my own preferences**,
So that **my settings don't affect others**.

**Acceptance Criteria:**

**Given** User A sets "Dark Mode"
**When** User B logs in with "Light Mode" preference
**Then** User B sees Light Mode
**And** preferences are user-specific

**Given** personal preferences include:
- Theme (Light/Dark)
- Default view (Grid/List)
- Subtitle language priority
- Dashboard layout
**When** the user changes any setting
**Then** it saves to their profile only

**Given** a new user is created
**When** they first access preferences
**Then** system defaults are applied
**And** they can customize as needed

**Given** the user wants to reset
**When** clicking "Reset to Defaults"
**Then** all personal preferences revert
**And** confirmation is required

**Technical Notes:**
- Implements FR74: Personal preference settings
- User preferences JSON in user table
- System defaults as fallback

---
