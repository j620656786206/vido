# Technical Stack Foundation (Brownfield Project)

## Existing Technology Stack

Vido is a **brownfield project** with an established technical foundation. The following technology decisions have already been made and are reflected in the existing codebase:

**Backend Architecture:**

- **Language:** Go 1.21+
  - Statically typed with strong type safety
  - High performance and low resource footprint
  - Native concurrency support (goroutines, channels)
  - Excellent standard library for HTTP, JSON, and networking
  
- **Web Framework:** Gin
  - Lightweight, high-performance HTTP routing framework
  - Middleware support for cross-cutting concerns
  - Native JSON binding and validation
  - Widely adopted in Go ecosystem
  
- **Development Experience:** Air (cosmtrek/air)
  - Automatic hot reload during development
  - Watches `.go` files and rebuilds on changes
  - Configured via `.air.toml` (already present)
  - Significantly improves development iteration speed
  
- **API Documentation:** Swaggo
  - Automatic OpenAPI/Swagger specification generation
  - Code annotation-based documentation
  - Swagger UI integration for interactive API testing
  - Configured with `swag init` command
  
- **Database:** SQLite with WAL mode
  - Zero-configuration, single-file database
  - Perfect for self-hosted deployment model
  - WAL (Write-Ahead Logging) mode for concurrent read performance
  - Supports up to 10,000 media items (scalability boundary defined in PRD)
  
- **Data Access Pattern:** Repository Pattern (architectural requirement)
  - Database abstraction layer required from MVP
  - Enables future migration from SQLite to PostgreSQL
  - Interface-based design for implementation swapping
  - Critical for achieving NFR-SC2 (zero-downtime migration)

**Frontend Architecture:**

- **Framework:** React 19
  - Latest stable version with modern React features
  - Server Components ready (future consideration)
  - Improved concurrent rendering
  - Better TypeScript integration
  
- **Routing:** TanStack Router
  - Type-safe routing with full TypeScript support
  - File-based routing conventions
  - Built-in code splitting
  - Excellent developer experience
  
- **Data Fetching:** TanStack Query v5
  - Server state management and caching
  - Built-in polling support (critical for qBittorrent monitoring)
  - Automatic retry with exponential backoff (satisfies NFR-R5)
  - Optimistic updates for better UX
  - Devtools for debugging
  
- **Build Tooling:** Vite
  - Fast development server with HMR (Hot Module Replacement)
  - Native ESM support
  - Optimized production builds
  - Plugin ecosystem for additional functionality
  - Achieves <500KB bundle target (NFR-P17)
  
- **Language:** TypeScript (strict mode)
  - Type safety across frontend codebase
  - Reduces runtime errors
  - Better IDE support and autocomplete
  - Enforces architectural patterns through types

**Project Organization:**

- **Monorepo Management:** Nx
  - Unified workspace for frontend and backend
  - Shared configuration and tooling
  - Task orchestration and caching
  - Dependency graph visualization
  
- **Containerization:** Docker + Docker Compose
  - Standardized deployment environment
  - Isolation from host system
  - Version-pinned dependencies
  - Achieves <5 minute deployment goal (NFR-U1)
  
- **Version Control:** Git
  - Worktree workflow established (evidence: `.worktrees/` directory)
  - Feature branch development pattern

## Architecture Alignment with PRD Requirements

**Performance Requirements Satisfaction (18 NFRs):**

✅ **Satisfied by existing stack:**
- Go + Gin backend: Native support for <500ms p95 API response times (NFR-P5)
- React 19 + Vite: Modern build tooling supports <500KB bundle, code splitting (NFR-P17, NFR-P18)
- TanStack Query: Built-in caching and polling for real-time updates <5s (NFR-P8)
- SQLite WAL mode: Concurrent read performance, supports 10,000 items (NFR-SC1)

⚠️ **Requires implementation:**
- Virtual scrolling components for large libraries (NFR-P10, NFR-SC6)
- SQLite FTS5 full-text search for <500ms query time (NFR-P15, NFR-SC8)
- Image lazy loading with Intersection Observer (NFR-P12)
- Route-based code splitting configuration (NFR-P18)

**Security Requirements Satisfaction (19 NFRs):**

✅ **Satisfied by existing stack:**
- Go type safety: Reduces memory vulnerabilities, injection attacks (contributes to NFR-S19)
- TypeScript strict mode: Frontend type safety, reduces XSS vulnerabilities (contributes to NFR-S19)

⚠️ **Requires implementation:**
- AES-256 encryption layer for secrets (NFR-S2, NFR-S3)
- Authentication middleware (password/PIN, session tokens) (NFR-S9, NFR-S10, NFR-S11)
- Rate limiting middleware (NFR-S12, NFR-S13)
- Secrets management service (NFR-S1, NFR-S4, NFR-S5, NFR-S6)
- API key zero-logging enforcement (NFR-S4)

**Scalability Requirements Satisfaction (10 NFRs):**

