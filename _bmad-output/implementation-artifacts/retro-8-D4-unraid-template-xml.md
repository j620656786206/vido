# Story retro-8-D4: Unraid Community Apps XML Template

Status: ready-for-dev

## Story

As a developer deploying Vido on Unraid,
I want an Unraid Community Apps XML template file that pre-configures ports, volumes, and environment variables,
so that Unraid users can install Vido from Community Apps with a single click.

## Acceptance Criteria

1. `unraid-template/vido.xml` exists and is valid Unraid Community Apps XML format
2. Template references the correct GHCR image: `ghcr.io/alexyu/vido:latest`
3. Port mapping: WebUI port `8080` is configurable, maps to container port `8080`
4. Three volume mappings are defined:
   - `/vido-data` → `/mnt/user/appdata/vido` (database + cache)
   - `/vido-backups` → `/mnt/user/appdata/vido/backups` (backups)
   - `/media` → user-configurable media path (read-only)
5. Essential environment variables exposed in Unraid UI with descriptions:
   - `TMDB_API_KEY` (required for metadata)
   - `TMDB_DEFAULT_LANGUAGE` (default: `zh-TW`)
   - `VIDO_PORT` (default: `8080`)
   - `GIN_MODE` (default: `release`)
   - `AI_PROVIDER` (default: `gemini`)
   - `GEMINI_API_KEY` / `CLAUDE_API_KEY` (optional)
6. Template includes `<WebUI>` tag pointing to `http://[IP]:[PORT:8080]`
7. Template includes icon URL and project description
8. Template passes XML lint validation (`xmllint --noout`)

## Tasks / Subtasks

- [ ] Task 1: Create Unraid XML template file (AC: 1, 2, 3, 4, 5, 6, 7)
  - [ ] 1.1 Create `unraid-template/vido.xml` with Unraid Community Apps XML structure
  - [ ] 1.2 Configure `<Repository>` to `ghcr.io/alexyu/vido:latest`
  - [ ] 1.3 Add `<Config>` entries for port (8080), three volumes, and env vars
  - [ ] 1.4 Add `<WebUI>` tag: `http://[IP]:[PORT:8080]`
  - [ ] 1.5 Add `<Icon>` URL (use GitHub raw URL to project icon or placeholder)
  - [ ] 1.6 Add `<Overview>` with EN description of Vido
- [ ] Task 2: Validate XML (AC: 8)
  - [ ] 2.1 Run `xmllint --noout unraid-template/vido.xml`

## Dev Notes

### Unraid Community Apps XML Format

Unraid template XMLs follow a specific schema. Key elements:

```xml
<?xml version="1.0"?>
<Container version="2">
  <Name>Vido</Name>
  <Repository>ghcr.io/alexyu/vido:latest</Repository>
  <Registry>https://github.com/alexyu/vido/pkgs/container/vido</Registry>
  <Network>bridge</Network>
  <WebUI>http://[IP]:[PORT:8080]</WebUI>
  <Icon>https://raw.githubusercontent.com/alexyu/vido/main/apps/web/public/vido-icon.png</Icon>
  <Overview>NAS media management platform...</Overview>

  <!-- Port -->
  <Config Name="WebUI Port" Target="8080" Default="8080" Mode="tcp" Type="Port" Display="always" Required="true"/>

  <!-- Volumes -->
  <Config Name="App Data" Target="/vido-data" Default="/mnt/user/appdata/vido" Mode="rw" Type="Path" Display="always" Required="true"/>
  <Config Name="Backups" Target="/vido-backups" Default="/mnt/user/appdata/vido/backups" Mode="rw" Type="Path" Display="always" Required="true"/>
  <Config Name="Media Library" Target="/media" Default="" Mode="ro" Type="Path" Display="always" Required="true"/>

  <!-- Environment Variables -->
  <Config Name="TMDb API Key" Target="TMDB_API_KEY" Default="" Mode="" Type="Variable" Display="always" Required="false"/>
</Container>
```

### Container Architecture (from retro-8-D1)

- **Single unified container**: Go API serves both `/api/v1/*` endpoints and static React frontend assets
- **Port**: `8080` only (no nginx, no multi-port)
- **Non-root user**: `vido` (UID 1000)
- **Healthcheck**: `GET /health` (built into Dockerfile)
- **Read-only root FS**: Supported in production

### Environment Variables to Expose

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `TMDB_API_KEY` | Variable | (empty) | false | TMDb API key for movie/TV metadata |
| `TMDB_DEFAULT_LANGUAGE` | Variable | `zh-TW` | false | Default metadata language |
| `GIN_MODE` | Variable | `release` | false | Gin framework mode |
| `AI_PROVIDER` | Variable | `gemini` | false | AI provider: gemini or claude |
| `GEMINI_API_KEY` | Variable | (empty) | false | Google Gemini API key |
| `CLAUDE_API_KEY` | Variable | (empty) | false | Anthropic Claude API key |

### What NOT to Expose in Unraid UI

- Database internals (`DB_WAL_*`, `DB_MAX_*`, `DB_CACHE_SIZE`) — advanced tuning, keep defaults
- `VIDO_PUBLIC_DIR` — container-internal, never user-configured
- `ENCRYPTION_KEY` — auto-generated on first run
- `VIDO_CORS_ORIGINS` — not relevant for NAS deployment
- Testing/CI variables

### Icon

If no icon exists yet, use a placeholder URL. The icon should be a PNG, ideally 512x512.

### References

- [Source: Dockerfile] — Container build stages, ports, volumes, healthcheck
- [Source: docker-compose.yml] — Volume mappings and env var passthrough
- [Source: docker-compose.prod.yml] — Production resource limits and security settings
- [Source: .env.example] — Complete environment variable documentation
- [Source: epic-8-retro-2026-03-25.md#D4] — Retro action item origin
- [Source: Unraid Community Apps template format](https://wiki.unraid.net/Docker_template_XMLs)

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

## Change Log
