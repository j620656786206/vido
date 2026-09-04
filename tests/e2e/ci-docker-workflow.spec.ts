/**
 * CI Docker Workflow Validation Tests
 *
 * Validates .github/workflows/docker.yml structure and configuration
 * against retro-8-D2 acceptance criteria. Ensures workflow config
 * stays correct as developers modify it — catches regressions like
 * action version drift, permission changes, or broken push conditions.
 *
 * These tests run offline (no Docker/GitHub needed) — pure YAML validation.
 *
 * 2026-09-05: the single `docker` job became a `build` matrix (one NATIVE
 * runner per arch) plus a `merge` job. Several tests here had pinned the
 * MECHANISM of the old shape — one step listing both platforms, a mandatory
 * `setup-qemu-action`, `needs` being undefined — and those mechanisms are
 * exactly what the split removes. They are rewritten below to assert the
 * INTENT instead (both arches get built; nothing is published from a PR; the
 * manifest is assembled from both legs), the same correction retro-m2-AI1
 * made to the GHCR-login test. A test that pins how something is done blocks
 * the fix for the thing it was protecting.
 *
 * NOTE: These tests live in tests/e2e/ alongside other Playwright tests
 * but only perform filesystem I/O. The Playwright webServer (Go + Vite)
 * will start when running the full suite. To run these alone efficiently:
 *   npx playwright test ci-docker-workflow --project=chromium
 *
 * Priority: P1 (High - run on PR to main)
 *
 * @tags @ci @p1 @validation
 */

import { test, expect } from '../support/fixtures';
import * as fs from 'fs';
import * as path from 'path';
import * as yaml from 'js-yaml';

// -- Type definitions for workflow YAML structure --

interface WorkflowStep {
  name?: string;
  uses?: string;
  run?: string;
  if?: string;
  id?: string;
  with?: Record<string, unknown>;
  'working-directory'?: string;
  env?: Record<string, string>;
}

interface WorkflowJob {
  name: string;
  needs?: string[];
  if?: string;
  'runs-on': string;
  'timeout-minutes'?: number;
  strategy?: {
    'fail-fast'?: boolean;
    matrix?: { include?: Array<Record<string, string>> };
  };
  steps: WorkflowStep[];
}

interface GHAWorkflow {
  name: string;
  on: {
    push?: { branches?: string[]; tags?: string[] };
    pull_request?: { branches?: string[] };
    workflow_dispatch?: unknown;
  };
  permissions?: Record<string, string>;
  concurrency?: { group: string; 'cancel-in-progress': boolean };
  env?: Record<string, string>;
  jobs: Record<string, WorkflowJob>;
}

// -- Helper: find step by action name prefix --

function findStepByAction(steps: WorkflowStep[], actionPrefix: string): WorkflowStep | undefined {
  return steps.find((s) => s.uses?.startsWith(actionPrefix));
}

/** Every step in the workflow, across all jobs — for whole-file invariants. */
function allSteps(wf: GHAWorkflow): WorkflowStep[] {
  return Object.values(wf.jobs).flatMap((j) => j.steps);
}

// -- Load and parse workflow YAML once --

const WORKFLOW_PATH = path.resolve(__dirname, '../../.github/workflows/docker.yml');
const TEST_WORKFLOW_PATH = path.resolve(__dirname, '../../.github/workflows/test.yml');
const GO_MOD_PATH = path.resolve(__dirname, '../../apps/api/go.mod');

let dockerWorkflow: GHAWorkflow;
let testWorkflow: GHAWorkflow;
let goModContent: string;

test.beforeAll(() => {
  dockerWorkflow = yaml.load(fs.readFileSync(WORKFLOW_PATH, 'utf-8')) as GHAWorkflow;
  testWorkflow = yaml.load(fs.readFileSync(TEST_WORKFLOW_PATH, 'utf-8')) as GHAWorkflow;
  goModContent = fs.readFileSync(GO_MOD_PATH, 'utf-8');
});

