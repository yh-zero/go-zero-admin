import { createHash } from 'node:crypto';
import { open, readFile, realpath, stat, unlink } from 'node:fs/promises';
import { basename, dirname, extname, isAbsolute, relative, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { isDeepStrictEqual } from 'node:util';
import { loadDataset } from './eval-kb-quality.mjs';
import { normalized, validateAnalysis } from './eval-kb-quality-metrics.mjs';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const hash = value => createHash('sha256').update(value).digest('hex');
const object = value => value !== null && typeof value === 'object' && !Array.isArray(value);
const positive = value => Number.isSafeInteger(value) && value > 0;
const text = (value, limit, allowEmpty = false) => typeof value === 'string' && value.isWellFormed()
  && Buffer.byteLength(value) <= limit && (allowEmpty || value.trim().length > 0);
const within = (root, path) => { const part = relative(root, path); return part !== '' && !isAbsolute(part) && part !== '..' && !part.startsWith('../') && !part.startsWith('..\\'); };
// chunkText只在双LF段落边界TrimSpace后重接；不删段内空白、JSON标点或变换数字。
const paragraphText = value => value.split('\n\n').map(part => part.trim()).join('\n\n').trim();
const bodyContains = (body, excerpt) => body.includes(excerpt) || paragraphText(body).includes(paragraphText(excerpt));
// 与kb_analyze的来源标题预算一致；冻结素材和文档快照仍用完整标题匹配。
function sourceTitle(title) {
  let prefix = '', size = 0;
  for (const rune of title) {
    const bytes = Buffer.byteLength(rune); if (size + bytes > 256) break;
    prefix += rune; size += bytes;
  }
  return prefix;
}

function exactKeys(value, names, label) {
  if (!object(value) || Object.keys(value).length !== names.length || names.some(name => !Object.hasOwn(value, name))) throw new Error(`${label}字段不完整或包含额外字段`);
}

// JSON.parse会静默覆盖重复键；导出指纹和审核对象必须具有唯一解释。
function parseJSON(bytes, limit, label) {
  if (!(bytes instanceof Uint8Array) || bytes.byteLength > limit) throw new Error(`${label}字节数超限或类型无效`);
  let raw, parsed;
  try { raw = new TextDecoder('utf-8', { fatal: true }).decode(bytes); parsed = JSON.parse(raw); }
  catch { throw new Error(`${label}不是有效UTF-8 JSON`); }
  let offset = 0;
  const whitespace = () => { while (/\s/u.test(raw[offset] ?? '') && offset < raw.length) offset++; };
  function stringToken() {
    const start = offset++;
    while (offset < raw.length) {
      if (raw[offset] === '\\') { offset += 2; continue; }
      if (raw[offset++] === '"') break;
    }
    return JSON.parse(raw.slice(start, offset));
  }
  function value(depth) {
    if (depth > 32) throw new Error(`${label}嵌套过深`);
    whitespace();
    if (raw[offset] === '{') {
      offset++; whitespace(); const keys = new Set();
      if (raw[offset] === '}') { offset++; return; }
      while (true) {
        whitespace(); const key = stringToken();
        if (keys.has(key) || ['__proto__', 'prototype', 'constructor'].includes(key)) throw new Error(`${label}含重复或不安全的JSON键`);
        keys.add(key); whitespace(); offset++; value(depth + 1); whitespace();
        if (raw[offset++] === '}') return;
      }
    }
    if (raw[offset] === '[') {
      offset++; whitespace(); if (raw[offset] === ']') { offset++; return; }
      while (true) { value(depth + 1); whitespace(); if (raw[offset++] === ']') return; }
    }
    if (raw[offset] === '"') { const entry = stringToken(); if (!entry.isWellFormed()) throw new Error(`${label}含非法Unicode字符串`); return; }
    while (offset < raw.length && !/[\s,}\]]/u.test(raw[offset])) offset++;
  }
  value(0); return parsed;
}

async function readBounded(filename, limit, label) {
  let info;
  try { info = await stat(filename); } catch { throw new Error(`${label}不可读`); }
  if (!info.isFile() || info.size > limit) throw new Error(`${label}不是文件或字节数超限`);
  const bytes = await readFile(filename);
  if (bytes.length > limit) throw new Error(`${label}字节数超限`);
  return bytes;
}

