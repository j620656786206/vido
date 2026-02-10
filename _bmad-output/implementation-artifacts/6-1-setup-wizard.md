# Story 6.1: Setup Wizard

Status: ready-for-dev

## Story

As a **first-time user**,
I want a **guided setup wizard**,
So that **I can configure Vido quickly without confusion**.

## Acceptance Criteria

1. **Given** the user opens Vido for the first time, **When** no configuration exists, **Then** the setup wizard launches automatically and shows progress: "Step 1 of 5"
2. **Given** the wizard is running, **When** completing each step (Welcome & language, qBittorrent connection, Media folder config, API keys, Complete), **Then** each step validates before proceeding and back navigation is available
3. **Given** the user completes the wizard, **When** clicking "Finish", **Then** settings are saved and the user is taken to the main dashboard
4. **Given** the wizard step for qBittorrent, **When** the user doesn't have qBittorrent, **Then** a skip option is available
5. **Given** the wizard step for API keys, **When** the user doesn't have API keys yet, **Then** a skip option is available with explanation that some features will be limited

## Tasks / Subtasks

- [ ] Task 1: Create Setup Detection Service (AC: 1)
  - [ ] 1.1: Create `/apps/api/internal/services/setup_service.go` with `SetupServiceInterface`
  - [ ] 1.2: Add `IsFirstRun(ctx) (bool, error)` method - checks if `settings` table has `setup_completed` flag
  - [ ] 1.3: Add `CompleteSetup(ctx, config SetupConfig) error` method - saves all wizard settings
  - [ ] 1.4: Write unit tests `setup_service_test.go` (≥80% coverage)

- [ ] Task 2: Create Setup API Endpoints (AC: 1, 3)
  - [ ] 2.1: Create `/apps/api/internal/handlers/setup_handler.go`
  - [ ] 2.2: `GET /api/v1/setup/status` → returns `{ needsSetup: bool }`
  - [ ] 2.3: `POST /api/v1/setup/complete` → accepts setup config, saves settings
  - [ ] 2.4: `POST /api/v1/setup/validate-step` → validates individual step data
  - [ ] 2.5: Write handler tests `setup_handler_test.go` (≥70% coverage)

- [ ] Task 3: Create Setup Wizard Frontend Route (AC: 1)
  - [ ] 3.1: Create `/apps/web/src/routes/setup.tsx` (TanStack Router)
  - [ ] 3.2: Add route guard in `__root.tsx` - redirect to `/setup` if `needsSetup` is true
  - [ ] 3.3: Create `/apps/web/src/hooks/useSetupStatus.ts` - TanStack Query hook for setup status

- [ ] Task 4: Create Wizard Step Components (AC: 2)
  - [ ] 4.1: Create `/apps/web/src/components/setup/SetupWizard.tsx` - main wizard container with step progress
  - [ ] 4.2: Create `/apps/web/src/components/setup/WelcomeStep.tsx` - language selection step
  - [ ] 4.3: Create `/apps/web/src/components/setup/QBittorrentStep.tsx` - qBittorrent connection with skip option
  - [ ] 4.4: Create `/apps/web/src/components/setup/MediaFolderStep.tsx` - media folder path configuration
  - [ ] 4.5: Create `/apps/web/src/components/setup/ApiKeysStep.tsx` - optional API keys (TMDb, AI) with skip
  - [ ] 4.6: Create `/apps/web/src/components/setup/CompleteStep.tsx` - summary and finish

- [ ] Task 5: Implement Wizard Navigation (AC: 2)
  - [ ] 5.1: Create `/apps/web/src/components/setup/StepProgress.tsx` - visual step indicator (1/5 dots)
  - [ ] 5.2: Implement forward/back navigation with step validation
  - [ ] 5.3: Persist wizard state in React state (lost on page refresh is OK)

- [ ] Task 6: Implement Step Validation (AC: 2, 4, 5)
  - [ ] 6.1: Validate qBittorrent connection by calling test endpoint
  - [ ] 6.2: Validate media folder path exists (backend check)
  - [ ] 6.3: Validate API key format (TMDb key format check)
  - [ ] 6.4: Show validation errors using form error patterns from UX spec

- [ ] Task 7: Complete Setup Flow (AC: 3)
  - [ ] 7.1: Create `/apps/web/src/services/setupService.ts` - API client
  - [ ] 7.2: On finish, POST all settings to backend
  - [ ] 7.3: Invalidate setup status query and redirect to dashboard
  - [ ] 7.4: Write component tests for SetupWizard and individual steps

- [ ] Task 8: Wire Up in main.go (AC: all)
  - [ ] 8.1: Register SetupService and SetupHandler in dependency injection
  - [ ] 8.2: Register setup routes in Gin router