✅ **Satisfied by existing stack:**
- Repository Pattern: Architectural requirement confirmed, enables SQLite → PostgreSQL migration (NFR-SC3)
- SQLite: Supports 10,000 media items target (NFR-SC1)

⚠️ **Requires implementation:**
- Performance monitoring dashboard (NFR-SC2, NFR-SC10)
- 8,000 items warning mechanism (NFR-SC2)
- FTS5 full-text search indexing (NFR-SC8)
- Virtual scrolling implementation (NFR-SC6, NFR-SC7)

**Reliability Requirements Satisfaction (13 NFRs):**

✅ **Satisfied by existing stack:**
- Go explicit error handling: Supports graceful degradation patterns (NFR-R2, NFR-R4)
- TanStack Query: Auto-retry with exponential backoff built-in (NFR-R5)
- SQLite ACID transactions: Data integrity guaranteed (NFR-R10)

⚠️ **Requires implementation:**
- Circuit breaker pattern for external services (NFR-R2, NFR-R6, NFR-R11)
- Health check scheduler (NFR-R6, NFR-R12)
- Atomic backup system with checksums (NFR-R7, NFR-R8, NFR-R9)
- Automatic recovery mechanisms (NFR-R3, NFR-R5)

**Integration Requirements Satisfaction (13 NFRs):**

✅ **Satisfied by existing stack:**
- Go native HTTP client: Supports RESTful API integration (qBittorrent, TMDb, Wikipedia) (NFR-I1, NFR-I2)
- Swaggo: OpenAPI/Swagger spec generation (NFR-I17)

⚠️ **Requires implementation:**
- Adapter pattern for external services (NFR-I1, NFR-I6, NFR-I9, NFR-I13)
- AI provider abstraction layer (NFR-I9, NFR-I10, NFR-I11, NFR-I12)
- Rate limiting for external APIs (NFR-I6, NFR-I14)
- API versioning strategy (/api/v1) (NFR-I16)
- Webhook support infrastructure (NFR-I18)

**Maintainability Requirements Satisfaction (13 NFRs):**

✅ **Satisfied by existing stack:**
- Air + Vite HMR: Hot reload configured (NFR-M5)
- Swaggo: API documentation auto-generation (NFR-M7, NFR-U7)
- Nx monorepo: Unified build and test workflow (NFR-M6, NFR-M7)

⚠️ **Requires implementation:**
- Test infrastructure (backend >80%, frontend >70% coverage) (NFR-M1, NFR-M2)
- Database migration system (versioned, automated) (NFR-M6, NFR-M9)
- Logging infrastructure (severity levels, no sensitive data) (NFR-M11, NFR-M12)
- Performance metrics dashboard (NFR-M12, NFR-M13)

**Usability Requirements Satisfaction (9 NFRs):**

✅ **Satisfied by existing stack:**
- Docker Compose: Supports <5 minute deployment goal (NFR-U1)

⚠️ **Requires implementation:**
- Setup wizard (<5 steps) (NFR-U2)
- Configuration management with sensible defaults (NFR-U3)
- User feedback mechanisms (NFR-U4, NFR-U5, NFR-U9)
- Keyboard navigation support (NFR-U6)

## Identified Architecture Gaps

Based on requirements analysis, the following components need to be architected and implemented:

**High Priority (Affects Core Functionality):**

1. **AI Provider Abstraction Layer**
   - Purpose: Support Gemini/Claude provider switching, implement 30-day caching
   - Requirements: NFR-I9, NFR-I10, NFR-I11, NFR-I12
   - Complexity: Medium
   - Dependencies: Secrets management, cache system

2. **Multi-Source Metadata Orchestrator**
   - Purpose: Implement TMDb → Douban → Wikipedia → AI → Manual fallback chain
   - Requirements: FR15-FR20, NFR-R3, NFR-R4, NFR-I6
   - Complexity: High
   - Dependencies: All metadata source adapters, circuit breaker, caching

3. **Background Task Queue**
   - Purpose: AI parsing (10s non-blocking), auto-retry, scheduled backups
   - Requirements: FR15, FR22, FR25, NFR-R5, NFR-M9
   - Complexity: Medium
   - Dependencies: Job persistence, worker pool management

4. **Cache Management System**
   - Purpose: Multi-tier caching (metadata 24h, AI results 30d, images permanent)
   - Requirements: NFR-I7, NFR-I10, Cross-cutting concern #2
   - Complexity: Medium
   - Dependencies: Storage layer, TTL management

5. **Secrets Management Service**
   - Purpose: Environment variable priority, AES-256 encryption, zero-logging
   - Requirements: NFR-S1, NFR-S2, NFR-S3, NFR-S4, NFR-S5
   - Complexity: Medium
   - Dependencies: Encryption library, logging filters

**Medium Priority (Quality & Scalability):**