function validateReview(review) {
  exactKeys(review, ['answerable', 'expected_answer', 'required_facts', 'evidence', 'reason'], '复核结论');
  if (typeof review.answerable !== 'boolean' || !text(review.expected_answer, 4000) || !text(review.reason, 1000)
    || !Array.isArray(review.required_facts) || review.required_facts.length > 10 || review.required_facts.some(fact => !text(fact, 200))
    || new Set(review.required_facts.map(normalized)).size !== review.required_facts.length || !Array.isArray(review.evidence) || review.evidence.length > 5) throw new Error('复核结论类型、字节数或事实项无效');
  for (const row of review.evidence) {
    exactKeys(row, ['source_index', 'quote'], '复核证据');
    if (!positive(row.source_index) || !text(row.quote, 500)) throw new Error('复核证据索引或字节数无效');
  }
  if (new Set(review.evidence.map(row => JSON.stringify([row.source_index, row.quote]))).size !== review.evidence.length) throw new Error('复核证据重复');
  if (review.answerable ? !review.required_facts.length || !review.evidence.length : review.required_facts.length || review.evidence.length) throw new Error('有据与资料不足题的事实项和证据数量不一致');
}

function validateExport(bytes) {
  const exported = parseJSON(bytes, 512 * 1024, '反馈导出');
  exactKeys(exported, ['version', 'type', 'feedback'], '反馈导出');
  if (exported.version !== 1 || exported.type !== 'kb_feedback_regression') throw new Error('反馈导出版本或类型不支持');
  const feedback = exported.feedback;
  exactKeys(feedback, ['id', 'kb_id', 'task_id', 'step_index', 'feedback', 'note', 'status', 'revision', 'question', 'answer', 'snapshot', 'review', 'created_at', 'updated_at', 'reviewed_at'], '反馈记录');
  if (![feedback.id, feedback.kb_id, feedback.task_id, feedback.revision].every(positive)
    || !Number.isSafeInteger(feedback.step_index) || feedback.step_index < 0 || feedback.step_index > 31 || feedback.status !== 'accepted'
    || !['helpful', 'incorrect_answer', 'missing_knowledge', 'irrelevant_source'].includes(feedback.feedback)
    || !text(feedback.note, 1000, true) || !text(feedback.question, 2000) || !text(feedback.answer, 24000)
    || !text(feedback.snapshot, 256 * 1024) || ![feedback.created_at, feedback.updated_at, feedback.reviewed_at].every(positive)
    || feedback.created_at > feedback.reviewed_at || feedback.reviewed_at > feedback.updated_at) throw new Error('只接受字段、状态、版本和时间一致的已采纳反馈');
  validateReview(feedback.review);
  const snapshot = parseJSON(Buffer.from(feedback.snapshot), 256 * 1024, '反馈快照');
  exactKeys(snapshot, ['version', 'kb_id', 'query', 'step_input', 'step_output', 'sources', 'documents'], '反馈快照');
  if (snapshot.version !== 1 || snapshot.kb_id !== feedback.kb_id || snapshot.query !== feedback.question
    || !text(snapshot.step_input, 16000) || !text(snapshot.step_output, 24000) || !Array.isArray(snapshot.sources)
    || !Array.isArray(snapshot.documents) || snapshot.documents.length > 8) throw new Error('反馈快照与原问题、知识库或步骤不一致');
  const input = parseJSON(Buffer.from(snapshot.step_input), 16000, '步骤输入');
  exactKeys(input, ['kb_id', 'query'], '步骤输入');
  if (input.kb_id !== feedback.kb_id || input.query !== feedback.question) throw new Error('实际步骤输入与反馈问题不一致');
  const output = parseJSON(Buffer.from(snapshot.step_output), 24000, '步骤输出');
  exactKeys(output, ['version', 'type', 'status', 'analysis', 'sources', 'usage'], '步骤输出');
  validateAnalysis(output);
  if (output.analysis !== feedback.answer || !isDeepStrictEqual(output.sources, snapshot.sources)) throw new Error('实际步骤输出与反馈答案或来源不一致');
  const chunks = new Set();
  for (const [index, source] of snapshot.sources.entries()) {
    exactKeys(source, ['index', 'chunk_id', 'kb_id', 'doc_id', 'doc_title', 'snippet', 'score'], '分析来源');
    if (source.index !== index + 1 || source.kb_id !== feedback.kb_id || chunks.has(source.chunk_id)) throw new Error('分析来源顺序、知识库或分块重复');
    chunks.add(source.chunk_id);
  }
  return { feedback, snapshot };
}