// =============================================================================
// AC1: Trigger Configuration
// =============================================================================
test.describe('Trigger Configuration @ci @validation', () => {
  test('[P1] workflow triggers on push to main branch', () => {
    // GIVEN: The docker workflow file
    // WHEN: Checking push trigger branches
    const pushBranches = dockerWorkflow.on.push?.branches;
    // THEN: main should be in the push branches
    expect(pushBranches).toContain('main');
  });

  test('[P1] workflow triggers on semver tags v*.*.*', () => {
    // GIVEN: The docker workflow file
    // WHEN: Checking push trigger tags
    const pushTags = dockerWorkflow.on.push?.tags;
    // THEN: semver tag pattern should be configured
    expect(pushTags).toContainEqual(expect.stringContaining('v*'));
  });

  test('[P1] workflow triggers on PRs to main (validation only)', () => {
    // GIVEN: The docker workflow file
    // WHEN: Checking pull_request trigger
    const prBranches = dockerWorkflow.on.pull_request?.branches;
    // THEN: PRs to main should trigger the workflow
    expect(prBranches).toContain('main');
  });

  test('[P1] workflow supports manual dispatch', () => {
    // GIVEN: The docker workflow file
    // WHEN: Checking workflow_dispatch trigger
    // THEN: workflow_dispatch should be defined
    expect(dockerWorkflow.on).toHaveProperty('workflow_dispatch');
  });
});

// =============================================================================
// AC2: Multi-Platform Build
// =============================================================================
test.describe('Multi-Platform Build @ci @validation', () => {
  test('[P1] builds for linux/amd64 and linux/arm64', () => {
    // GIVEN: The build matrix (one leg per architecture)
    const legs = (dockerWorkflow.jobs.build.strategy?.matrix?.include ?? []) as Array<
      Record<string, string>
    >;
    // WHEN: Taking the union of every leg's platform
    const platforms = legs.map((l) => l.platform);
    // THEN: Both arches must still be built — the split changed WHERE, not WHETHER
    expect(platforms).toContain('linux/amd64');
    expect(platforms).toContain('linux/arm64');
  });

  test('[P1] each arch builds on a NATIVE runner (no QEMU emulation)', () => {
    // The whole point of the 2026-09-05 split. Measured before it: arm64
    // `go build` 702s vs amd64 128s, arm64 total ~21min against a 30-min cap
    // that it kept blowing. A stray setup-qemu-action would let a mistyped
    // `platforms:` fall back to the slow emulated path SILENTLY rather than
    // failing, so its absence is the invariant worth pinning.
    expect(findStepByAction(allSteps(dockerWorkflow), 'docker/setup-qemu-action')).toBeUndefined();

    const legs = (dockerWorkflow.jobs.build.strategy?.matrix?.include ?? []) as Array<
      Record<string, string>
    >;
    for (const leg of legs) {
      const nativeArm = leg.platform === 'linux/arm64' && leg.runner.endsWith('-arm');
      const nativeAmd = leg.platform === 'linux/amd64' && !leg.runner.endsWith('-arm');
      expect(nativeArm || nativeAmd, `${leg.platform} must run on a native runner`).toBe(true);
    }
  });

  test('[P1] one leg failing does not cancel the other', () => {
    // fail-fast would leave the surviving arch's registry cache half-written,
    // so the next run rebuilds it cold — the slow path this split exists to end.
    expect(dockerWorkflow.jobs.build.strategy?.['fail-fast']).toBe(false);
  });
});