6. **Repository Pattern Implementation**
   - Purpose: SQLite and PostgreSQL interface abstraction
   - Requirements: NFR-SC3, NFR-SC10, Cross-cutting concern #5
   - Complexity: Medium
   - Dependencies: Database drivers, migration tooling

7. **Authentication Middleware**
   - Purpose: Password/PIN protection, session tokens, rate limiting
   - Requirements: NFR-S9, NFR-S10, NFR-S11, NFR-S12, NFR-S13
   - Complexity: Medium
   - Dependencies: Session storage, token generation, rate limiter

8. **Circuit Breaker Pattern**
   - Purpose: Protect external service calls, implement fallback logic
   - Requirements: NFR-R2, NFR-R6, NFR-R11, Cross-cutting concern #1
   - Complexity: Low-Medium
   - Dependencies: Health check data, timeout configuration

9. **Health Check Scheduler**
   - Purpose: Monitor qBittorrent, TMDb, AI APIs connection health
   - Requirements: NFR-I2, NFR-M13, Cross-cutting concern #4
   - Complexity: Low
   - Dependencies: HTTP client, scheduling library

10. **Performance Monitoring Dashboard**
    - Purpose: Track p95 latency, cache hit rate, API usage
    - Requirements: NFR-M12, NFR-SC2, Cross-cutting concern #4
    - Complexity: Medium
    - Dependencies: Metrics collection, aggregation, storage

**Low Priority (Developer Experience & Future Preparation):**

11. **Test Infrastructure**
    - Purpose: Unit, integration, E2E test frameworks
    - Requirements: NFR-M1, NFR-M2
    - Complexity: Medium
    - Dependencies: Testing libraries, CI/CD pipeline

12. **Database Migration System**
    - Purpose: Versioned schema changes, automated migrations
    - Requirements: NFR-M6, NFR-M9
    - Complexity: Low-Medium
    - Dependencies: Migration library (e.g., golang-migrate)

13. **i18n Framework Integration**
    - Purpose: Prepare for multi-language support (1.0 focuses on zh-TW)
    - Requirements: Cross-cutting concern #6
    - Complexity: Low
    - Dependencies: i18n library (frontend and backend)

14. **Virtual Scrolling Components**
    - Purpose: Large library performance (>1,000 items)
    - Requirements: NFR-P10, NFR-SC6, NFR-SC7
    - Complexity: Low-Medium
    - Dependencies: React virtualization library (e.g., react-window)

## Pending Architecture Decisions

**Decision Required: CSS Framework Selection**

The PRD mentions "CSS Modules or Tailwind CSS (TBD)" - this architectural decision needs to be made:

**Option A: Tailwind CSS**
- **Pros:** Utility-first rapid development, excellent Vite integration, smaller runtime overhead, atomic CSS approach reduces bundle size
- **Cons:** Steeper learning curve, verbose HTML classes, potential design consistency challenges
- **Best for:** Fast prototyping, component libraries, design system consistency

**Option B: CSS Modules**
- **Pros:** Scoped styles prevent conflicts, closer to traditional CSS workflow, more granular control
- **Cons:** More boilerplate, manual theme management, potential duplicate styles across modules
- **Best for:** Traditional CSS developers, fine-grained style control

**Option C: CSS-in-JS (styled-components / Emotion)**
- **Pros:** Dynamic styling, theme integration, component-scoped styles
- **Cons:** Runtime overhead, larger bundle size, potential performance impact
- **Best for:** Heavily dynamic UIs, complex theming requirements

**Recommendation:** Tailwind CSS
- Aligns with <500KB bundle target (NFR-P17)
- Vite has excellent Tailwind integration
- Component-based design system approach matches React architecture
- Wide community adoption and extensive documentation
- **Decision required in next architecture step**

## Technology Stack Summary

**Confirmed Technical Foundation:**

- **Backend:** Go 1.21+ with Gin framework
- **Frontend:** React 19 with TypeScript (strict mode)
- **Build Tools:** Vite (frontend), Air (backend hot reload)
- **Data Fetching:** TanStack Query v5
- **Routing:** TanStack Router
- **Database:** SQLite with WAL mode (Repository Pattern for future PostgreSQL)
- **API Documentation:** Swaggo (OpenAPI/Swagger)
- **Containerization:** Docker + Docker Compose
- **Monorepo:** Nx workspace

**Pending Decisions:**

- CSS Framework: Tailwind CSS (recommended) vs CSS Modules vs CSS-in-JS
- Testing Frameworks: Backend (Go testing + testify?) Frontend (Vitest + React Testing Library?)
- CI/CD Platform: GitHub Actions? GitLab CI? Other?

**Next Steps:**

The architectural foundations are solid and align well with PRD requirements. The next phase will focus on:
1. Making pending technology decisions (CSS framework, testing setup)
2. Designing specific architectural components (data models, API contracts, component hierarchy)
3. Establishing architectural patterns and conventions
4. Creating implementation guidelines for AI agent consistency