export async function mergeFeedback({ feedbackBytes, casesFile, kbMap }) {
  const { feedback, snapshot } = validateExport(feedbackBytes);
  if (!object(kbMap) || !Object.values(kbMap).every(positive) || new Set(Object.values(kbMap)).size !== Object.values(kbMap).length) throw new Error('知识库映射缺失、重复或无效');
  const before = await readBounded(casesFile, 2 * 1024 * 1024, '原题集');
  const original = parseJSON(before, 2 * 1024 * 1024, '原题集'), dataset = await loadDataset(casesFile);
  if (!isDeepStrictEqual(original, dataset) || hash(before) !== hash(await readBounded(casesFile, 2 * 1024 * 1024, '原题集'))) throw new Error('读取期间原题集变化，停止转换');
  if (dataset.cases.length >= 100) throw new Error('题集已达100题上限，请显式创建新的回归题集');
  const selected = [...new Set(dataset.documents.map(doc => doc.kb))].filter(key => kbMap[key] === feedback.kb_id);
  if (selected.length !== 1) throw new Error('反馈知识库ID无法唯一映射到原题集');
  const kb = selected[0], verified = new Map();
  for (const doc of snapshot.documents) {
    exactKeys(doc, ['doc_id', 'kb_id', 'title', 'version', 'content_sha256', 'status'], '快照文档');
    if (!positive(doc.doc_id) || doc.kb_id !== feedback.kb_id || !positive(doc.version) || doc.status !== 'verified'
      || !text(doc.title, 512) || !/^[a-f0-9]{64}$/u.test(doc.content_sha256) || verified.has(doc.doc_id)) throw new Error('历史资料状态、版本、归属或指纹不可核对；拒绝替换当前资料');
    const candidates = dataset.documents.filter(source => source.kb === kb && source.title === doc.title && source.sha256 === doc.content_sha256);
    if (candidates.length !== 1) throw new Error('历史资料标题或SHA与冻结语料不唯一匹配，须先复核');
    const source = candidates[0], bytes = await readFile(resolve(repositoryRoot, source.source));
    if (hash(bytes) !== source.sha256) throw new Error('读取期间冻结素材变化，停止转换');
    verified.set(doc.doc_id, { ...source, content: new TextDecoder('utf-8', { fatal: true }).decode(bytes) });
  }
  const sourceDocuments = new Set(snapshot.sources.map(source => source.doc_id));
  if (sourceDocuments.size !== verified.size || [...sourceDocuments].some(id => !verified.has(id))) throw new Error('来源与快照文档集合不一致');
  for (const source of snapshot.sources) {
    const doc = verified.get(source.doc_id);
    if (source.doc_title !== sourceTitle(doc.title) || !bodyContains(doc.content, source.snippet)) throw new Error('历史来源标题或片段无法在同指纹正文按段落边界核对，须重新核对资料');
  }
  const evidence = new Map();
  for (const row of feedback.review.evidence) {
    const source = snapshot.sources.find(item => item.index === row.source_index), doc = source && verified.get(source.doc_id);
    if (!source || !source.snippet.includes(row.quote) || !bodyContains(doc.content, row.quote)) throw new Error('复核引用不存在于实际历史来源和冻结正文');
    if (!evidence.has(doc.key)) evidence.set(doc.key, []);
    const groups = evidence.get(doc.key); if (!groups.some(group => group[0] === row.quote)) groups.push([row.quote]);
  }
  for (const fact of feedback.review.required_facts) {
    if (!normalized(feedback.review.expected_answer).includes(normalized(fact))
      || !feedback.review.evidence.some(row => normalized(row.quote).includes(normalized(fact)))) throw new Error('事实检查项未同时属于预期回答及已核对摘录，须先复核');
  }
  const id = `feedback-${feedback.kb_id}-${feedback.id}-r${feedback.revision}`;
  const metadata = dataset.metadata ?? {};
  if (!object(metadata) || (metadata.feedback_regressions !== undefined && !Array.isArray(metadata.feedback_regressions))) throw new Error('原题集反馈来源元数据无效');
  const history = metadata.feedback_regressions ?? [];
  if (history.some(row => !object(row) || !positive(row.feedback_id) || !positive(row.kb_id) || !positive(row.revision)
    || !text(row.case_id, 200) || !/^[a-f0-9]{64}$/u.test(row.export_sha256) || !/^[a-f0-9]{64}$/u.test(row.base_dataset_sha256)
    || !dataset.cases.some(item => item.id === row.case_id))) throw new Error('原题集已有反馈来源记录不完整，停止转换');
  if (dataset.cases.some(item => item.id === id) || history.some(item => item?.feedback_id === feedback.id && item?.kb_id === feedback.kb_id)) throw new Error('该反馈已有回归记录；不能重复、静默覆盖或混入另一revision');
  const result = structuredClone(dataset);
  result.cases.push({ id, split: 'development', category: 'feedback_regression', kb, query: feedback.question,
    expected_documents: [...evidence.keys()], evidence: [...evidence].map(([document, groups]) => ({ document, groups })),
    answer: { kind: feedback.review.answerable ? 'answerable' : 'unanswerable', required_groups: feedback.review.required_facts.map(fact => [fact]), forbidden_phrases: [] },
    expected_answer_example: feedback.review.expected_answer,
    rationale: feedback.review.reason });
  result.metadata = { ...metadata, feedback_regressions: [...history, { case_id: id, feedback_id: feedback.id, revision: feedback.revision,
    kb_id: feedback.kb_id, task_id: feedback.task_id, step_index: feedback.step_index, reviewed_at: feedback.reviewed_at,
    export_sha256: hash(feedbackBytes), base_dataset_sha256: hash(before), snapshot_sha256: hash(feedback.snapshot),
    review_origin: 'exported_review_record', human_identity_verified: false,
    note: '来源指纹记录文件一致性，不证明导出签名或审核者身份；正文指纹属于反馈采集时核对，不是模型调用时全库快照。' }] };
  return result;
}

