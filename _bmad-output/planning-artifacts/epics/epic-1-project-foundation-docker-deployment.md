# Epic 1: Project Foundation & Docker Deployment

**Phase:** MVP (Q1 - March 2026)

**Goal:** Users can deploy Vido on their NAS within 5 minutes using Docker Compose, with zero-configuration startup and sensible defaults.

## Story 1.1: Repository Pattern Database Abstraction Layer

As a **developer**,
I want a **Repository Pattern abstraction layer for database operations**,
So that **we can migrate from SQLite to PostgreSQL in the future without changing business logic**.

**Acceptance Criteria:**

**Given** the application needs database operations
**When** the developer implements data access code
**Then** all database operations go through repository interfaces (MediaRepository, ConfigRepository)
**And** the SQLite implementation is provided as the default
**And** the interface design supports future PostgreSQL implementation
**And** WAL mode is enabled for SQLite concurrent read performance

**Given** a new entity needs to be persisted
**When** the developer creates the repository method
**Then** the method signature is database-agnostic
**And** only the SQLite-specific implementation contains SQL syntax

**Technical Notes:**
- Implements ARCH-1: Repository Pattern mandatory from MVP
- Implements NFR-SC3: Zero-downtime migration path
- Database schema versioning with migrations table

---

## Story 1.2: Docker Compose Production Configuration

As a **NAS user**,
I want to **deploy Vido using a single docker-compose command**,
So that **I can have the application running within 5 minutes without complex setup**.

**Acceptance Criteria:**

**Given** a user has Docker and Docker Compose installed
**When** they run `docker-compose up -d`
**Then** the Vido container starts successfully within 60 seconds
**And** the web interface is accessible at `http://localhost:8080`
**And** data persists across container restarts via volume mounts

**Given** the container is running
**When** the user checks container health
**Then** a health check endpoint returns status 200
**And** the container reports as "healthy" in Docker

**Given** no environment variables are set
**When** the container starts
**Then** it uses sensible defaults for all configuration
**And** the application is functional without any manual configuration

**Technical Notes:**
- Volume mounts: `/vido-data` (database, cache), `/vido-backups` (backups), `/media` (read-only)
- Implements NFR-U1: Docker Compose deployment <5 minutes
- Implements FR47, FR48

---

## Story 1.3: Environment Variable Configuration System

As a **system administrator**,
I want to **configure Vido using environment variables**,
So that **I can customize the application without modifying files inside the container**.

**Acceptance Criteria:**

**Given** environment variables are set in docker-compose.yml
**When** the container starts
**Then** the application reads and applies all configuration from environment variables
**And** environment variables take precedence over config file values

**Given** the following environment variables are supported:
- `VIDO_PORT` (default: 8080)
- `VIDO_DATA_DIR` (default: /vido-data)
- `VIDO_MEDIA_DIRS` (comma-separated paths)
- `TMDB_API_KEY` (optional)
- `GEMINI_API_KEY` (optional)
- `ENCRYPTION_KEY` (optional, for secrets encryption)
**When** any variable is not set
**Then** the application uses the documented default value
**And** the application logs which configuration source is being used (env var vs default)

**Given** an invalid configuration value is provided
**When** the application starts
**Then** it logs a clear error message indicating the problem
**And** it exits with a non-zero status code (fail fast)

**Technical Notes:**
- Implements FR50: Configure API keys via environment variables
- Implements NFR-S1: API keys support environment variable injection
- Implements NFR-U3: Sensible defaults

---

## Story 1.4: Secrets Management with AES-256 Encryption

As a **security-conscious user**,
I want **my API keys and credentials encrypted when stored**,
So that **my sensitive data is protected even if the database file is accessed**.

**Acceptance Criteria:**

**Given** an API key is saved through the UI
**When** it is stored in the database
**Then** it is encrypted using AES-256-GCM encryption
**And** the plaintext key never appears in database files

**Given** the `ENCRYPTION_KEY` environment variable is set
**When** the application encrypts/decrypts secrets
**Then** it uses this key for encryption operations

**Given** the `ENCRYPTION_KEY` environment variable is NOT set
**When** the application needs an encryption key
**Then** it derives a key from the machine ID as fallback
**And** logs a warning recommending setting ENCRYPTION_KEY for better security

**Given** any application component logs data
**When** the log contains API keys or credentials
**Then** the sensitive values are masked (e.g., `TMDB_****1234`)
**And** the full value never appears in logs, errors, or HTTP responses

**Technical Notes:**
- Implements FR51: Store sensitive data in encrypted format
- Implements NFR-S2: AES-256 encryption for UI-stored keys
- Implements NFR-S3: Encryption key from env var or machine-ID
- Implements NFR-S4: Zero-logging policy for secrets

---

## Story 1.5: Media Folder Configuration

As a **NAS user**,
I want to **configure which folders contain my media files**,
So that **Vido knows where to scan for movies and TV shows**.

**Acceptance Criteria:**

**Given** the user sets `VIDO_MEDIA_DIRS=/movies,/tv,/anime`
**When** the application starts
**Then** it validates that each path exists and is accessible
**And** it stores the configured paths for future scanning operations

**Given** a configured media path does not exist
**When** the application starts
**Then** it logs a warning about the inaccessible path
**And** it continues starting with the valid paths (graceful degradation)

**Given** no media directories are configured
**When** the application starts
**Then** it logs a notice that no media directories are set
**And** the application starts successfully (search-only mode)

**Given** media directories are configured
**When** a user views the settings page
**Then** they see the list of configured media directories
**And** they see the accessibility status of each directory (accessible/not found)

**Technical Notes:**
- Implements FR49: Configure media folder locations
- Paths are read-only mounted in Docker (`/media:ro`)
- Supports multiple directories for different media types

---
