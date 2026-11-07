import type { AiCreateKbDocReq, AiKbDocInfo, AiKbDocResp, AiKbIngestBatchInfo } from '../../api/types';
import { DOCUMENT_MAX_BYTES, documentWriteFailure, validateDocumentContent } from './knowledgeDocuments';

export const KNOWLEDGE_IMPORT_MAX_FILES = 20;
export interface KnowledgeImportItem { id: string; values: AiCreateKbDocReq }
export type KnowledgeImportState = 'waiting' | 'submitting' | 'ready' | 'failed' | 'pending' | 'unknown';
export interface KnowledgeImportChange { id: string; state: KnowledgeImportState; doc?: AiKbDocInfo; error?: string }
export interface KnowledgeImportAttempt { batchId: string; requestIds: string[]; unsubmittedIds?: string[] }
type ImportStorage = Pick<Storage, 'getItem' | 'setItem'>;
const attemptEvent = 'ai-knowledge-import-attempt';
interface MemoryAttempt { value: KnowledgeImportAttempt | null; dirty: boolean }
const attemptMemory = new WeakMap<object, Map<string, MemoryAttempt>>();

function memoryFor(storage: object) {
  let memory = attemptMemory.get(storage);
  if (!memory) { memory = new Map(); attemptMemory.set(storage, memory); }
  return memory;
}

function notifyAttempt(key: string) {
  if (typeof window !== 'undefined') window.dispatchEvent?.(new CustomEvent(attemptEvent, { detail: key }));
}

export function writeKnowledgeImportAttempt(storage: Pick<ImportStorage, 'setItem'>, key: string, attempt: KnowledgeImportAttempt) {
  const record = { value: attempt, dirty: true };
  memoryFor(storage).set(key, record);
  try { storage.setItem(key, JSON.stringify(attempt)); record.dirty = false; }
  finally { notifyAttempt(key); }
}

export function clearKnowledgeImportAttempt(storage: Pick<Storage, 'removeItem'>, key: string) {
  const record = { value: null, dirty: true };
  memoryFor(storage).set(key, record);
  try { storage.removeItem(key); record.dirty = false; }
  finally { notifyAttempt(key); }
}

export function subscribeKnowledgeImportAttempt(storage: Pick<ImportStorage, 'getItem'>, key: string, onChange: (attempt: KnowledgeImportAttempt | null) => void) {
  const listener = (event: Event) => {
    if (!(event instanceof CustomEvent) || event.detail !== key) return;
    try { onChange(readKnowledgeImportAttempt(storage, key)); } catch { /* 无法读取时维持原未知锁，不能当成已确认。 */ }
  };
  window.addEventListener(attemptEvent, listener);
  return () => window.removeEventListener(attemptEvent, listener);
}

export function readKnowledgeImportAttempt(storage: Pick<ImportStorage, 'getItem'>, key: string): KnowledgeImportAttempt | null {
  const memory = memoryFor(storage);
  const record = memory.get(key);
  if (record?.dirty) return record.value;
  let raw: string | null;
  try { raw = storage.getItem(key); }
  catch (error) { if (record) return record.value; throw error; }
  if (!raw) { memory.set(key, { value: null, dirty: false }); return null; }
  const value = JSON.parse(raw);
  const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
  if (!value || typeof value.batchId !== 'string' || !uuid.test(value.batchId) || !Array.isArray(value.requestIds) || value.requestIds.length > 20
    || !value.requestIds.every((id: unknown) => typeof id === 'string' && uuid.test(id))
    || value.unsubmittedIds !== undefined && (!Array.isArray(value.unsubmittedIds) || value.unsubmittedIds.length > 20 || !value.unsubmittedIds.every((id: unknown) => typeof id === 'string' && uuid.test(id)))) throw new Error('原导入标识无法读取，请先核对已有批次。');
  memory.set(key, { value, dirty: false });
  return value;
}

