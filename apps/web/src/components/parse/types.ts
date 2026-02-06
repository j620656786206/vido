/**
 * Parse Status Types (Story 3.10)
 * Types for parse progress tracking and SSE events
 */

// Parse status enum matching backend
export type ParseStatus = 'pending' | 'success' | 'needs_ai' | 'failed';

// Step status enum matching backend
export type StepStatus = 'pending' | 'in_progress' | 'success' | 'failed' | 'skipped';

// Parse step structure
export interface ParseStep {
  name: string;
  label: string;
  status: StepStatus;
  startedAt?: string;
  endedAt?: string;
  error?: string;
}

// Parse result structure
export interface ParseResult {
  mediaId?: string;
  title?: string;
  year?: number;
  mediaType?: string;
  metadataSource?: 'tmdb' | 'douban' | 'wikipedia' | 'manual';
  confidence?: number;
}

// Parse progress structure
export interface ParseProgress {
  taskId: string;
  filename: string;
  status: ParseStatus;
  steps: ParseStep[];
  currentStep: number;
  percentage: number;
  message?: string;
  result?: ParseResult;
  startedAt: string;
  completedAt?: string;
}

// SSE Event types matching backend
export type ParseEventType =
  | 'connected'
  | 'parse_started'
  | 'step_started'
  | 'step_completed'
  | 'step_failed'
  | 'step_skipped'
  | 'parse_completed'
  | 'parse_failed'
  | 'progress_update'
  | 'ping';

// Parse event structure
export interface ParseEvent {
  type: ParseEventType;
  taskId: string;
  timestamp: string;
  data?: unknown;
}

// Parse started event data
export interface ParseStartedData {
  filename: string;
  totalSteps: number;
  steps: ParseStep[];
}

// Step event data
export interface StepEventData {
  stepIndex: number;
  step: ParseStep;
  progress?: ParseProgress;
}

// Parse completed event data
export interface ParseCompletedData {
  result?: ParseResult;
  progress: ParseProgress;
}

// Parse failed event data
export interface ParseFailedData {
  message: string;
  failedSteps: ParseStep[];
  progress: ParseProgress;
}

// Progress update event data
export interface ProgressUpdateData {
  percentage: number;
  currentStep: number;
  progress: ParseProgress;
}

// Standard parse steps (matching backend)
export const STANDARD_PARSE_STEPS: ParseStep[] = [
  { name: 'filename_extract', label: '解析檔名', status: 'pending' },
  { name: 'tmdb_search', label: '搜尋 TMDb', status: 'pending' },
  { name: 'douban_search', label: '搜尋豆瓣', status: 'pending' },
  { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'pending' },
  { name: 'ai_retry', label: 'AI 重試', status: 'pending' },
  { name: 'download_poster', label: '下載海報', status: 'pending' },
];

// Status display configuration
export const STATUS_CONFIG: Record<
  ParseStatus,
  { label: string; color: string; icon: string }
> = {
  pending: { label: '等待中', color: 'text-muted-foreground', icon: 'clock' },
  success: { label: '已完成', color: 'text-green-500', icon: 'check-circle' },
  needs_ai: { label: '需要處理', color: 'text-yellow-500', icon: 'alert-triangle' },
  failed: { label: '失敗', color: 'text-red-500', icon: 'x-circle' },
};

// Step status display configuration
export const STEP_STATUS_CONFIG: Record<
  StepStatus,
  { label: string; color: string }
> = {
  pending: { label: '等待中', color: 'text-muted-foreground' },
  in_progress: { label: '進行中', color: 'text-blue-500' },
  success: { label: '成功', color: 'text-green-500' },
  failed: { label: '失敗', color: 'text-red-500' },
  skipped: { label: '跳過', color: 'text-muted-foreground' },
};
