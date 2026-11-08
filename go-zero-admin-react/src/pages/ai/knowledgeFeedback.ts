import { queryOptions } from '@tanstack/react-query';
import { fetchAiKbFeedback, fetchAiKbFeedbackList, fetchAiTaskFeedback, reviewAiKbFeedback, setAiTaskFeedback } from '../../api/ai';
import type { AiKbFeedbackInfo, AiKbFeedbackKind, AiListKbFeedbackReq, AiReviewKbFeedbackReq, AiSearchKbHit, AiSetTaskFeedbackReq } from '../../api/types';
import { isAiWorkspaceCurrent } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';

export const feedbackLabels: Record<AiKbFeedbackKind, string> = {
  helpful: '有帮助', incorrect_answer: '回答错误', missing_knowledge: '回答遗漏或资料不足', irrelevant_source: '来源不符',
};
export const reviewStatusLabels = { pending: '待复核', accepted: '已采纳', dismissed: '已忽略' } as const;

export interface FeedbackSnapshot {
  version: 1; kb_id: number; query: string; step_input: string; step_output: string;
  sources: (AiSearchKbHit & { index: number })[];
  documents: { doc_id: number; kb_id: number; title: string; version: number; content_sha256: string; status: 'verified' | 'stale' | 'unavailable' }[];
}

export interface FeedbackReviewValues {
  status: 'accepted' | 'dismissed'; answerable?: boolean; expected_answer?: string; required_facts?: string;
  evidence?: { source_index: number; quote: string }[]; reason: string;
}

const bytes = (value: string) => new TextEncoder().encode(value).byteLength;
const positive = (value: unknown): value is number => typeof value === 'number' && Number.isSafeInteger(value) && value > 0;
const object = (value: unknown): value is Record<string, unknown> => typeof value === 'object' && value !== null && !Array.isArray(value);

export function feedbackNoteError(value: string): string | undefined {
  if (bytes(value) > 1000) return '说明最多 1000 UTF-8 字节';
}

export function parseFeedbackSnapshot(raw: string): FeedbackSnapshot | null {
  if (bytes(raw) > 256 * 1024) return null;
  try {
    const value: unknown = JSON.parse(raw);
    if (!object(value) || value.version !== 1 || !positive(value.kb_id) || typeof value.query !== 'string'
      || typeof value.step_input !== 'string' || typeof value.step_output !== 'string' || !Array.isArray(value.sources) || value.sources.length > 8 || !Array.isArray(value.documents) || value.documents.length > 8
      || bytes(value.query) > 2000 || bytes(value.step_input) > 16000 || bytes(value.step_output) > 24000 || bytes(JSON.stringify(value.sources)) > 10000) return null;
    if (!value.sources.every((source: unknown, index: number) => object(source) && source.index === index + 1 && positive(source.kb_id)
      && source.kb_id === value.kb_id && positive(source.doc_id) && positive(source.chunk_id) && typeof source.doc_title === 'string'
      && typeof source.snippet === 'string' && typeof source.score === 'number' && Number.isFinite(source.score))) return null;
    if (!value.documents.every((doc: unknown) => object(doc) && positive(doc.doc_id) && doc.kb_id === value.kb_id && typeof doc.title === 'string'
      && typeof doc.version === 'number' && Number.isSafeInteger(doc.version) && doc.version >= 0 && typeof doc.content_sha256 === 'string'
      && ['verified', 'stale', 'unavailable'].includes(String(doc.status)))) return null;
    return value as unknown as FeedbackSnapshot;
  } catch { return null; }
}

function requiredText(value: string, label: string, limit: number) {
  if (!value.trim()) throw new Error(`请填写${label}`);
  if (bytes(value) > limit) throw new Error(`${label}最多 ${limit} UTF-8 字节`);
}

export function reviewValuesToRequest(values: FeedbackReviewValues, snapshot: FeedbackSnapshot, revision: number): AiReviewKbFeedbackReq {
  if (!positive(revision)) throw new Error('请刷新当前反馈版本');
  requiredText(values.reason ?? '', '复核说明', 1000);
  const empty = { answerable: false, expected_answer: '', required_facts: [], evidence: [], reason: values.reason };
  if (values.status === 'dismissed') return { expected_revision: revision, status: 'dismissed', review: empty };
  if (values.status !== 'accepted' || typeof values.answerable !== 'boolean') throw new Error('请选择复核结论');
  const expected_answer = values.expected_answer ?? '';
  requiredText(expected_answer, '预期答案', 4000);
  const required_facts = (values.required_facts ?? '').split(/\r?\n/).map(value => value.trim()).filter(Boolean);
  const evidence = values.evidence ?? [];
  if (!values.answerable && (required_facts.length || evidence.length)) throw new Error('资料不足题须清空事实项和依据摘录');
  if (values.answerable && (!required_facts.length || !evidence.length)) throw new Error('有据题须填写事实检查项和依据摘录');
  if (required_facts.length > 10 || required_facts.some(fact => bytes(fact) > 200)) throw new Error('事实检查项最多10项，每项最多200 UTF-8字节');
  if (evidence.length > 5 || evidence.some(item => !positive(item.source_index) || item.source_index > 8 || typeof item.quote !== 'string' || !item.quote.trim()
    || bytes(item.quote) > 500 || !snapshot.sources.some(source => source.index === item.source_index && source.snippet.includes(item.quote))))
    throw new Error('依据摘录最多5项，每项最多500 UTF-8字节，须逐字属于对应的历史片段');
  validateFactEvidence(expected_answer, required_facts, evidence);
  return { expected_revision: revision, status: 'accepted', review: { answerable: values.answerable, expected_answer, required_facts, evidence, reason: values.reason } };
}

