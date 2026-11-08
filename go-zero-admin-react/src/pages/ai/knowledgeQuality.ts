import { evaluateRetrieval, screenAnswer, summarize, validateAnalysis } from '../../../../go-zero-admin/data/scripts/eval-kb-quality-metrics.mjs';
import type { QualityAnswerScreen, QualityRetrieval, QualitySummary } from '../../../../go-zero-admin/data/scripts/eval-kb-quality-metrics.mjs';
import { validateContext } from '../../../../go-zero-admin/data/scripts/eval-kb-quality-context.mjs';
import { parseCustomDataset } from '../../../../go-zero-admin/data/scripts/kb-custom-dataset.mjs';
import type { AiKbEvaluationDatasetInfo, AiKbEvaluationInfo, AiKbEvaluationStatus, AiSearchKbHit, AiStartKbEvaluationReq } from '../../api/types';
import type { WorkflowAnalysisResult } from './workflowAnalysis';
import type { AiWorkspaceScope } from './workspaceQueries';

export interface QualityDocument { key: string; kb: string; source: string; title: string; sha256: string; source_doc_id?: number; source_doc_version?: number; content?: string }
export interface QualityCase {
  id: string; kb: string; split: 'development' | 'validation'; category: string; query: string; rationale: string;
  expected_documents: string[]; evidence: { document: string; groups: string[][] }[];
  answer: { kind: 'answerable' | 'unanswerable'; required_groups: string[][]; forbidden_phrases: string[]; absence_policy?: 'required' | 'forbidden' };
  checks?: { label: string; kind: 'fact' | 'condition' | 'risk' | 'unknown'; required_group: number }[]; expected_answer_example?: string;
}
export interface QualityDataset { version: 1; description?: string; documents: QualityDocument[]; cases: QualityCase[] }
export interface QualityCatalogue { key: string; name: string; sha256: string; raw: string; dataset: QualityDataset }
export interface QualityBinding extends QualityDocument { doc_id: number; kb_id: number; version: number; status: 'ready'; embedding_model: string; content_sha256: string }
export interface QualityInventory { id: number; kb: string; kb_id: number; title: string; version: number; status: 'ready' | 'pending' | 'failed'; embedding_model: string }
export interface QualityContext { binary_sha: string; model: { provider: string; name: string }; embedding: { provider: string; model: string; dimension: number }; config: Record<string, unknown>; [key: string]: unknown }
export interface QualityRow {
  item: QualityCase; requestId: string; taskId: number; status: 'waiting' | 'running' | 'succeeded' | 'failed' | 'unknown'; stage: string;
  list: AiSearchKbHit[]; analysis: WorkflowAnalysisResult | null; error: string; executionFailed: boolean;
  searchClientMs: number; searchServerMs: number; analysisMs: number; retrieval: QualityRetrieval | null;
  screen: QualityAnswerScreen | null; checks: { label: string; kind: string; requiredGroup: number; matched: boolean }[];
}
export interface QualityRunView { run: AiKbEvaluationInfo; dataset: QualityDataset; context: QualityContext; bindings: { documents: QualityBinding[]; inventory: QualityInventory[] }; rows: QualityRow[]; summary: QualitySummary }
const encoder = new TextEncoder();
const positive = (v: unknown): v is number => typeof v === 'number' && Number.isSafeInteger(v) && v > 0;
const count = (v: unknown): v is number => typeof v === 'number' && Number.isSafeInteger(v) && v >= 0;
const finite = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v) && v >= 0;
const object = (v: unknown): v is Record<string, unknown> => typeof v === 'object' && v !== null && !Array.isArray(v);
const label = (v: unknown): v is string => typeof v === 'string' && !!v.trim();
const digest = (v: unknown): v is string => typeof v === 'string' && /^[a-f0-9]{64}$/u.test(v);
const uuid = (v: unknown): v is string => typeof v === 'string' && /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/u.test(v);
const groups = (v: unknown): v is string[][] => Array.isArray(v) && v.every(g => Array.isArray(g) && g.length > 0 && g.every(label));
const strings = (v: unknown): v is string[] => Array.isArray(v) && v.every(label);