async function outputPath(filename) {
  if (typeof filename !== 'string' || !filename || extname(filename) !== '.json') throw new Error('output必须是tmp下新的JSON文件');
  const target = resolve(filename), parent = await realpath(dirname(target)), tmpRoot = await realpath(resolve(repositoryRoot, 'tmp'));
  const actualRoot = await realpath(repositoryRoot);
  if (!within(actualRoot, tmpRoot) || (parent !== tmpRoot && !within(tmpRoot, parent))) throw new Error('output必须位于仓库忽略的tmp，拒绝路径或符号链接越界');
  return resolve(parent, basename(target));
}

export async function convertFeedbackFiles({ feedbackFile, casesFile, kbMapFile, output }) {
  const target = await outputPath(output);
  const feedbackBytes = await readBounded(feedbackFile, 512 * 1024, '反馈导出');
  const kbMap = parseJSON(await readBounded(kbMapFile, 64 * 1024, '知识库映射'), 64 * 1024, '知识库映射');
  const merged = await mergeFeedback({ feedbackBytes, casesFile, kbMap });
  let handle;
  try { handle = await open(target, 'wx', 0o600); }
  catch { throw new Error('输出已存在或无法独占创建，原文件未覆盖'); }
  try {
    await handle.writeFile(JSON.stringify(merged, null, 2) + '\n', 'utf8'); await handle.sync(); await handle.close(); handle = null;
    await loadDataset(target);
  } catch {
    if (handle) await handle.close().catch(() => {});
    await unlink(target);
    throw new Error('新副本写入或题集复验失败，已删除本次创建的不完整输出');
  }
  return { mode: 'offline_feedback_merge', http_requests: 0, case_id: merged.cases.at(-1).id, total_cases: merged.cases.length,
    output: relative(repositoryRoot, target), output_sha256: hash(await readFile(target)) };
}

export function parseArgs(args) {
  const options = {}, names = { '--feedback': 'feedbackFile', '--cases': 'casesFile', '--kb-map': 'kbMapFile', '--output': 'output' };
  for (let index = 0; index < args.length; index++) {
    const key = args[index], name = Object.hasOwn(names, key) ? names[key] : undefined;
    if (!name || Object.hasOwn(options, name) || !args[index + 1] || args[index + 1].startsWith('--')) throw new Error('参数缺失、重复或不支持；仅接受--feedback/--cases/--kb-map/--output');
    options[name] = args[++index];
  }
  if (Object.keys(options).length !== 4) throw new Error('须指定--feedback、--cases、--kb-map和tmp下新的--output；转换全程离线');
  return options;
}

async function main() { console.log(JSON.stringify(await convertFeedbackFiles(parseArgs(process.argv.slice(2))))); }
if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) main()
  .catch(error => { console.error(error.message); process.exitCode = 1; });
