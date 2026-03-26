# Story retro-8-D2: CI Docker Build + Push to GHCR

Status: review

## Story

As a developer deploying Vido to my NAS,
I want a GitHub Actions workflow that automatically builds and pushes the unified Docker image to GHCR on every release tag,
so that I can pull versioned images directly without building locally.

## Acceptance Criteria

1. A new GitHub Actions workflow file `.github/workflows/docker.yml` exists and runs on:
   - Push to `main` branch (builds image tagged `main`)
   - Push of semver tags `v*.*.*` (builds image tagged `1.2.3`, `1.2`, `1`, `latest`)
   - Pull requests to `main` (builds but does NOT push — validation only)
   - Manual `workflow_dispatch` trigger
2. The workflow builds multi-platform images: `linux/amd64` and `linux/arm64` (NAS devices like Synology/QNAP use ARM64)
3. Images are pushed to `ghcr.io/<owner>/vido` using `GITHUB_TOKEN` (no PAT required)
4. Docker metadata (OCI labels) is automatically applied: title, description, version, source URL, created timestamp, license
5. Build layer caching uses GHCR registry cache (`type=registry,ref=ghcr.io/<owner>/vido:buildcache,mode=max`) to speed up subsequent builds
6. Go backend tests (`go test ./...`) pass before Docker build starts (fail-fast)
7. The workflow reuses the existing unified `Dockerfile` at project root (created in retro-8-D1)
8. Images include provenance attestation (`provenance: mode=max`) and SBOM (`sbom: true`)
9. The workflow file passes `actionlint` validation (proper YAML syntax, correct action versions)
10. Workflow completes in under 15 minutes for cached builds

## Tasks / Subtasks

- [x] Task 1: Create `.github/workflows/docker.yml` workflow file (AC: 1, 9)
  - [x] 1.1 Define trigger events: `push` (main + v* tags), `pull_request` (main), `workflow_dispatch`
  - [x] 1.2 Set `permissions: contents: read, packages: write, id-token: write`
  - [x] 1.3 Set `env: GO_VERSION: '1.25'` (match `apps/api/go.mod`)
- [x] Task 2: Add Go backend test job as prerequisite (AC: 6)
  - [x] 2.1 Job `test-go`: checkout, setup-go@v5, `go test ./...` in `apps/api/`
  - [x] 2.2 Use `cache-dependency-path: apps/api/go.sum` for Go module caching
- [x] Task 3: Add Docker build+push job (AC: 2, 3, 4, 5, 7, 8)
  - [x] 3.1 `needs: [test-go]` — only build after tests pass
  - [x] 3.2 `actions/checkout@v4`
  - [x] 3.3 `docker/setup-qemu-action@v4` — enable ARM64 emulation
  - [x] 3.4 `docker/setup-buildx-action@v4` — enable BuildKit
  - [x] 3.5 `docker/login-action@v4` — login to `ghcr.io` with `GITHUB_TOKEN`
  - [x] 3.6 `docker/metadata-action@v6` — generate tags and OCI labels
  - [x] 3.7 `docker/build-push-action@v7` — build, push (except PRs), cache, provenance, SBOM
- [x] Task 4: Configure Docker metadata tags (AC: 4)
  - [x] 4.1 `type=semver,pattern={{version}}` — e.g., `1.2.3`
  - [x] 4.2 `type=semver,pattern={{major}}.{{minor}}` — e.g., `1.2`
  - [x] 4.3 `type=semver,pattern={{major}}` — e.g., `1`
  - [x] 4.4 `type=ref,event=branch` — e.g., `main`
  - [x] 4.5 `type=sha,prefix=sha-,format=short` — e.g., `sha-a1b2c3d`
  - [x] 4.6 OCI labels: `org.opencontainers.image.title=Vido`, description, vendor, licenses=MIT
- [x] Task 5: Update existing `test.yml` GO_VERSION (AC: 6)
  - [x] 5.1 Change `GO_VERSION: '1.24'` to `GO_VERSION: '1.25'` in `.github/workflows/test.yml` to match `go.mod`
- [x] Task 6: Verify workflow (AC: 9, 10)
  - [x] 6.1 YAML lint the workflow file (validate syntax)
  - [x] 6.2 Verify all action versions are latest stable (checkout@v4, setup-go@v5, etc.)
  - [x] 6.3 Verify conditional push logic: `push: ${{ github.event_name != 'pull_request' }}`

## Dev Notes

### Prerequisite: retro-8-D1 (DONE)

The unified `Dockerfile` at project root is already created and verified (57.1MB image). This story ONLY creates the CI workflow — do NOT modify the Dockerfile.

