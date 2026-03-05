# Epic 7: User Authentication & Access Control

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users must authenticate to access Vido, with secure session management and API protection.

## Story 7.1: Password/PIN Setup

As a **first-time user**,
I want to **set up a password or PIN for Vido**,
So that **my media library is protected from unauthorized access**.

**Acceptance Criteria:**

**Given** the user completes the setup wizard
**When** reaching the security step
**Then** they must set a password or PIN
**And** minimum requirements: 6+ characters (password) or 4+ digits (PIN)

**Given** a password is set
**When** stored in the database
**Then** it is hashed using bcrypt
**And** the plaintext is never stored

**Given** the user wants to change their password
**When** accessing Settings > Security
**Then** they must enter current password first
**And** can then set a new password

**Technical Notes:**
- Implements FR67: Authenticate via password/PIN
- Bcrypt with appropriate cost factor
- Password strength indicator in UI

---

## Story 7.2: Login Page

As a **returning user**,
I want to **log in with my password or PIN**,
So that **I can access my media library**.

**Acceptance Criteria:**

**Given** the user is not authenticated
**When** accessing any Vido page
**Then** they are redirected to the login page

**Given** the login page is displayed
**When** the user enters correct credentials
**Then** they are authenticated
**And** redirected to their intended destination

**Given** incorrect credentials are entered
**When** login fails
**Then** an error message is shown: "Invalid password"
**And** failed attempt is logged (NFR-S13)

**Given** 5 failed attempts in 15 minutes
**When** the limit is reached
**Then** login is temporarily blocked
**And** message: "Too many attempts. Try again in X minutes."

**Technical Notes:**
- Implements NFR-S13: Failed auth attempts rate-limited
- No username required (single-user system)
- Rate limiting per IP address

---

## Story 7.3: Session Management

As an **authenticated user**,
I want **my session to be secure and persistent**,
So that **I don't have to log in repeatedly but remain protected**.

**Acceptance Criteria:**

**Given** the user logs in successfully
**When** a session is created
**Then** a cryptographically-signed JWT token is issued (NFR-S10)
**And** stored in httpOnly cookie

**Given** a session is active
**When** the user makes requests
**Then** the session token is validated
**And** refreshed automatically before expiry

**Given** the session expires (after 7 days)
**When** the user tries to access Vido
**Then** they are redirected to login
**And** a message indicates session expiration

**Given** the user clicks "Logout"
**When** logout is processed
**Then** the session is invalidated
**And** the user is redirected to login

**Technical Notes:**
- Implements FR68: Manage user sessions
- Implements NFR-S10: Cryptographically-signed tokens
- JWT with RS256 or HS256 signing

---

## Story 7.4: API Authentication

As a **developer integrating with Vido**,
I want **API endpoints to be authenticated**,
So that **only authorized requests can access data**.

**Acceptance Criteria:**

**Given** an API request is made
**When** no authentication token is provided
**Then** the request is rejected with 401 Unauthorized

**Given** a valid session cookie exists
**When** making API requests from the browser
**Then** the session cookie authenticates the request
**And** CSRF protection is enforced

**Given** an API token is generated
**When** used in Authorization header
**Then** the request is authenticated
**And** the token has configurable expiration

**Technical Notes:**
- Implements FR69: Protect API endpoints
- Implements NFR-S11: API endpoints protected
- Support both session and API token auth

---

## Story 7.5: Rate Limiting

As a **system administrator**,
I want **API rate limiting**,
So that **the system is protected from abuse**.

**Acceptance Criteria:**

**Given** API requests are made
**When** rate exceeds 100 requests/minute from same IP
**Then** subsequent requests return 429 Too Many Requests
**And** Retry-After header indicates when to retry

**Given** rate limit is hit
**When** the user sees the error
**Then** a friendly message explains the limit
**And** suggests waiting before retrying

**Given** different endpoints
**When** rate limits are applied
**Then** more restrictive limits for sensitive operations (login)
**And** relaxed limits for read-only operations

**Technical Notes:**
- Implements FR70: Implement rate limiting
- Implements NFR-S12: Rate limiting (100 req/min per IP)
- Token bucket or sliding window algorithm

---
