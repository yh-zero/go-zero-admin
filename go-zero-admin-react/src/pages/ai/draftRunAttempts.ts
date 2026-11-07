import type { runAiDraft } from '../../api/ai';

export type DraftRunAttempt = Parameters<typeof runAiDraft>[0];
type AttemptStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;
interface MemoryAttempt { value: DraftRunAttempt | null; dirty: boolean }
const attemptMemory = new WeakMap<object, Map<string, MemoryAttempt>>();

function memoryFor(storage: object) {
  let memory = attemptMemory.get(storage);
  if (!memory) { memory = new Map(); attemptMemory.set(storage, memory); }
  return memory;
}

export const draftRunAttemptKey = (scope: readonly unknown[], key: string) => `ai-draft-run:${JSON.stringify([...scope, key])}`;

export function readDraftRunAttempt(storage: Pick<AttemptStorage, 'getItem'>, key: string): DraftRunAttempt | null {
  const memory = memoryFor(storage);
  const record = memory.get(key);
  // 未落盘的原请求或确认清理结果优先，路由重挂载不能从旧存储恢复另一状态。
  if (record?.dirty) return record.value;
  let raw: string | null;
  try { raw = storage.getItem(key); }
  catch (error) { if (record) return record.value; throw error; }
  if (!raw) { memory.set(key, { value: null, dirty: false }); return null; }
  const attempt = JSON.parse(raw);
  if (!attempt || typeof attempt.name !== 'string' || typeof attempt.definition !== 'string' || typeof attempt.input !== 'string'
    || typeof attempt.request_id !== 'string' || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(attempt.request_id)) throw new Error('运行标识无法恢复，请先到运行记录核对。');
  memory.set(key, { value: attempt, dirty: false });
  return attempt;
}

export function writeDraftRunAttempt(storage: Pick<AttemptStorage, 'setItem'>, key: string, attempt: DraftRunAttempt) {
  const record = { value: attempt, dirty: true };
  memoryFor(storage).set(key, record);
  storage.setItem(key, JSON.stringify(attempt));
  record.dirty = false;
}

export function clearDraftRunAttempt(storage: Pick<AttemptStorage, 'removeItem'>, key: string) {
  const record = { value: null, dirty: true };
  memoryFor(storage).set(key, record);
  storage.removeItem(key);
  record.dirty = false;
}