### Existing CI: `.github/workflows/test.yml`

The project already has a test pipeline with lint, unit tests, build, and E2E sharded tests. The Docker workflow is a **separate** workflow file — do NOT merge into `test.yml`. The two workflows run independently.

**Known issue in test.yml:** `GO_VERSION: '1.24'` but `go.mod` says `go 1.25.0`. Fix this as part of Task 5.

### Action Versions (Latest Stable as of March 2026)

| Action | Version |
|--------|---------|
| `actions/checkout` | `v4` |
| `actions/setup-go` | `v5` |
| `docker/setup-qemu-action` | `v4` |
| `docker/setup-buildx-action` | `v4` |
| `docker/login-action` | `v4` |
| `docker/metadata-action` | `v6` |
| `docker/build-push-action` | `v7` |

### GHCR Authentication

Use `GITHUB_TOKEN` (automatic, no setup needed):
```yaml
- uses: docker/login-action@v4
  with:
    registry: ghcr.io
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}
```

Requires `permissions: packages: write` in workflow.

### Multi-Platform: QEMU Emulation

For NAS deployment (Synology, QNAP, Unraid), ARM64 support is essential. QEMU emulation is acceptable for build times — the Go binary and Node build are the bottlenecks, not the runtime stage.

```yaml
platforms: linux/amd64,linux/arm64
```

### Registry Cache Strategy

Use GHCR registry cache (NOT GitHub Actions cache) — it supports `mode=max` for all intermediate layers and works across branches/PRs. GHA cache has a 10GB repo-wide cap that fills quickly with multi-platform builds.

```yaml
cache-from: type=registry,ref=ghcr.io/${{ github.repository }}:buildcache
cache-to: ${{ github.event_name != 'pull_request' && format('type=registry,ref=ghcr.io/{0}:buildcache,mode=max', github.repository) || '' }}
```

**Important:** Only write cache on push (not PRs) to avoid cache pollution.

### Provenance and SBOM

```yaml
provenance: mode=max
sbom: true
```

Requires `permissions: id-token: write` for OIDC attestations. Do NOT pass secrets as `--build-arg` (provenance exposes build args).

### Conditional Push Logic

- **Push events** (main branch, tags): Build AND push to GHCR
- **Pull requests**: Build only (validation), NO push
- **Rationale:** PRs verify the Dockerfile builds correctly without polluting the registry

```yaml
push: ${{ github.event_name != 'pull_request' }}
```

### Project Structure Notes

- New file: `.github/workflows/docker.yml` (alongside existing `test.yml`)
- Modified file: `.github/workflows/test.yml` (Go version fix only)
- No Dockerfile changes — reuse existing unified Dockerfile from retro-8-D1
- No backend/frontend code changes

### References

- [Source: Dockerfile] — Unified 3-stage multi-stage build (retro-8-D1)
- [Source: .github/workflows/test.yml] — Existing CI pipeline pattern
- [Source: apps/api/go.mod] — Go 1.25.0
- [Source: .nvmrc] — Node LTS Iron (v20)
- [Source: epic-8-retro-2026-03-25.md#D2] — Retro action item origin
- [Source: docker/build-push-action docs] — Latest v7 patterns
- [Source: docker/metadata-action docs] — v6 tag generation

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (1M context)

### Debug Log References

### Completion Notes List

- Task 1-4: Created `.github/workflows/docker.yml` with 2-job pipeline (test-go → docker build+push). Triggers on push to main, semver tags, PRs, and manual dispatch. Multi-platform amd64+arm64 via QEMU. GHCR login with GITHUB_TOKEN. Metadata tags: semver (version/major.minor/major), branch ref, sha prefix. OCI labels: title, description, vendor, licenses. Registry cache with mode=max (write-on-push only). Provenance mode=max + SBOM enabled.
- Task 5: Fixed `test.yml` GO_VERSION from `1.24` to `1.25` to match `go.mod` (go 1.25.0).
- Task 6: actionlint validation PASS on docker.yml. All 7 action versions verified as latest stable. Conditional push logic confirmed correct.
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### File List

- `.github/workflows/docker.yml` (NEW) — Docker build+push workflow
- `.github/workflows/test.yml` (MODIFIED) — GO_VERSION 1.24 → 1.25
- `_bmad-output/implementation-artifacts/retro-8-D2-ci-docker-build-push-ghcr.md` (MODIFIED) — Story status + task checkboxes + dev record
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (MODIFIED) — Story status ready-for-dev → review

## Change Log

- 2026-03-26: Implemented CI Docker build+push workflow for GHCR with multi-platform support, registry caching, provenance, and SBOM. Fixed GO_VERSION mismatch in test.yml.
