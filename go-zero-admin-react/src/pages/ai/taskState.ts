import type { AiListTasksReq, AiTaskInfo, AiTaskStatus } from '../../api/types';
import { correctPageParams } from '../../hooks/usePagedQuery';

export const taskStatuses: Record<AiTaskStatus, { label: string; color: string }> = {
  pending: { label: '等待执行', color: 'default' },
  running: { label: '执行中', color: 'processing' },
  pending_approval: { label: '待确认', color: 'warning' },
  succeeded: { label: '已完成', color: 'success' },
  failed: { label: '执行失败', color: 'error' },
  rejected: { label: '已拒绝', color: 'default' },
};

export interface TaskDecision { id: number; expected_step: number; action: 'approve' | 'reject' }

export function correctTaskPageParams(current: AiListTasksReq, requested: AiListTasksReq,
  result: { isSuccess: boolean; isFetching: boolean; total: number | undefined }): AiListTasksReq {
  if (current !== requested || !result.isSuccess || result.isFetching || result.total === undefined) return current;
  return correctPageParams(current, result.total);
}

export function taskStepLabel(task: Pick<AiTaskInfo, 'status' | 'next_step'>): string {
  return ['succeeded', 'failed', 'rejected'].includes(task.status) ? `已完成 ${task.next_step} 步` : `第 ${task.next_step + 1} 步`;
}

export function captureTaskDecision(task: Pick<AiTaskInfo, 'id' | 'status' | 'next_step'>, action: TaskDecision['action']): TaskDecision | null {
  if (task.status !== 'pending_approval' || !Number.isSafeInteger(task.next_step) || task.next_step < 0) return null;
  return { id: task.id, expected_step: task.next_step, action };
}

export function isTaskDecisionCurrent(decision: TaskDecision, task: Pick<AiTaskInfo, 'id' | 'status' | 'next_step'> | undefined) {
  return task?.status === 'pending_approval' && decision.id === task.id && decision.expected_step === task.next_step;
}

export function validateRejectReason(reason: string): string | undefined {
  const trimmed = reason.trim();
  if (!trimmed) return '请填写拒绝原因';
  if (Array.from(trimmed).length > 512) return '拒绝原因最多 512 个字符';
}

export function taskPollInterval(status: AiTaskStatus | undefined, active: boolean, elapsed: number, failures: number): number | false {
  return active && (status === 'pending' || status === 'running') && elapsed < 120_000 && failures === 0 ? 3000 : false;
}

export function parseTaskId(value: string | null): number | null {
  if (!value || !/^[1-9]\d*$/.test(value)) return null;
  const id = Number(value);
  return Number.isSafeInteger(id) ? id : null;
}

export function formatTaskValue(value: string): string {
  try { return JSON.stringify(JSON.parse(value), null, 2); }
  catch { return value; }
}