// =============================================================================
// AC3: GHCR Authentication
// =============================================================================
test.describe('GHCR Authentication @ci @validation', () => {
  test('[P1] uses GITHUB_TOKEN for GHCR login (no PAT required)', () => {
    // GIVEN: The docker job GHCR login step (may have multiple login steps)
    const dockerJob = dockerWorkflow.jobs.build;
    const ghcrLoginStep = dockerJob.steps.find(
      (s: WorkflowStep) =>
        s.uses?.startsWith('docker/login-action') && s.with?.registry === 'ghcr.io'
    );
    // WHEN: Checking login credentials
    // THEN: Should use GITHUB_TOKEN, not a PAT
    expect(ghcrLoginStep).toBeDefined();
    expect(ghcrLoginStep?.with?.password).toContain('GITHUB_TOKEN');
  });

  // retro-m2-AI1 (2026-08-07) REPLACES the former '[P1] login is skipped for
  // pull requests'. That test pinned the MECHANISM (no login on PRs) rather
  // than the intent (PRs must not publish images), and the mechanism turned
  // out to be actively harmful: the GHCR package is private (issue #178 GPL
  // constraint — it cannot be made public), so skipping login left
  // `cache-from` unauthenticated and EVERY PR build ran fully cold on both
  // arches, repeatedly blowing the 30-min job cap. Login now runs always; the
  // publish invariant is pinned directly by the sibling test below.
  test('[P1] login runs unconditionally so cache-from can authenticate', () => {
    // GIVEN: The docker job GHCR login step
    const dockerJob = dockerWorkflow.jobs.build;
    const ghcrLoginStep = dockerJob.steps.find(
      (s: WorkflowStep) =>
        s.uses?.startsWith('docker/login-action') && s.with?.registry === 'ghcr.io'
    );
    // THEN: No event gate — a private registry cache needs auth on PRs too
    expect(ghcrLoginStep?.if).toBeUndefined();
  });

  test('[P1] pull requests never publish images to GHCR', () => {
    // GIVEN: The build-push step
    // (the real safety invariant the removed login gate stood in for — and
    // one this suite never asserted directly until retro-m2-AI1)
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    // THEN: the output mode must be gated on the event being anything but a PR.
    // `push:` is gone — push-by-digest carries the push flag inside `outputs:`,
    // and a PR gets type=cacheonly (builds everything, emits nothing).
    const outputs = String(buildStep?.with?.outputs);
    expect(outputs).toContain('pull_request');
    expect(outputs).toContain('cacheonly');
    // AND the job that applies the user-facing tags must not run on a PR at all
    expect(String(dockerWorkflow.jobs.merge.if)).toContain("!= 'pull_request'");
  });
});

// =============================================================================
// AC4: Docker Metadata (OCI Labels)
// =============================================================================
test.describe('Docker Metadata @ci @validation', () => {
  test('[P1] generates semver tags (version, major.minor, major)', () => {
    // GIVEN: The metadata step
    // The merge job owns the published tag set (per-arch legs push by digest).
    const metaStep = findStepByAction(dockerWorkflow.jobs.merge.steps, 'docker/metadata-action');
    const tags = metaStep?.with?.tags as string;
    // THEN: All three semver patterns should be present
    expect(tags).toContain('type=semver,pattern={{version}}');
    expect(tags).toContain('type=semver,pattern={{major}}.{{minor}}');
    expect(tags).toContain('type=semver,pattern={{major}}');
  });

  test('[P1] generates branch ref and SHA tags', () => {
    // GIVEN: The metadata step
    // The merge job owns the published tag set (per-arch legs push by digest).
    const metaStep = findStepByAction(dockerWorkflow.jobs.merge.steps, 'docker/metadata-action');
    const tags = metaStep?.with?.tags as string;
    // THEN: Branch and SHA tags should be configured
    expect(tags).toContain('type=ref,event=branch');
    expect(tags).toMatch(/type=sha/);
  });

  test('[P1] applies OCI labels (title, description, vendor, license)', () => {
    // GIVEN: The metadata step
    // The merge job owns the published tag set (per-arch legs push by digest).
    const metaStep = findStepByAction(dockerWorkflow.jobs.merge.steps, 'docker/metadata-action');
    const labels = metaStep?.with?.labels as string;
    // THEN: Required OCI labels should be present
    expect(labels).toContain('org.opencontainers.image.title=Vido');
    expect(labels).toMatch(/org\.opencontainers\.image\.description=/);
    expect(labels).toMatch(/org\.opencontainers\.image\.vendor=/);
    expect(labels).toMatch(/org\.opencontainers\.image\.licenses=/);
  });
});