export function updateKnowledgeImportAttempt(storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>, key: string, batchId: string,
  update: (latest: KnowledgeImportAttempt) => KnowledgeImportAttempt | null): KnowledgeImportAttempt | null {
  const latest = readKnowledgeImportAttempt(storage, key);
  // 同一标签页内同步读改写；迟到响应只能推进仍存在的同批次回执。
  if (!latest || latest.batchId !== batchId) return latest;
  const changed = update(latest);
  if (changed === latest) return latest;
  if (changed && changed.batchId !== batchId) throw new Error('不能替换原导入批次标识');
  const remaining = changed && (changed.requestIds.length || changed.unsubmittedIds?.length) ? changed : null;
  if (remaining) writeKnowledgeImportAttempt(storage, key, remaining);
  else clearKnowledgeImportAttempt(storage, key);
  return remaining;
}

export function advanceKnowledgeImportAttempt(attempt: KnowledgeImportAttempt, change: KnowledgeImportChange): KnowledgeImportAttempt {
  const tracked = attempt.requestIds.includes(change.id);
  if (change.state === 'submitting' ? !tracked && !attempt.unsubmittedIds?.includes(change.id) : !tracked) return attempt;
  const unsubmittedIds = (attempt.unsubmittedIds ?? []).filter(id => id !== change.id);
  const requestIds = change.state === 'submitting' || change.state === 'unknown'
    ? [...new Set([...attempt.requestIds, change.id])]
    : attempt.requestIds.filter(id => id !== change.id);
  return { ...attempt, requestIds, unsubmittedIds };
}

export function prepareKnowledgeBatch(values: Pick<AiCreateKbDocReq, 'title' | 'content'>[]) {
  if (values.length < 1 || values.length > KNOWLEDGE_IMPORT_MAX_FILES) throw new Error('每批须包含1..20篇资料');
  const batchId = crypto.randomUUID();
  return { batchId, items: values.map(value => { const id = crypto.randomUUID(); return { id, values: { ...value, request_id: id, batch_id: batchId } }; }) };
}

export function knowledgeBatchPollInterval(batch: Pick<AiKbIngestBatchInfo, 'queued' | 'running'> | undefined, active: boolean): number | false {
  return active && !!batch && batch.queued + batch.running > 0 ? 3000 : false;
}

export async function readKnowledgeFile(file: File): Promise<Pick<AiCreateKbDocReq, 'title' | 'content'>> {
  if (!/\.(txt|md|markdown)$/i.test(file.name)) throw new Error('仅支持 TXT、MD 和 Markdown 文件');
  if (file.size === 0 || file.size > DOCUMENT_MAX_BYTES) throw new Error('每个文件须为 1..65536 字节');
  const title = file.name.replace(/\.(txt|md|markdown)$/i, '').trim();
  if (!title || Array.from(title).length > 128) throw new Error('文件名去掉扩展名后须为 1..128 字符');
  let content: string;
  try {
    // 严格解码避免静默替换乱码；BOM 和 CRLF 保持原貌，不改变待索引资料。
    content = new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(await file.arrayBuffer());
  } catch { throw new Error('文件必须使用 UTF-8 编码，请转换编码后重新选择'); }
  const error = validateDocumentContent(content);
  if (error) throw new Error(error);
  return { title, content };
}

export async function runKnowledgeImport(items: KnowledgeImportItem[], submit: (values: AiCreateKbDocReq) => Promise<AiKbDocResp>,
  onChange: (change: KnowledgeImportChange) => void, isCurrent: () => boolean,
  onAttempt?: (change: KnowledgeImportChange) => void): Promise<void> {
  if (items.length > KNOWLEDGE_IMPORT_MAX_FILES) throw new Error('每批最多 20 个文件');
  let cursor = 0;
  async function worker() {
    while (isCurrent() && cursor < items.length) {
      const item = items[cursor++];
      onAttempt?.({ id: item.id, state: 'submitting' });
      onChange({ id: item.id, state: 'submitting' });
      try {
        const result = await submit(item.values);
        onAttempt?.({ id: item.id, state: result.doc.status, doc: result.doc });
        if (isCurrent()) onChange({ id: item.id, state: result.doc.status, doc: result.doc });
      } catch (cause) {
        // 未获得权威文档ID时无法断言回滚，也不能自动重放可能已保存的资料。
        const failure = documentWriteFailure(cause);
        onAttempt?.({ id: item.id, state: failure.uncertain ? 'unknown' : 'failed', error: failure.message });
        if (isCurrent()) onChange({ id: item.id, state: failure.uncertain ? 'unknown' : 'failed', error: failure.message });
      }
    }
  }
  await Promise.all([worker(), worker()]);
}