// 与离线merge-kb-feedback使用相同的NFKC/全部空白归一规则，仍只是词法检查。
function validateFactEvidence(answer: string, facts: string[], evidence: { source_index: number; quote: string }[]) {
  const normalized = (value: string) => value.normalize('NFKC').replace(/\s+/gu, '');
  if (new Set(facts.map(normalized)).size !== facts.length) throw new Error('事实关键词不能重复（按NFKC和空白归一后比较）');
  if (new Set(evidence.map(row => JSON.stringify([row.source_index, row.quote]))).size !== evidence.length) throw new Error('同一历史片段的相同摘录不能重复');
  if (facts.some(fact => !normalized(answer).includes(normalized(fact)) || !evidence.some(row => normalized(row.quote).includes(normalized(fact)))))
    throw new Error('每个事实关键词都须同时出现在预期答案和至少一条依据摘录中');
}

export function feedbackRegressionJson(record: AiKbFeedbackInfo): string {
  const snapshot = parseFeedbackSnapshot(record.snapshot);
  if (record.status !== 'accepted' || !record.review || !snapshot || snapshot.kb_id !== record.kb_id) throw new Error('只能导出已采纳的完整反馈记录');
  reviewValuesToRequest({ status: 'accepted', ...record.review, required_facts: record.review.required_facts.join('\n') }, snapshot, record.revision);
  validateFactEvidence(record.review.expected_answer, record.review.required_facts, record.review.evidence);
  const exported = JSON.stringify({ version: 1, type: 'kb_feedback_regression', feedback: record }, null, 2) + '\n';
  if (bytes(exported) > 512 * 1024) throw new Error('反馈导出超过512 KiB字节上限');
  return exported;
}

function assertCurrent(scope: AiWorkspaceScope, readCode: string, writeCode?: string) {
  if (!isAiWorkspaceCurrent(scope, readCode) || writeCode && !isAiWorkspaceCurrent(scope, writeCode)) throw new Error('当前工作区或操作权限已变化，请重新打开');
}

async function readCurrent<T>(scope: AiWorkspaceScope, readCode: string, signal: AbortSignal, read: () => Promise<T>): Promise<T> {
  signal.throwIfAborted(); assertCurrent(scope, readCode);
  const result = await read();
  signal.throwIfAborted(); assertCurrent(scope, readCode);
  return result;
}

export function taskFeedbackOptions(scope: AiWorkspaceScope, taskId: number) {
  return queryOptions({ queryKey: aiWorkspaceKeys.taskFeedback(scope, taskId), staleTime: 0, retry: false,
    queryFn: ({ signal }) => readCurrent(scope, 'sys:ai:task:list', signal, () => fetchAiTaskFeedback(taskId, signal)) });
}

export function kbFeedbackListOptions(scope: AiWorkspaceScope, kbId: number, params: AiListKbFeedbackReq) {
  return queryOptions({ queryKey: [...aiWorkspaceKeys.kbFeedbackList(scope, kbId), params], staleTime: 0, retry: false,
    queryFn: ({ signal }) => readCurrent(scope, 'sys:ai:kb:list', signal, () => fetchAiKbFeedbackList(kbId, params, signal)) });
}

export function kbFeedbackDetailOptions(scope: AiWorkspaceScope, kbId: number, id: number) {
  return queryOptions({ queryKey: aiWorkspaceKeys.kbFeedback(scope, kbId, id), staleTime: 0, retry: false,
    queryFn: ({ signal }) => readCurrent(scope, 'sys:ai:kb:list', signal, () => fetchAiKbFeedback(kbId, id, signal)) });
}

export function taskFeedbackMutationOptions(scope: AiWorkspaceScope, taskId: number) {
  return { retry: false as const, gcTime: 0, mutationFn: async (data: AiSetTaskFeedbackReq) => {
    assertCurrent(scope, 'sys:ai:task:list', 'sys:ai:task:feedback');
    const error = feedbackNoteError(data.note);
    if (error) throw new Error(error);
    const result = await setAiTaskFeedback(taskId, data);
    assertCurrent(scope, 'sys:ai:task:list', 'sys:ai:task:feedback');
    return result;
  } };
}

export function kbReviewMutationOptions(scope: AiWorkspaceScope, kbId: number, id: number) {
  return { retry: false as const, gcTime: 0, mutationFn: async (data: AiReviewKbFeedbackReq) => {
    assertCurrent(scope, 'sys:ai:kb:list', 'sys:ai:kb:review');
    const result = await reviewAiKbFeedback(kbId, id, data);
    assertCurrent(scope, 'sys:ai:kb:list', 'sys:ai:kb:review');
    return result;
  } };
}
