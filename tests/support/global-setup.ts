/**
 * Playwright Global Setup
 *
 * Creates a session-specific tracking file for process cleanup.
 * Each test session gets a unique ID based on the parent process PID.
 */

import { FullConfig } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as crypto from 'crypto';

// Session tracking directory
const SESSION_DIR = path.join(process.cwd(), 'node_modules', '.cache', 'vido-test-sessions');

export interface TestSession {
  sessionId: string;
  parentPid: number;
  startTime: number;
  pids: number[];
}

/**
 * Get the session file path for the current process
 */
export function getSessionFilePath(): string {
  // Use a combination of PID and random ID for uniqueness
  const sessionId = process.env.VIDO_TEST_SESSION_ID || `session-${process.pid}`;
  return path.join(SESSION_DIR, `${sessionId}.json`);
}

/**
 * Read the current session data
 */
export function readSession(): TestSession | null {
  const filePath = getSessionFilePath();
  try {
    if (fs.existsSync(filePath)) {
      return JSON.parse(fs.readFileSync(filePath, 'utf-8'));
    }
  } catch {
    // Ignore read errors
  }
  return null;
}

/**
 * Write session data
 */
export function writeSession(session: TestSession): void {
  const filePath = getSessionFilePath();
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(session, null, 2));
}

/**
 * Add a PID to the current session
 */
export function trackPid(pid: number): void {
  const session = readSession();
  if (session && !session.pids.includes(pid)) {
    session.pids.push(pid);
    writeSession(session);
  }
}

/**
 * Complete the setup wizard so the app doesn't redirect to /setup.
 * In fresh environments (CI or first-time local), the API returns needsSetup: true,
 * which causes __root.tsx to redirect all routes to /setup, breaking E2E tests.
 */
async function completeSetupWizard(apiUrl: string): Promise<void> {
  // Check if setup is needed
  const statusRes = await fetch(`${apiUrl}/setup/status`);
  if (!statusRes.ok) {
    console.log('   Setup status check failed, skipping setup completion');
    return;
  }

  const statusBody = await statusRes.json();
  const needsSetup = statusBody?.data?.needsSetup ?? statusBody?.needsSetup;

  if (!needsSetup) {
    console.log('   Setup already completed, skipping');
    return;
  }

  // Complete setup with minimal config
  const completeRes = await fetch(`${apiUrl}/setup/complete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      language: 'zh-TW',
      mediaFolderPath: '/media',
    }),
  });

  if (completeRes.ok) {
    console.log('   Setup wizard completed successfully');
  } else {
    const body = await completeRes.text();
    console.warn(`   Setup completion failed (${completeRes.status}): ${body}`);
  }
}

/**
 * Global setup function
 */
async function globalSetup(config: FullConfig): Promise<void> {
  // Generate unique session ID
  const sessionId = `session-${process.pid}-${crypto.randomBytes(4).toString('hex')}`;

  // Set environment variable for child processes
  process.env.VIDO_TEST_SESSION_ID = sessionId;

  // Create session tracking file
  const session: TestSession = {
    sessionId,
    parentPid: process.pid,
    startTime: Date.now(),
    pids: [process.pid],
  };

  writeSession(session);

  console.log(`\n📋 Test session started: ${sessionId}`);
  console.log(`   Session file: ${getSessionFilePath()}`);

  // Complete setup wizard to prevent redirect to /setup
  const apiUrl = process.env.API_URL || 'http://localhost:8080/api/v1';
  console.log(`   Checking setup wizard status at ${apiUrl}...`);
  await completeSetupWizard(apiUrl);
  console.log('');
}

export default globalSetup;