// 先由原生JSON解析确认语法，再检查键的真实解码值，避免重复键悄悄覆盖状态。
export function parseQualityJson(raw: string, maxBytes = 1_048_576): unknown {
  if (typeof raw !== 'string' || encoder.encode(raw).length > maxBytes) throw new Error('记录超过大小限制');
  let value: unknown;
  try { value = JSON.parse(raw); } catch { throw new Error('记录不是完整有效的JSON'); }
  const stack: ({ keys: Set<string>; key: boolean } | null)[] = [];
  for (let i = 0; i < raw.length; i++) {
    const ch = raw[i];
    if (ch === '"') {
      const start = i;
      for (i++; i < raw.length; i++) { if (raw[i] === '\\') i++; else if (raw[i] === '"') break; }
      const current = stack.at(-1);
      const decoded: string = JSON.parse(raw.slice(start, i + 1));
      if (/[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/u.test(decoded)) throw new Error('记录包含无效Unicode字符');
      if (current?.key) {
        const name = decoded;
        if (current.keys.has(name)) throw new Error('记录包含重复JSON字段');
        current.keys.add(name); current.key = false;
      }
    } else if (ch === '{') stack.push({ keys: new Set(), key: true });
    else if (ch === '[') stack.push(null);
    else if (ch === '}' || ch === ']') stack.pop();
    else if (ch === ',' && stack.at(-1)) stack.at(-1)!.key = true;
    if (stack.length > 64) throw new Error('记录嵌套过深');
  }
  return value;
}

export async function qualitySha256(raw: string): Promise<string> {
  const bytes = await crypto.subtle.digest('SHA-256', encoder.encode(raw));
  return [...new Uint8Array(bytes)].map(value => value.toString(16).padStart(2, '0')).join('');
}

export async function parseQualityDataset(info: AiKbEvaluationDatasetInfo): Promise<QualityCatalogue> {
  if (!info || !label(info.key) || !label(info.name) || !digest(info.sha256) || typeof info.dataset !== 'string') throw new Error('题集信息无效');
  if (info.key.startsWith('custom-')) {
    const dataset = await parseCustomDataset(info.dataset) as QualityDataset;
    if (await qualitySha256(info.dataset) !== info.sha256 || info.key !== `custom-${info.sha256.slice(0, 32)}`) throw new Error('题集指纹不符，请刷新后重试');
    return { key: info.key, name: info.name, sha256: info.sha256, raw: info.dataset, dataset };
  }
  const value = parseQualityJson(info.dataset, 262_144);
  if (await qualitySha256(info.dataset) !== info.sha256) throw new Error('题集指纹不符，请刷新后重试');
  if (!object(value) || value.version !== 1 || !Array.isArray(value.documents) || !value.documents.length || value.documents.length > 100 || !Array.isArray(value.cases) || !value.cases.length || value.cases.length > 100) throw new Error('题集结构或数量无效');
  if (value.description !== undefined && typeof value.description !== 'string') throw new Error('题集说明无效');
  const docs = new Map<string, QualityDocument>();
  for (const raw of value.documents) {
    if (!object(raw) || !label(raw.key) || docs.has(raw.key) || !label(raw.kb) || !label(raw.title) || !label(raw.source) || !digest(raw.sha256)) throw new Error('题集资料重复或字段无效');
    if (raw.source_doc_id !== undefined && !positive(raw.source_doc_id) || raw.source_doc_version !== undefined && !positive(raw.source_doc_version)) throw new Error('资料原ID或版本无效');
    if (raw.content !== undefined && (typeof raw.content !== 'string' || !raw.content.trim() || raw.source !== `inline:${raw.key}` || await qualitySha256(raw.content) !== raw.sha256)) throw new Error('内联资料正文或指纹不符');
    if (raw.source.startsWith('inline:') && typeof raw.content !== 'string') throw new Error('内联资料缺少正文');
    docs.set(raw.key, raw as unknown as QualityDocument);
  }
  const ids = new Set();
  for (const raw of value.cases) {
    if (!object(raw) || !label(raw.id) || ids.has(raw.id) || !label(raw.kb) || !['development', 'validation'].includes(String(raw.split)) || !label(raw.category) || !label(raw.query) || encoder.encode(raw.query).length > 2000 || !label(raw.rationale)
      || !strings(raw.expected_documents) || new Set(raw.expected_documents).size !== raw.expected_documents.length || raw.expected_documents.some(key => docs.get(key)?.kb !== raw.kb) || !Array.isArray(raw.evidence) || !object(raw.answer)) throw new Error('题目字段、分组或标识无效');
    const answer = raw.answer;
    if (raw.expected_answer_example !== undefined && (typeof raw.expected_answer_example !== 'string' || encoder.encode(raw.expected_answer_example).length > 4000)
      || raw.source_case_id !== undefined && !label(raw.source_case_id)) throw new Error('题目预期答案或来源标识无效');
    if (!['answerable', 'unanswerable'].includes(String(answer.kind)) || !groups(answer.required_groups) || !Array.isArray(answer.forbidden_phrases) || !answer.forbidden_phrases.every(label)
      || answer.absence_policy !== undefined && !['required', 'forbidden'].includes(String(answer.absence_policy))
      || (answer.kind === 'answerable' ? !answer.required_groups.length || !raw.expected_documents.length || raw.evidence.length !== raw.expected_documents.length
        : raw.expected_documents.length !== 0 || raw.evidence.length !== 0 || answer.required_groups.length !== 0)) throw new Error('题目答案标注无效');
    const evidenceDocs = new Set<string>();
    for (const evidence of raw.evidence) {
      if (!object(evidence) || !label(evidence.document) || evidenceDocs.has(evidence.document) || !raw.expected_documents.includes(evidence.document) || !groups(evidence.groups) || !evidence.groups.length) throw new Error('题目依据标注无效');
      evidenceDocs.add(evidence.document);
    }
    const groupCount = answer.required_groups.length;
    if (raw.checks !== undefined && (!Array.isArray(raw.checks) || raw.checks.length !== groupCount || new Set(raw.checks.map(c => object(c) ? c.required_group : null)).size !== raw.checks.length
      || raw.checks.some(c => !object(c) || !label(c.label) || !['fact', 'condition', 'risk', 'unknown'].includes(String(c.kind)) || !count(c.required_group) || c.required_group >= groupCount))) throw new Error('题目检查项无效');
    ids.add(raw.id);
  }
  return { key: info.key, name: info.name, sha256: info.sha256, raw: info.dataset, dataset: value as unknown as QualityDataset };
}

export async function importQualityDataset(raw: string): Promise<QualityCatalogue> {
  const sha256 = await qualitySha256(raw);
  return parseQualityDataset({ key: `custom-${sha256.slice(0, 32)}`, name: '自定义冻结题集', sha256, dataset: raw });
}

export function assertQualityRunInfo(run: AiKbEvaluationInfo, kbId: number): void {
  if (!run || !positive(run.id) || run.kb_id !== kbId || !uuid(run.request_id) || !label(run.dataset_key) || !digest(run.dataset_sha256) || !label(run.group_key)
    || !['pending', 'running', 'succeeded', 'failed', 'cancelled', 'unknown'].includes(run.status) || !positive(run.case_count) || run.case_count > 30
    || !count(run.completed_count) || !count(run.failed_count) || run.completed_count + run.failed_count > run.case_count || typeof run.stop_requested !== 'boolean' || typeof run.corpus_verified !== 'boolean' || typeof run.error !== 'string'
    || ![run.created_at, run.updated_at, run.started_at, run.completed_at].every(count)) throw new Error('批次状态或知识库归属无效');
}

function validateHits(value: unknown, kbId: number, known: Map<number, QualityInventory>): AiSearchKbHit[] {
  if (!Array.isArray(value) || value.length > 5) throw new Error('检索来源数量无效');
  const ids = new Set<number>(); let previous: AiSearchKbHit | undefined;
  for (const raw of value) {
    if (!object(raw) || !positive(raw.chunk_id) || !positive(raw.doc_id) || raw.kb_id !== kbId || ids.has(raw.chunk_id) || !label(raw.snippet) || !label(raw.doc_title) || typeof raw.score !== 'number' || !Number.isFinite(raw.score)) throw new Error('检索来源归属或内容无效');
    const doc = known.get(raw.doc_id);
    if (!doc || doc.status !== 'ready' || doc.title !== raw.doc_title) throw new Error('检索来源与实际库存不符');
    const hit = raw as unknown as AiSearchKbHit;
    if (previous && (hit.score > previous.score || hit.score === previous.score && hit.chunk_id < previous.chunk_id)) throw new Error('检索来源顺序无效');
    ids.add(hit.chunk_id); previous = hit;
  }
  return value as AiSearchKbHit[];
}

function analysisValue(value: unknown, kbId: number, known: Map<number, QualityInventory>): WorkflowAnalysisResult {
  validateAnalysis(value);
  const result = value as WorkflowAnalysisResult;
  if (encoder.encode(JSON.stringify(result)).length > 24000 || result.sources.some((s, i) => s.index !== i + 1 || s.kb_id !== kbId || known.get(s.doc_id)?.status !== 'ready' || known.get(s.doc_id)?.title !== s.doc_title)
    || new Set(result.sources.map(s => s.chunk_id)).size !== result.sources.length) throw new Error('分析来源或大小无效');
  return result;
}

export async function decodeQualityEvaluation(run: AiKbEvaluationInfo, kbId: number): Promise<QualityRunView> {
  assertQualityRunInfo(run, kbId);
  const catalogue = await parseQualityDataset({ key: run.dataset_key, name: run.dataset_key, sha256: run.dataset_sha256, dataset: run.snapshot });
  const dataset = catalogue.dataset, rawBindings = parseQualityJson(run.bindings), rawContext = parseQualityJson(run.context, 16_384), observations = parseQualityJson(run.observations);
  validateContext(rawContext);
  if (!object(rawBindings) || !Array.isArray(rawBindings.documents) || !Array.isArray(rawBindings.inventory) || !Array.isArray(observations) || observations.length !== run.case_count) throw new Error('批次资料绑定或观测数量无效');
  const inventory = new Map<number, QualityInventory>();
  for (const raw of rawBindings.inventory) {
    if (!object(raw) || !positive(raw.id) || inventory.has(raw.id) || raw.kb_id !== kbId || raw.kb !== run.group_key || !positive(raw.version) || !label(raw.title) || !['pending', 'ready', 'failed'].includes(String(raw.status)) || typeof raw.embedding_model !== 'string') throw new Error('批次库存重复或归属无效');
    inventory.set(raw.id, raw as unknown as QualityInventory);
  }
  const documents = new Map<string, QualityBinding>();
  const expectedDocs = dataset.documents.filter(d => d.kb === run.group_key);
  for (const raw of rawBindings.documents) {
    if (!object(raw) || !label(raw.key) || documents.has(raw.key) || !positive(raw.doc_id) || raw.kb_id !== kbId || raw.kb !== run.group_key || !positive(raw.version) || raw.status !== 'ready' || !label(raw.embedding_model)) throw new Error('批次资料绑定无效');
    const expected = expectedDocs.find(d => d.key === raw.key), listed = inventory.get(raw.doc_id);
    if (!expected || expected.title !== raw.title || expected.source !== raw.source || expected.sha256 !== raw.sha256 || expected.sha256 !== raw.content_sha256 || expected.source_doc_id !== undefined && expected.source_doc_id !== raw.doc_id || expected.source_doc_version !== undefined && expected.source_doc_version !== raw.version || !listed || listed.version !== raw.version || listed.title !== raw.title || listed.status !== raw.status || listed.embedding_model !== raw.embedding_model) throw new Error('资料指纹、版本或库存不符');
    documents.set(raw.key, raw as unknown as QualityBinding);
  }
  if (documents.size !== expectedDocs.length || !documents.size || new Set([...documents.values()].map(d => d.doc_id)).size !== documents.size) throw new Error('批次绑定资料缺失或重复');
  const seen = new Set<string>(), requests = new Set<string>(), tasks = new Set<number>(), rows: QualityRow[] = [];
  let previousCaseIndex = -1;
  for (const raw of observations) {
    if (!object(raw) || !label(raw.case_id) || seen.has(raw.case_id) || !uuid(raw.request_id) || requests.has(raw.request_id) || !count(raw.task_id) || raw.task_id > 0 && tasks.has(raw.task_id)
      || !['waiting', 'running', 'succeeded', 'failed', 'unknown'].includes(String(raw.status)) || typeof raw.stage !== 'string' || typeof raw.error !== 'string') throw new Error('逐题标识、状态或原请求无效');
    const item = dataset.cases.find(c => c.id === raw.case_id);
    if (!item || item.kb !== run.group_key || item.query !== raw.query) throw new Error('观测问题与冻结题集不符');
    const caseIndex = dataset.cases.indexOf(item);
    if (caseIndex <= previousCaseIndex) throw new Error('观测题目顺序与题集不符');
    previousCaseIndex = caseIndex;
    const searchClient = raw.search_client_ms, searchServer = raw.search_server_ms, elapsed = raw.analysis_elapsed_ms;
    if (!finite(searchClient) || !count(searchServer) || !count(elapsed) || elapsed > 110_000) throw new Error('逐题耗时无效');
    const list = validateHits(raw.list, kbId, inventory), status = raw.status as QualityRow['status'];
    if (status === 'succeeded' && [searchClient, searchServer, elapsed].some(ms => ms > 100_000)
      || status === 'waiting' && (raw.task_id !== 0 || list.length !== 0 || searchClient !== 0 || searchServer !== 0 || elapsed !== 0)) throw new Error('逐题耗时与执行状态不符');
    let analysis: WorkflowAnalysisResult | null = null, screen: QualityAnswerScreen | null = null, retrieval: QualityRetrieval | null = null;
    if (status === 'succeeded') {
      if (!positive(raw.task_id) || !raw.analysis) throw new Error('成功项缺少实际任务或分析');
      analysis = analysisValue(raw.analysis, kbId, inventory);
      screen = screenAnswer(item, analysis, kbId, inventory, documents);
      retrieval = evaluateRetrieval(item, list, documents);
    } else if (raw.analysis !== undefined && raw.analysis !== null) throw new Error('未成功项不能填造分析或用量');
    const checks = (item.checks ?? item.answer.required_groups.map((g, i) => ({ label: g.join(' / '), kind: 'fact', required_group: i }))).map(check => ({ label: check.label, kind: check.kind, requiredGroup: check.required_group, matched: screen?.required_groups_matched[check.required_group] ?? false }));
    rows.push({ item, requestId: raw.request_id, taskId: raw.task_id, status, stage: raw.stage, list, analysis, error: raw.error, executionFailed: status === 'failed' || status === 'unknown', searchClientMs: searchClient, searchServerMs: searchServer, analysisMs: elapsed, retrieval, screen, checks });
    seen.add(raw.case_id); requests.add(raw.request_id); if (raw.task_id) tasks.add(raw.task_id);
  }
  const successes = rows.filter(row => row.status === 'succeeded'), failures = rows.filter(row => row.executionFailed);
  if (successes.length !== run.completed_count || failures.length !== run.failed_count || run.status === 'succeeded' && (successes.length !== run.case_count || !run.corpus_verified)) throw new Error('批次计数或完成状态不一致');
  const summary = summarize(successes.map(row => ({ case_id: row.item.id, split: row.item.split, category: row.item.category, analysis: row.analysis, answer_screen: row.screen, retrieval: row.retrieval, search_client_ms: row.searchClientMs, search_server_ms: row.searchServerMs, analysis_elapsed_ms: row.analysisMs })));
  return { run, dataset, context: rawContext as QualityContext, bindings: { documents: [...documents.values()], inventory: [...inventory.values()] }, rows, summary };
}

function canonical(value: unknown): string {
  const visit = (v: unknown): unknown => Array.isArray(v) ? v.map(visit) : object(v) ? Object.fromEntries(Object.keys(v).sort().map(k => [k, visit(v[k])])) : v;
  return JSON.stringify(visit(value));
}
export function compareQualityRuns(before: QualityRunView, after: QualityRunView) {
  const reasons: string[] = [], same = (a: unknown, b: unknown) => canonical(a) === canonical(b);
  if (before.run.id === after.run.id) reasons.push('请选择两个不同批次');
  if ([before, after].some(v => v.run.status !== 'succeeded' || !v.run.corpus_verified || v.rows.some(r => r.status !== 'succeeded'))) reasons.push('只能比较语料已复核的完整批次');
  if (before.run.kb_id !== after.run.kb_id || before.run.dataset_sha256 !== after.run.dataset_sha256 || before.run.group_key !== after.run.group_key || !same(before.rows.map(r => r.item.id).sort(), after.rows.map(r => r.item.id).sort())) reasons.push('题集、问题范围或知识库不同');
  if (!same([...before.bindings.documents].sort((a, b) => a.key.localeCompare(b.key)), [...after.bindings.documents].sort((a, b) => a.key.localeCompare(b.key))) || !same([...before.bindings.inventory].sort((a, b) => a.id - b.id), [...after.bindings.inventory].sort((a, b) => a.id - b.id))) reasons.push('语料版本、正文指纹或库存不同');
  const fixed = (ctx: QualityContext) => Object.fromEntries(Object.entries(ctx).filter(([key]) => !['binary_sha', 'source_revision', 'experiment'].includes(key)));
  if (!same(fixed(before.context), fixed(after.context))) reasons.push('模型、Embedding或预算不同');
  return { comparable: !reasons.length, reasons, versionChanged: before.context.binary_sha !== after.context.binary_sha,
    delta: reasons.length ? null : { flagged: after.summary.answer_rule_screen.flagged_count - before.summary.answer_rule_screen.flagged_count, tokens: after.summary.usage.known_total_tokens - before.summary.usage.known_total_tokens,
      recall5: before.summary.positive.recall_at[5] === null || after.summary.positive.recall_at[5] === null ? null : after.summary.positive.recall_at[5]! - before.summary.positive.recall_at[5]!,
      evidence5: before.summary.positive.evidence_coverage_at[5] === null || after.summary.positive.evidence_coverage_at[5] === null ? null : after.summary.positive.evidence_coverage_at[5]! - before.summary.positive.evidence_coverage_at[5]! } };
}

export function createQualityAttempt(catalogue: QualityCatalogue, group: string, ids: string[], requestId: string): AiStartKbEvaluationReq {
  if (!uuid(requestId) || !ids.length || ids.length > 30 || new Set(ids).size !== ids.length || ids.some(id => catalogue.dataset.cases.find(c => c.id === id)?.kb !== group)) throw new Error('请选择同一题组的1..30道题目');
  return { request_id: requestId, dataset_key: catalogue.key, dataset_sha256: catalogue.sha256, group_key: group, case_ids: catalogue.dataset.cases.filter(c => ids.includes(c.id)).map(c => c.id), ...(catalogue.key.startsWith('custom-') ? { dataset_snapshot: catalogue.raw } : {}) };
}
export function canRepeatQualityRow(parentStatus: AiKbEvaluationStatus, row: Pick<QualityRow, 'status' | 'taskId'>): boolean {
  return row.status === 'succeeded' || row.status === 'failed'
    || row.status === 'waiting' && row.taskId === 0 && (parentStatus === 'failed' || parentStatus === 'cancelled');
}
// 版本号仅隔离活跃请求；持久恢复使用会话及真实部门，避免F5重置版本后丢失原请求。
export const qualityAttemptKey = (scope: AiWorkspaceScope, departmentId: number | null, kbId: number) => `ai-kb-quality:${JSON.stringify([scope[0], departmentId, kbId])}`;
interface Storage { getItem(key: string): string | null; setItem(key: string, value: string): void; removeItem(key: string): void }
export function readQualityAttempt(storage: Storage, key: string): AiStartKbEvaluationReq | null {
  const raw = storage.getItem(key); if (!raw) return null;
  const value = parseQualityJson(raw, 2_000_000);
  if (!object(value) || !uuid(value.request_id) || !label(value.dataset_key) || value.dataset_key.length > 64 || !digest(value.dataset_sha256) || !label(value.group_key) || value.group_key.length > 64 || !strings(value.case_ids) || !value.case_ids.length || value.case_ids.length > 30 || value.case_ids.some(id => id.length > 128) || new Set(value.case_ids).size !== value.case_ids.length) throw new Error('上次评测标识无法恢复，请核对运行记录');
  if (value.dataset_key.startsWith('custom-') && (typeof value.dataset_snapshot !== 'string' || encoder.encode(value.dataset_snapshot).length > 262_144 || value.dataset_key !== `custom-${value.dataset_sha256.slice(0, 32)}`)) throw new Error('原自定义题集快照无法恢复');
  if (!value.dataset_key.startsWith('custom-') && value.dataset_snapshot !== undefined) throw new Error('内置题集不能附加自定义快照');
  return { request_id: value.request_id, dataset_key: value.dataset_key, dataset_sha256: value.dataset_sha256, group_key: value.group_key, case_ids: [...value.case_ids], ...(typeof value.dataset_snapshot === 'string' ? { dataset_snapshot: value.dataset_snapshot } : {}) };
}
export function assertQualityAttemptResult(run: AiKbEvaluationInfo, request: AiStartKbEvaluationReq) {
  if (run.request_id !== request.request_id || run.dataset_key !== request.dataset_key || run.dataset_sha256 !== request.dataset_sha256 || run.group_key !== request.group_key || run.case_count !== request.case_ids.length) throw new Error('批次与原提交内容不符');
  // 列表摘要没有snapshot；有详情时必须逐字核对，不能借相同UUID换题集。
  if (request.dataset_snapshot !== undefined && run.snapshot !== request.dataset_snapshot) throw new Error('批次快照与原提交内容不符');
  if (run.observations) {
    const observed = parseQualityJson(run.observations);
    if (!Array.isArray(observed) || observed.length !== request.case_ids.length || observed.some(row => !object(row) || typeof row.case_id !== 'string' || !request.case_ids.includes(row.case_id)) || new Set(observed.map(row => (row as Record<string, unknown>).case_id)).size !== observed.length) throw new Error('批次题目与原提交内容不符');
  }
}
export function writeQualityAttempt(storage: Storage, key: string, value: AiStartKbEvaluationReq) { storage.setItem(key, JSON.stringify(value)); }
export function clearQualityAttempt(storage: Storage, key: string) { storage.removeItem(key); }
export const qualityGroupLabel = (key: string) => ({ project_ops: '项目运维资料', project_ai: 'AI平台资料' })[key] ?? key;
export const qualityStatusLabels = { pending: '等待执行', running: '执行中', succeeded: '已完成，待复核', failed: '执行失败', cancelled: '已停止', unknown: '结果待确认' } as const;
export const qualityReasonLabel = (key: string) => ({ missing_required_facts: '可能遗漏检查项', missing_insufficiency: '资料不足说明待核对', missing_scoped_insufficiency: '局部未知说明待核对', additional_absence_claim_needs_review: '可能存在额外不足措辞', cited_evidence_incomplete: '引用片段依据待核对', missing_expected_citations: '预期文档引用待核对', missing_citation: '缺少引用', invalid_citation: '引用编号无效', malformed_citation: '引用格式待核对', source_scope_mismatch: '来源归属待核对', duplicate_source_chunk: '来源重复', unbound_source: '来源不在冻结绑定中', truncated_answer: '回答已截断', no_matches_for_answerable: '未召回预期依据', forbidden_phrase: '出现禁答内容' })[key] ?? '需要复核';
