import type { AiToolInfo, AiUpdateWorkflowReq, AiWorkflowInfo } from '../../api/types';
import type { WorkflowDraft } from './workflowDefinition';
import { parseWorkflow } from './workflowDefinition';

export type WorkflowSubmissionState = 'idle' | 'submitting' | 'unknown';
export interface StoredWorkflowDraft { schemaVersion: 1; draft: WorkflowDraft; baseSnapshot: string | null; updatedAt: number; submissionState: WorkflowSubmissionState }
type DraftStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;

export const workflowDraftKey = (sessionId: string, departmentId: number | null, workflowId: number | 'new') => `ai-workflow-draft:${JSON.stringify([sessionId, departmentId, workflowId])}`;
export const makeWorkflowDraft = (draft: WorkflowDraft, baseSnapshot: string | null, submissionState: WorkflowSubmissionState = 'idle'): StoredWorkflowDraft => ({ schemaVersion: 1, draft, baseSnapshot, updatedAt: Date.now(), submissionState });
export const writeWorkflowDraft = (storage: Pick<DraftStorage, 'setItem'>, key: string, record: StoredWorkflowDraft) => storage.setItem(key, JSON.stringify(record));
export const clearWorkflowDraft = (storage: Pick<DraftStorage, 'removeItem'>, key: string) => storage.removeItem(key);

export function readWorkflowDraft(storage: Pick<DraftStorage, 'getItem'>, key: string): StoredWorkflowDraft | null {
  const raw = storage.getItem(key);
  if (!raw) return null;
  let value: StoredWorkflowDraft;
  try { value = JSON.parse(raw); } catch { return null; }
  if (value?.schemaVersion !== 1 || !value.draft || typeof value.draft.name !== 'string' || typeof value.draft.enabled !== 'boolean' || !Array.isArray(value.draft.steps)
    || value.draft.steps.length < 1 || value.draft.steps.length > 32
    || value.draft.steps.some(step => !step || typeof step.key !== 'string' || typeof step.name !== 'string' || typeof step.tool !== 'string' || typeof step.input !== 'string' || typeof step.approval !== 'boolean'
      || step.id !== undefined && (typeof step.id !== 'string' || !/^[A-Za-z0-9_-]{1,64}$/.test(step.id))
      || step.toolInputs !== undefined && (step.toolInputs === null || typeof step.toolInputs !== 'object' || Array.isArray(step.toolInputs) || Object.values(step.toolInputs).some(input => typeof input !== 'string')))
    || (value.baseSnapshot !== null && typeof value.baseSnapshot !== 'string') || !Number.isFinite(value.updatedAt) || !['idle', 'submitting', 'unknown'].includes(value.submissionState)) return null;
  return { ...value, submissionState: value.submissionState === 'submitting' ? 'unknown' : value.submissionState };
}

export const workflowSnapshot = (workflow: AiWorkflowInfo) => JSON.stringify({ name: workflow.name, enabled: workflow.enabled, definition: workflow.definition, updatedAt: workflow.updated_at, revision: workflow.revision });
export function workflowBaseRevision(snapshot: string | null): number | null {
  try {
    const revision = snapshot ? (JSON.parse(snapshot) as { revision?: unknown } | null)?.revision : undefined;
    return typeof revision === 'number' && Number.isSafeInteger(revision) && revision >= 1 ? revision : null;
  } catch { return null; }
}
export function workflowUpdateRequest(draft: WorkflowDraft, definition: string, snapshot: string | null): AiUpdateWorkflowReq {
  const revision = workflowBaseRevision(snapshot);
  if (revision === null) throw new Error('草稿缺少有效基线版本，请先核对最新工作流。');
  return { expected_revision: revision, name: draft.name.trim(), definition, enabled: draft.enabled ? 1 : 0 };
}
export const draftContent = (draft: WorkflowDraft) => JSON.stringify({ ...draft, steps: draft.steps.map(({ key: _key, ...step }) => step) });

export function resolveWorkflowEditorDraft(workflow: AiWorkflowInfo | undefined, restored: StoredWorkflowDraft | null, tools: readonly AiToolInfo[], fallback: WorkflowDraft) {
  // 已校验的本地草稿优先恢复。远端旧工具下线只影响远端可编辑性，不能吞掉本地草稿。
  if (restored) return { draft: restored.draft, baseSnapshot: restored.baseSnapshot, initialContent: '' };
  const draft = workflow ? { name: workflow.name, enabled: workflow.enabled === 1, steps: parseWorkflow(workflow.definition, tools) } : fallback;
  return { draft, baseSnapshot: workflow ? workflowSnapshot(workflow) : null, initialContent: draftContent(draft) };
}