// =============================================================================
// AC5: Build Layer Caching
// =============================================================================
test.describe('Build Layer Caching @ci @validation', () => {
  test('[P1] uses GHCR registry cache (not GHA cache)', () => {
    // GIVEN: The build-push step
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    const cacheFrom = buildStep?.with?.['cache-from'] as string;
    // THEN: cache-from should use registry type
    expect(cacheFrom).toContain('type=registry');
    expect(cacheFrom).toContain('buildcache');
  });

  test('[P1] cache is scoped PER ARCH so the two legs cannot clobber each other', () => {
    // A registry cache ref is one manifest. Two jobs writing the same
    // `:buildcache` tag overwrite each other (docker/buildx#1044), so one arch
    // silently runs fully cold every time — no error, just "CI is slow again".
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    expect(String(buildStep?.with?.['cache-from'])).toContain('buildcache-${{ matrix.arch }}');
    expect(String(buildStep?.with?.['cache-to'])).toContain('matrix.arch');
  });

  test('[P1] cache-to only writes on push events (not PRs)', () => {
    // GIVEN: The build-push step
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    const cacheTo = buildStep?.with?.['cache-to'] as string;
    // THEN: cache-to should be conditional on event type
    expect(cacheTo).toContain('pull_request');
    expect(cacheTo).toContain('mode=max');
  });
});

// =============================================================================
// AC6: Docker Build Verifies Compilation (test-go removed; Dockerfile handles it)
// =============================================================================
test.describe('Docker Build Standalone @ci @validation', () => {
  test('[P1] build jobs depend on nothing (unit tests run in the Tests workflow)', () => {
    // GIVEN: The per-arch build job
    // THEN: it must not wait on a test job — compilation IS the Dockerfile's job
    expect(dockerWorkflow.jobs.build.needs).toBeUndefined();
  });

  test('[P1] the manifest is assembled from BOTH arch legs', () => {
    // The one dependency that must exist: merging before both legs have pushed
    // their digest would publish a half-populated manifest.
    expect(dockerWorkflow.jobs.merge.needs).toContain('build');
  });

  test('[P1] test-go job does not exist in docker workflow', () => {
    // GIVEN: The docker workflow jobs
    // THEN: test-go job should not be present
    expect(dockerWorkflow.jobs['test-go']).toBeUndefined();
  });
});

// =============================================================================
// AC7: Reuses Existing Dockerfile
// =============================================================================
test.describe('Dockerfile Reuse @ci @validation', () => {
  test('[P1] build uses project root context (existing Dockerfile)', () => {
    // GIVEN: The build-push step
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    // THEN: Context should be project root
    expect(buildStep?.with?.context).toBe('.');
  });
});

// =============================================================================
// AC8: Provenance and SBOM
// =============================================================================
test.describe('Provenance & SBOM @ci @validation', () => {
  test('[P1] provenance attestation is enabled with mode=max', () => {
    // GIVEN: The build-push step
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    // THEN: mode=max on published builds. It is an expression now — attestations
    // are registry objects, so a PR (which pushes nothing) never had any.
    expect(String(buildStep?.with?.provenance)).toContain('mode=max');
  });

  test('[P1] SBOM generation is enabled', () => {
    // GIVEN: The build-push step
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    // THEN: enabled on published builds (expression, same reason as provenance)
    expect(String(buildStep?.with?.sbom)).toContain("!= 'pull_request'");
  });

  test('[P1] the merge step preserves attestations and proves it', () => {
    // `docker manifest create` silently drops the unknown/unknown attestation
    // manifests; `imagetools create` carries them. The build stays green either
    // way, so the workflow asserts the merged index afterwards rather than
    // trusting it — and this test pins that the assertion exists.
    const runSteps = dockerWorkflow.jobs.merge.steps.filter((st) => st.run);
    const script = runSteps.map((st) => st.run).join('\n');
    expect(script).toContain('imagetools create');
    expect(script).not.toContain('docker manifest create');
    expect(script).toContain('attestation-manifest');
  });

  test('[P1] id-token write permission for OIDC attestations', () => {
    // GIVEN: The workflow permissions
    // THEN: id-token should be write for provenance OIDC
    expect(dockerWorkflow.permissions?.['id-token']).toBe('write');
  });
});