## Dev Notes

### Architecture Requirements

**FR52: Initial setup via wizard**
- Setup wizard <5 steps (NFR-U2)
- Minimal onboarding (UX-7)
- Sensible defaults requiring zero manual configuration for basic usage (NFR-U3)

### Existing Codebase Context

**Settings already exist:** The `settings` table was created in migration `003_create_settings_table.go`. Use existing `repository.SettingsRepository` for persisting setup state.

**Config module:** `/apps/api/internal/config/` handles environment variables. The wizard should save to both settings table AND environment config.

**qBittorrent connection:** Will be implemented in Epic 4 stories. For the wizard, create a simple connection test endpoint that validates URL + credentials.

### Backend Implementation

```go
// /apps/api/internal/services/setup_service.go
package services

type SetupConfig struct {
    Language         string `json:"language"`          // "zh-TW"
    QBTUrl           string `json:"qbtUrl,omitempty"`  // optional
    QBTUsername      string `json:"qbtUsername,omitempty"`
    QBTPassword      string `json:"qbtPassword,omitempty"`
    MediaFolderPath  string `json:"mediaFolderPath"`
    TMDbApiKey       string `json:"tmdbApiKey,omitempty"`
    AIProvider       string `json:"aiProvider,omitempty"`
    AIApiKey         string `json:"aiApiKey,omitempty"`
}

type SetupServiceInterface interface {
    IsFirstRun(ctx context.Context) (bool, error)
    CompleteSetup(ctx context.Context, config SetupConfig) error
    ValidateStep(ctx context.Context, step string, data map[string]interface{}) error
}
```

### Frontend Implementation

```tsx
// /apps/web/src/components/setup/SetupWizard.tsx
interface WizardStep {
  id: string;
  title: string;
  component: React.ComponentType<StepProps>;
  optional?: boolean;
  validate?: (data: StepData) => Promise<string | null>;
}

const WIZARD_STEPS: WizardStep[] = [
  { id: 'welcome', title: '歡迎', component: WelcomeStep },
  { id: 'qbittorrent', title: 'qBittorrent', component: QBittorrentStep, optional: true },
  { id: 'media-folder', title: '媒體資料夾', component: MediaFolderStep },
  { id: 'api-keys', title: 'API 金鑰', component: ApiKeysStep, optional: true },
  { id: 'complete', title: '完成', component: CompleteStep },
];
```

### UX Patterns

- Use Form Patterns from UX spec for input fields (bg-secondary, border-border, rounded-lg)
- Step progress indicator: dots with active state using `--accent` color
- Skip option: secondary/ghost button variant
- Validation: real-time on blur, error display below fields with `role="alert"`
- Tailwind CSS utility classes for all styling

### API Response Format

```json
// GET /api/v1/setup/status
{
  "success": true,
  "data": {
    "needsSetup": true,
    "currentStep": 0
  }
}

// POST /api/v1/setup/complete
{
  "success": true,
  "data": {
    "message": "Setup completed successfully"
  }
}
```

### Error Codes

- `SETUP_ALREADY_COMPLETED` - Setup wizard already completed
- `SETUP_VALIDATION_FAILED` - Step validation failed
- `SETUP_QBT_CONNECTION_FAILED` - Cannot connect to qBittorrent
- `SETUP_MEDIA_FOLDER_NOT_FOUND` - Media folder path doesn't exist

### Project Structure Notes

```
/apps/api/internal/services/
├── setup_service.go
└── setup_service_test.go

/apps/api/internal/handlers/
├── setup_handler.go
└── setup_handler_test.go

/apps/web/src/routes/
└── setup.tsx

/apps/web/src/components/setup/
├── SetupWizard.tsx
├── SetupWizard.spec.tsx
├── StepProgress.tsx
├── WelcomeStep.tsx
├── QBittorrentStep.tsx
├── MediaFolderStep.tsx
├── ApiKeysStep.tsx
├── CompleteStep.tsx
└── index.ts
```

### Testing Strategy

**Backend:** Setup service tests (mock settings repo), handler tests (mock service)
**Frontend:** Wizard navigation tests, step validation tests, skip flow tests
**Coverage Targets:** Services ≥80%, Handlers ≥70%, Components ≥70%

### Dependencies

- Story 1-3 (Environment Variable Configuration) - config system
- Story 1-5 (Media Folder Configuration) - media folder validation
- Settings table (migration 003)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-6.1]
- [Source: _bmad-output/planning-artifacts/prd.md#FR52]
- [Source: _bmad-output/planning-artifacts/prd.md#NFR-U2]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Form-Patterns]
- [Source: project-context.md#Rule-4-Layered-Architecture]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
