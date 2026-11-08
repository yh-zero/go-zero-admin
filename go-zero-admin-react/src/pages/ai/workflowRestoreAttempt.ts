import type { AiRestoreWorkflowRevisionReq } from '../../api/types';

export interface WorkflowRestoreAttempt extends AiRestoreWorkflowRevisionReq { revision: number }
type AttemptStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;
const memories = new WeakMap<object, Map<string, WorkflowRestoreAttempt | null>>();
function memory(storage: object) {
  let value = memories.get(storage);
  if (!value) { value = new Map(); memories.set(storage, value); }
  return value;
}
export const workflowRestoreAttemptKey = (sessionId: string, departmentId: number | null, workflowId: number) => `ai-workflow-restore:${JSON.stringify([sessionId, departmentId, workflowId])}`;
export function readWorkflowRestoreAttempt(storage: Pick<AttemptStorage, 'getItem'>, key: string): WorkflowRestoreAttempt | null {
  const current = memory(storage);
  if (current.has(key)) return current.get(key) ?? null;
  const raw = storage.getItem(key);
  if (!raw) return null;
  const value = JSON.parse(raw) as WorkflowRestoreAttempt | null;
  if (!value || !Number.isSafeInteger(value.revision) || value.revision < 1 || !Number.isSafeInteger(value.expected_revision) || value.expected_revision < 1) throw new Error('原恢复版本无法读取，请先核对最新版本和历史。');
  current.set(key, value);
  return value;
}
export function writeWorkflowRestoreAttempt(storage: Pick<AttemptStorage, 'setItem'>, key: string, value: WorkflowRestoreAttempt) {
  memory(storage).set(key, value);
  storage.setItem(key, JSON.stringify(value));
}
export function clearWorkflowRestoreAttempt(storage: Pick<AttemptStorage, 'removeItem'>, key: string) {
  memory(storage).set(key, null);
  storage.removeItem(key);
}
