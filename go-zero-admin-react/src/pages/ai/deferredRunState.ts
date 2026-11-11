import { readDraftRunAttempt } from './draftRunAttempts';
import type { DraftRunViewState } from './WorkflowDraftRun';

export function readDeferredRunState(storage: Pick<Storage, 'getItem'> | undefined, key: string): DraftRunViewState {
  if (!storage) return { kind: 'idle', taskId: null };
  try {
    if (readDraftRunAttempt(storage, key)) return { kind: 'unknown', taskId: null };
    const taskId = Number(storage.getItem(`${key}:task`));
    return Number.isSafeInteger(taskId) && taskId > 0 ? { kind: 'result', taskId } : { kind: 'idle', taskId: null };
  } catch { return { kind: 'unknown', taskId: null }; }
}
