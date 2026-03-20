/**
 * TanStack Query hooks for backup management (Story 6.5)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { backupService } from '../services/backupService';
import type {
  BackupListResponse,
  Backup,
  VerificationResult,
  RestoreResult,
  BackupSchedule,
} from '../services/backupService';

export const backupKeys = {
  all: ['backups'] as const,
  list: () => [...backupKeys.all, 'list'] as const,
  detail: (id: string) => [...backupKeys.all, 'detail', id] as const,
};

export function useBackups() {
  return useQuery<BackupListResponse, Error>({
    queryKey: backupKeys.list(),
    queryFn: () => backupService.listBackups(),
  });
}

export function useCreateBackup() {
  const queryClient = useQueryClient();

  return useMutation<Backup, Error>({
    mutationFn: () => backupService.createBackup(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: backupKeys.list() });
    },
  });
}

export function useDeleteBackup() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: (id) => backupService.deleteBackup(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: backupKeys.list() });
    },
  });
}

export function useRestoreBackup() {
  const queryClient = useQueryClient();

  return useMutation<RestoreResult, Error, string>({
    mutationFn: (id) => backupService.restoreBackup(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: backupKeys.list() });
    },
  });
}

export function useBackupSchedule() {
  return useQuery<BackupSchedule, Error>({
    queryKey: [...backupKeys.all, 'schedule'] as const,
    queryFn: () => backupService.getSchedule(),
  });
}

export function useUpdateSchedule() {
  const queryClient = useQueryClient();

  return useMutation<BackupSchedule, Error, Partial<BackupSchedule>>({
    mutationFn: (schedule) => backupService.updateSchedule(schedule),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...backupKeys.all, 'schedule'] });
    },
  });
}

export function useVerifyBackup() {
  const queryClient = useQueryClient();

  return useMutation<VerificationResult, Error, string>({
    mutationFn: (id) => backupService.verifyBackup(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: backupKeys.list() });
    },
  });
}
