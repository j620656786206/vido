/**
 * Playwright Global Teardown
 *
 * Cleans up processes tracked by the current session.
 * Only kills processes that were spawned by THIS session.
 */

import { FullConfig } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';
import { getSessionFilePath, readSession, TestSession } from './global-setup';

/**
 * Check if a process is still running
 */
function isProcessRunning(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

/**
 * Get all child PIDs of a given PID (macOS/Linux)
 */
function getChildPids(parentPid: number): number[] {
  try {
    // Use pgrep to find child processes
    const result = execSync(`pgrep -P ${parentPid} 2>/dev/null || true`, {
      encoding: 'utf-8',
    });
    return result
      .trim()
      .split('\n')
      .filter((line) => line.trim())
      .map((pid) => parseInt(pid, 10))
      .filter((pid) => !isNaN(pid));
  } catch {
    return [];
  }
}

/**
 * Recursively get all descendant PIDs
 */
function getAllDescendants(pid: number, visited = new Set<number>()): number[] {
  if (visited.has(pid)) return [];
  visited.add(pid);

  const children = getChildPids(pid);
  const descendants: number[] = [...children];

  for (const child of children) {
    descendants.push(...getAllDescendants(child, visited));
  }

  return descendants;
}

/**
 * Kill a process tree gracefully (SIGTERM first, then SIGKILL)
 */
function killProcessTree(pid: number): void {
  const descendants = getAllDescendants(pid);

  // Kill children first (reverse order - deepest first)
  for (const childPid of descendants.reverse()) {
    try {
      if (isProcessRunning(childPid)) {
        process.kill(childPid, 'SIGTERM');
      }
    } catch {
      // Process might have already exited
    }
  }

  // Give processes time to exit gracefully
  const waitForExit = (pids: number[], timeout: number): Promise<void> => {
    return new Promise((resolve) => {
      const start = Date.now();
      const check = () => {
        const stillRunning = pids.filter((p) => isProcessRunning(p));
        if (stillRunning.length === 0 || Date.now() - start > timeout) {
          resolve();
        } else {
          setTimeout(check, 100);
        }
      };
      check();
    });
  };

  // Wait up to 2 seconds for graceful shutdown
  setTimeout(async () => {
    await waitForExit(descendants, 2000);

    // Force kill any remaining processes
    for (const childPid of descendants) {
      try {
        if (isProcessRunning(childPid)) {
          process.kill(childPid, 'SIGKILL');
        }
      } catch {
        // Process might have already exited
      }
    }
  }, 0);
}

/**
 * Find orphaned test server processes
 */
function findOrphanedTestProcesses(session: TestSession): number[] {
  const orphans: number[] = [];

  try {
    // Find Go backend processes started by this session
    const goProcs = execSync(
      `pgrep -f "go run.*cmd/api" 2>/dev/null || true`,
      { encoding: 'utf-8' }
    )
      .trim()
      .split('\n')
      .filter((line) => line.trim())
      .map((pid) => parseInt(pid, 10))
      .filter((pid) => !isNaN(pid));

    // Find Vite dev server processes
    const viteProcs = execSync(
      `pgrep -f "vite.*serve|nx.*serve.*web" 2>/dev/null || true`,
      { encoding: 'utf-8' }
    )
      .trim()
      .split('\n')
      .filter((line) => line.trim())
      .map((pid) => parseInt(pid, 10))
      .filter((pid) => !isNaN(pid));

    // Only include processes that are descendants of our session
    const sessionDescendants = new Set<number>();
    for (const pid of session.pids) {
      getAllDescendants(pid).forEach((d) => sessionDescendants.add(d));
    }

    for (const pid of [...goProcs, ...viteProcs]) {
      if (sessionDescendants.has(pid) || session.pids.includes(pid)) {
        orphans.push(pid);
      }
    }
  } catch {
    // Ignore errors
  }

  return orphans;
}

/**
 * Clean up stale session files (older than 1 hour)
 */
function cleanupStaleSessions(): void {
  const sessionDir = path.dirname(getSessionFilePath());
  if (!fs.existsSync(sessionDir)) return;

  const oneHourAgo = Date.now() - 60 * 60 * 1000;

  try {
    const files = fs.readdirSync(sessionDir);
    for (const file of files) {
      const filePath = path.join(sessionDir, file);
      try {
        const session: TestSession = JSON.parse(fs.readFileSync(filePath, 'utf-8'));
        // Remove stale sessions where parent process is dead
        if (session.startTime < oneHourAgo || !isProcessRunning(session.parentPid)) {
          fs.unlinkSync(filePath);
        }
      } catch {
        // If we can't read it, it's probably stale
        fs.unlinkSync(filePath);
      }
    }
  } catch {
    // Ignore cleanup errors
  }
}

/**
 * Global teardown function
 */
async function globalTeardown(config: FullConfig): Promise<void> {
  const session = readSession();
  const sessionFile = getSessionFilePath();

  if (!session) {
    console.log('\n⚠️  No session file found, skipping cleanup\n');
    return;
  }

  console.log(`\n🧹 Cleaning up test session: ${session.sessionId}`);

  // Find orphaned test processes from this session
  const orphans = findOrphanedTestProcesses(session);

  if (orphans.length > 0) {
    console.log(`   Found ${orphans.length} orphaned process(es) to clean up`);
    for (const pid of orphans) {
      try {
        if (isProcessRunning(pid)) {
          console.log(`   Killing PID ${pid}`);
          killProcessTree(pid);
        }
      } catch (err) {
        console.log(`   Failed to kill PID ${pid}: ${err}`);
      }
    }
  } else {
    console.log('   No orphaned processes found');
  }

  // Remove session file
  try {
    if (fs.existsSync(sessionFile)) {
      fs.unlinkSync(sessionFile);
      console.log(`   Removed session file`);
    }
  } catch {
    // Ignore
  }

  // Clean up stale sessions from other runs
  cleanupStaleSessions();

  console.log('   Cleanup complete\n');
}

export default globalTeardown;
