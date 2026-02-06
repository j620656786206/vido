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
  console.log(`   Session file: ${getSessionFilePath()}\n`);
}

export default globalSetup;
