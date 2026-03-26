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
 * Priority: P1 (High - run on PR to main)
 *
 * @tags @ci @p1 @validation
 */

import { test, expect } from '../support/fixtures';
import * as fs from 'fs';
import * as path from 'path';
import * as yaml from 'js-yaml';

// Load and parse the workflow YAML once
const WORKFLOW_PATH = path.resolve(__dirname, '../../.github/workflows/docker.yml');
const TEST_WORKFLOW_PATH = path.resolve(__dirname, '../../.github/workflows/test.yml');
const GO_MOD_PATH = path.resolve(__dirname, '../../apps/api/go.mod');

let dockerWorkflow: Record<string, any>;
let testWorkflow: Record<string, any>;
let goModContent: string;

test.beforeAll(() => {
  dockerWorkflow = yaml.load(fs.readFileSync(WORKFLOW_PATH, 'utf-8')) as Record<string, any>;
  testWorkflow = yaml.load(fs.readFileSync(TEST_WORKFLOW_PATH, 'utf-8')) as Record<string, any>;
  goModContent = fs.readFileSync(GO_MOD_PATH, 'utf-8');
});

// =============================================================================
// AC1: Trigger Configuration
// =============================================================================
test.describe('Trigger Configuration @ci @validation', () => {
  test('[P1] workflow triggers on push to main branch', () => {
    // GIVEN: The docker workflow file
    // WHEN: Checking push trigger branches
    const pushBranches = dockerWorkflow.on.push.branches;
    // THEN: main should be in the push branches
    expect(pushBranches).toContain('main');
  });

  test('[P1] workflow triggers on semver tags v*.*.*', () => {
    // GIVEN: The docker workflow file
    // WHEN: Checking push trigger tags
    const pushTags = dockerWorkflow.on.push.tags;
    // THEN: semver tag pattern should be configured
    expect(pushTags).toContainEqual(expect.stringContaining('v*'));
  });

  test('[P1] workflow triggers on PRs to main (validation only)', () => {
    // GIVEN: The docker workflow file
    // WHEN: Checking pull_request trigger
    const prBranches = dockerWorkflow.on.pull_request.branches;
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
    // GIVEN: The docker job build-push step
    const dockerJob = dockerWorkflow.jobs.docker;
    const buildStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/build-push-action')
    );
    // WHEN: Checking platforms configuration
    const platforms = buildStep.with.platforms;
    // THEN: Both amd64 and arm64 should be included
    expect(platforms).toContain('linux/amd64');
    expect(platforms).toContain('linux/arm64');
  });

  test('[P1] QEMU action is configured for ARM64 emulation', () => {
    // GIVEN: The docker job steps
    const dockerJob = dockerWorkflow.jobs.docker;
    const qemuStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/setup-qemu-action')
    );
    // THEN: QEMU setup step should exist
    expect(qemuStep).toBeDefined();
  });
});

// =============================================================================
// AC3: GHCR Authentication
// =============================================================================
test.describe('GHCR Authentication @ci @validation', () => {
  test('[P1] uses GITHUB_TOKEN for GHCR login (no PAT required)', () => {
    // GIVEN: The docker job login step
    const dockerJob = dockerWorkflow.jobs.docker;
    const loginStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/login-action')
    );
    // WHEN: Checking login credentials
    // THEN: Should use GITHUB_TOKEN, not a PAT
    expect(loginStep.with.registry).toBe('ghcr.io');
    expect(loginStep.with.password).toContain('GITHUB_TOKEN');
  });

  test('[P1] login is skipped for pull requests', () => {
    // GIVEN: The docker job login step
    const dockerJob = dockerWorkflow.jobs.docker;
    const loginStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/login-action')
    );
    // THEN: Login should be conditional (skip on PR)
    expect(loginStep.if).toContain('pull_request');
  });
});

// =============================================================================
// AC4: Docker Metadata (OCI Labels)
// =============================================================================
test.describe('Docker Metadata @ci @validation', () => {
  test('[P1] generates semver tags (version, major.minor, major)', () => {
    // GIVEN: The metadata step
    const dockerJob = dockerWorkflow.jobs.docker;
    const metaStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/metadata-action')
    );
    const tags = metaStep.with.tags;
    // THEN: All three semver patterns should be present
    expect(tags).toContain('type=semver,pattern={{version}}');
    expect(tags).toContain('type=semver,pattern={{major}}.{{minor}}');
    expect(tags).toContain('type=semver,pattern={{major}}');
  });

  test('[P1] generates branch ref and SHA tags', () => {
    // GIVEN: The metadata step
    const dockerJob = dockerWorkflow.jobs.docker;
    const metaStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/metadata-action')
    );
    const tags = metaStep.with.tags;
    // THEN: Branch and SHA tags should be configured
    expect(tags).toContain('type=ref,event=branch');
    expect(tags).toMatch(/type=sha/);
  });

  test('[P1] applies OCI labels (title, description, vendor, license)', () => {
    // GIVEN: The metadata step
    const dockerJob = dockerWorkflow.jobs.docker;
    const metaStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/metadata-action')
    );
    const labels = metaStep.with.labels;
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
    const dockerJob = dockerWorkflow.jobs.docker;
    const buildStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/build-push-action')
    );
    // THEN: cache-from should use registry type
    expect(buildStep.with['cache-from']).toContain('type=registry');
    expect(buildStep.with['cache-from']).toContain('buildcache');
  });

  test('[P1] cache-to only writes on push events (not PRs)', () => {
    // GIVEN: The build-push step
    const dockerJob = dockerWorkflow.jobs.docker;
    const buildStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/build-push-action')
    );
    // THEN: cache-to should be conditional on event type
    const cacheTo = buildStep.with['cache-to'];
    expect(cacheTo).toContain('pull_request');
    expect(cacheTo).toContain('mode=max');
  });
});