// =============================================================================
// AC9: Correct Action Versions
// =============================================================================
test.describe('Action Versions @ci @validation', () => {
  test('[P1] all actions use latest stable versions', () => {
    // GIVEN: The docker job steps with uses
    const actionSteps = allSteps(dockerWorkflow).filter((s) => s.uses);

    // Expected minimum versions (latest stable as of March 2026).
    // docker/setup-qemu-action is deliberately absent — see the native-runner
    // test in AC2; its presence is now a regression, not a requirement.
    const expectedVersions: Record<string, string> = {
      'actions/checkout': 'v4',
      'docker/setup-buildx-action': 'v4',
      'docker/login-action': 'v4',
      'docker/metadata-action': 'v6',
      'docker/build-push-action': 'v7',
      'actions/upload-artifact': 'v4',
      'actions/download-artifact': 'v4',
    };

    for (const [action, version] of Object.entries(expectedVersions)) {
      const step = actionSteps.find((s) => s.uses!.startsWith(action));
      // THEN: Each action should use the expected version
      expect(step, `${action} should be present`).toBeDefined();
      expect(step!.uses).toBe(`${action}@${version}`);
    }
  });
});

// =============================================================================
// Concurrency Control
// =============================================================================
test.describe('Concurrency Control @ci @validation', () => {
  test('[P1] has concurrency group to prevent duplicate builds', () => {
    // GIVEN: The workflow configuration
    // THEN: Concurrency group should be defined
    expect(dockerWorkflow.concurrency).toBeDefined();
    expect(dockerWorkflow.concurrency!.group).toContain('docker');
  });

  test('[P1] cancel-in-progress is enabled', () => {
    // GIVEN: The workflow concurrency config
    // THEN: In-progress runs should be cancelled when superseded
    expect(dockerWorkflow.concurrency!['cancel-in-progress']).toBe(true);
  });
});

// =============================================================================
// Conditional Push Logic (PR vs Push)
// =============================================================================
test.describe('Conditional Push Logic @ci @validation', () => {
  test('[P1] push is disabled for pull requests', () => {
    // GIVEN: The build-push step
    const buildStep = findStepByAction(dockerWorkflow.jobs.build.steps, 'docker/build-push-action');
    // THEN: the push flag now lives inside `outputs:` (push-by-digest), and a
    // PR resolves to type=cacheonly — everything compiles, nothing is emitted.
    const outputs = String(buildStep?.with?.outputs);
    expect(outputs).toContain("!= 'pull_request'");
    expect(outputs).toContain('cacheonly');
    // AND cache-to must stay write-gated too, or a PR could poison the cache
    expect(String(buildStep?.with?.['cache-to'])).toContain("!= 'pull_request'");
  });
});

// =============================================================================
// Permissions
// =============================================================================
test.describe('Workflow Permissions @ci @validation', () => {
  test('[P1] has minimal required permissions', () => {
    // GIVEN: The workflow permissions
    const perms = dockerWorkflow.permissions;
    // THEN: Should have exactly the required permissions
    expect(perms?.contents).toBe('read');
    expect(perms?.packages).toBe('write');
    expect(perms?.['id-token']).toBe('write');
  });
});

// =============================================================================
// Cross-Workflow Go Version Consistency
// =============================================================================
test.describe('Go Version Consistency @ci @validation', () => {
  test('[P1] test.yml GO_VERSION matches go.mod', () => {
    // GIVEN: go.mod specifies a Go version
    const goModVersion = goModContent.match(/^go\s+(\d+\.\d+)/m)?.[1];
    // WHEN: Checking test.yml env
    const testGoVersion = testWorkflow.env?.GO_VERSION;
    // THEN: Versions should match
    expect(testGoVersion).toBe(goModVersion);
  });
});
