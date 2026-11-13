import type { AiRunWorkflowReq } from '../../api/types';
import { validateRunInput } from './workflowDefinition';

export interface WorkflowRunAttempt extends AiRunWorkflowReq { workflowId: number }
export const workflowRunAttemptKey = (sessionId: string, departmentId: number | null, id: number) => `ai-workflow-run:${JSON.stringify([sessionId, departmentId, id])}`;
type AttemptStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

export function readWorkflowRunAttempt(storage: Pick<AttemptStorage, 'getItem'>, memory: Map<string, WorkflowRunAttempt>, key: string, workflowId: number): WorkflowRunAttempt | null {
  const local = memory.get(key);
  if (local) return local;
  const raw = storage.getItem(key);
  if (!raw) return null;
  const value: unknown = JSON.parse(raw);
  if (!value || typeof value !== 'object' || !('workflowId' in value) || value.workflowId !== workflowId || !('request_id' in value) || typeof value.request_id !== 'string'
    || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(value.request_id) || !('input' in value) || typeof value.input !== 'string' || validateRunInput(value.input)) throw new Error('运行标识格式异常，请先到运行记录核对上次运行。');
  const attempt = { workflowId, request_id: value.request_id, input: value.input };
  memory.set(key, attempt); return attempt;
}

export function saveWorkflowRunAttempt(storage: Pick<AttemptStorage, 'setItem'>, memory: Map<string, WorkflowRunAttempt>, key: string, attempt: WorkflowRunAttempt) {
  // 写存储失败时仍保留当前工作区内唯一的请求标识，关闭弹窗不能将未知提交变成新任务。
  memory.set(key, attempt);
  storage.setItem(key, JSON.stringify(attempt));
}

export function clearWorkflowRunAttempt(storage: Pick<AttemptStorage, 'removeItem'>, memory: Map<string, WorkflowRunAttempt>, key: string) {
  memory.delete(key); storage.removeItem(key);
}