// =============================================================================
// AC6: Go Backend Tests as Prerequisite
// =============================================================================
test.describe('Go Test Prerequisite @ci @validation', () => {
  test('[P1] test-go job runs before docker build', () => {
    // GIVEN: The docker job dependencies
    const dockerJob = dockerWorkflow.jobs.docker;
    // THEN: Docker job should depend on test-go
    expect(dockerJob.needs).toContain('test-go');
  });

  test('[P1] test-go job runs go test ./... in apps/api', () => {
    // GIVEN: The test-go job
    const testGoJob = dockerWorkflow.jobs['test-go'];
    const testStep = testGoJob.steps.find((s: any) => s.run && s.run.includes('go test'));
    // THEN: Should run go test in apps/api directory
    expect(testStep).toBeDefined();
    expect(testStep.run).toContain('go test ./...');
    expect(testStep['working-directory']).toBe('apps/api');
  });
});

// =============================================================================
// AC7: Reuses Existing Dockerfile
// =============================================================================
test.describe('Dockerfile Reuse @ci @validation', () => {
  test('[P1] build uses project root context (existing Dockerfile)', () => {
    // GIVEN: The build-push step
    const dockerJob = dockerWorkflow.jobs.docker;
    const buildStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/build-push-action')
    );
    // THEN: Context should be project root
    expect(buildStep.with.context).toBe('.');
  });
});

// =============================================================================
// AC8: Provenance and SBOM
// =============================================================================
test.describe('Provenance & SBOM @ci @validation', () => {
  test('[P1] provenance attestation is enabled with mode=max', () => {
    // GIVEN: The build-push step
    const dockerJob = dockerWorkflow.jobs.docker;
    const buildStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/build-push-action')
    );
    // THEN: Provenance should be mode=max
    expect(buildStep.with.provenance).toBe('mode=max');
  });

  test('[P1] SBOM generation is enabled', () => {
    // GIVEN: The build-push step
    const dockerJob = dockerWorkflow.jobs.docker;
    const buildStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/build-push-action')
    );
    // THEN: SBOM should be true
    expect(buildStep.with.sbom).toBe(true);
  });

  test('[P1] id-token write permission for OIDC attestations', () => {
    // GIVEN: The workflow permissions
    // THEN: id-token should be write for provenance OIDC
    expect(dockerWorkflow.permissions['id-token']).toBe('write');
  });
});

// =============================================================================
// AC9: Correct Action Versions
// =============================================================================
test.describe('Action Versions @ci @validation', () => {
  test('[P1] all actions use latest stable versions', () => {
    // GIVEN: The docker job steps with uses
    const allSteps = [...dockerWorkflow.jobs['test-go'].steps, ...dockerWorkflow.jobs.docker.steps];
    const actionSteps = allSteps.filter((s: any) => s.uses);

    // Expected minimum versions (latest stable as of March 2026)
    const expectedVersions: Record<string, string> = {
      'actions/checkout': 'v4',
      'actions/setup-go': 'v5',
      'docker/setup-qemu-action': 'v4',
      'docker/setup-buildx-action': 'v4',
      'docker/login-action': 'v4',
      'docker/metadata-action': 'v6',
      'docker/build-push-action': 'v7',
    };

    for (const [action, version] of Object.entries(expectedVersions)) {
      const step = actionSteps.find((s: any) => s.uses.startsWith(action));
      // THEN: Each action should use the expected version
      expect(step, `${action} should be present`).toBeDefined();
      expect(step.uses).toBe(`${action}@${version}`);
    }
  });
});

// =============================================================================
// Conditional Push Logic (PR vs Push)
// =============================================================================
test.describe('Conditional Push Logic @ci @validation', () => {
  test('[P1] push is disabled for pull requests', () => {
    // GIVEN: The build-push step
    const dockerJob = dockerWorkflow.jobs.docker;
    const buildStep = dockerJob.steps.find(
      (s: any) => s.uses && s.uses.startsWith('docker/build-push-action')
    );
    // THEN: Push should be conditional (false for PRs)
    const pushExpr = buildStep.with.push;
    expect(pushExpr).toContain('pull_request');
    expect(pushExpr).toContain("!= 'pull_request'");
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
    expect(perms.contents).toBe('read');
    expect(perms.packages).toBe('write');
    expect(perms['id-token']).toBe('write');
  });
});

// =============================================================================
// Cross-Workflow Go Version Consistency
// =============================================================================
test.describe('Go Version Consistency @ci @validation', () => {
  test('[P1] docker.yml GO_VERSION matches go.mod', () => {
    // GIVEN: go.mod specifies a Go version
    const goModVersion = goModContent.match(/^go\s+(\d+\.\d+)/m)?.[1];
    // WHEN: Checking docker.yml env
    const dockerGoVersion = dockerWorkflow.env.GO_VERSION;
    // THEN: Versions should match
    expect(dockerGoVersion).toBe(goModVersion);
  });

  test('[P1] test.yml GO_VERSION matches go.mod', () => {
    // GIVEN: go.mod specifies a Go version
    const goModVersion = goModContent.match(/^go\s+(\d+\.\d+)/m)?.[1];
    // WHEN: Checking test.yml env
    const testGoVersion = testWorkflow.env.GO_VERSION;
    // THEN: Versions should match
    expect(testGoVersion).toBe(goModVersion);
  });

  test('[P1] docker.yml and test.yml GO_VERSION are identical', () => {
    // GIVEN: Both workflow files
    // THEN: GO_VERSION should be consistent across workflows
    expect(dockerWorkflow.env.GO_VERSION).toBe(testWorkflow.env.GO_VERSION);
  });
});
