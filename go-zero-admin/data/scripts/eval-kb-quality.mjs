import { createHash, randomUUID } from 'node:crypto';
import { readFile, realpath, writeFile } from 'node:fs/promises';
import { dirname, resolve, relative, isAbsolute } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { performance } from 'node:perf_hooks';
import { validateHits, openReport } from './eval-kb-recall.mjs';
import { normalized, evaluateRetrieval, screenAnswer, summarize, validateAnalysis } from './eval-kb-quality-metrics.mjs';
import { readRecords, parseRecords, renderReport, compareRuns, renderComparison, rescoreRecords } from './eval-kb-quality-report.mjs';
import { validateContext } from './eval-kb-quality-context.mjs';

export { openReport, evaluateRetrieval, screenAnswer, summarize, readRecords, renderReport, compareRuns, rescoreRecords, validateContext };
const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const defaultCases = resolve(repositoryRoot, 'ai_document/ai-rag/quality-cases.json');
const positiveID = value => Number.isSafeInteger(value) && value > 0;
const nonnegative = value => Number.isSafeInteger(value) && value >= 0;
const hash = value => createHash('sha256').update(value).digest('hex');
const groupsValid = groups => Array.isArray(groups) && groups.every(group => Array.isArray(group) && group.length > 0 && group.every(value => typeof value === 'string' && value.trim()));
const within = (root, path) => { const location = relative(root, path); return !!location && !isAbsolute(location) && location !== '..' && !location.startsWith('..\\') && !location.startsWith('../'); };
const pause = ms => new Promise(resolvePromise => setTimeout(resolvePromise, ms));

async function readJSON(filename, label) {
  try { return JSON.parse(await readFile(filename, 'utf8')); }
  catch { throw new Error(`${label}不可读或不是有效JSON`); }
}

export async function loadDataset(filename = defaultCases) {
  const dataset = await readJSON(filename, '题集');
  if (dataset.version !== 1 || !Array.isArray(dataset.documents) || !dataset.documents.length || !Array.isArray(dataset.cases) || !dataset.cases.length || dataset.cases.length > 100) throw new Error('题集结构或数量无效');
  const sourceRoot = await realpath(resolve(repositoryRoot, 'ai_document/ai-rag')), documents = new Map();
  for (const doc of dataset.documents) {
    if (!doc || typeof doc.key !== 'string' || !doc.key || documents.has(doc.key) || typeof doc.kb !== 'string' || !doc.kb || typeof doc.title !== 'string' || !doc.title.trim()
      || typeof doc.source !== 'string' || isAbsolute(doc.source) || !/^[a-f0-9]{64}$/.test(doc.sha256)) throw new Error('素材字段无效或重复');
    let bytes;
    try { const path = await realpath(resolve(repositoryRoot, doc.source)); if (!within(sourceRoot, path)) throw new Error('outside'); bytes = await readFile(path); }
    catch { throw new Error(`素材不可读或越界：${doc.key}`); }
    if (hash(bytes) !== doc.sha256) throw new Error(`素材SHA256不一致：${doc.key}`);
    let content; try { content = new TextDecoder('utf-8', { fatal: true }).decode(bytes); } catch { throw new Error(`素材不是UTF-8：${doc.key}`); }
    documents.set(doc.key, { ...doc, content });
  }
  const ids = new Set();
  for (const item of dataset.cases) {
    if (!item || typeof item.id !== 'string' || !item.id || ids.has(item.id) || !['development', 'validation'].includes(item.split)
      || typeof item.category !== 'string' || !item.category || typeof item.kb !== 'string' || !dataset.documents.some(doc => doc.kb === item.kb)
      || typeof item.query !== 'string' || !item.query.trim() || Buffer.byteLength(item.query) > 2000 || typeof item.rationale !== 'string' || !item.rationale.trim()
      || !Array.isArray(item.expected_documents) || new Set(item.expected_documents).size !== item.expected_documents.length || !Array.isArray(item.evidence)
      || !item.answer || !['answerable', 'unanswerable'].includes(item.answer.kind) || !groupsValid(item.answer.required_groups)
      || !Array.isArray(item.answer.forbidden_phrases) || item.answer.forbidden_phrases.some(value => typeof value !== 'string' || !value.trim())) throw new Error('题目字段、ID、答案规则或split无效');
    if (item.answer.absence_policy !== undefined && !['forbidden', 'required'].includes(item.answer.absence_policy)) throw new Error('answer.absence_policy仅支持forbidden或required');
    ids.add(item.id);
    if (item.answer.kind === 'unanswerable') {
      if (item.expected_documents.length || item.evidence.length || item.answer.required_groups.length) throw new Error(`负例不能标注预期文档、证据或必需事实：${item.id}`);
      continue;
    }
    if (!item.answer.required_groups.length || !item.expected_documents.length || item.expected_documents.some(key => documents.get(key)?.kb !== item.kb)
      || item.evidence.length !== item.expected_documents.length || new Set(item.evidence.map(row => row.document)).size !== item.evidence.length) throw new Error(`预期文档或证据数量无效：${item.id}`);
    for (const evidence of item.evidence) {
      const doc = documents.get(evidence.document);
      if (!doc || !item.expected_documents.includes(evidence.document) || !groupsValid(evidence.groups) || !evidence.groups.length
        || evidence.groups.some(group => !group.some(value => normalized(doc.content).includes(normalized(value))))) throw new Error(`证据组不在对应原文：${item.id}`);
    }
    if (item.answer.required_groups.some(group => !group.some(value => item.expected_documents.some(key => normalized(documents.get(key).content).includes(normalized(value)))))) throw new Error(`必需事实组不在预期原文：${item.id}`);
  }
  return dataset;
}

export function parseArgs(args) {
  const options = { run: false, base: 'http://127.0.0.1:7001', cases: defaultCases, split: 'all', phase: 'baseline' };
  const values = { '--auth-file': 'authFile', '--kb-map': 'kbMapFile', '--context-file': 'contextFile', '--output': 'output', '--base': 'base', '--cases': 'cases', '--split': 'split', '--case-ids': 'caseIDs', '--phase': 'phase', '--report': 'report', '--rescore': 'rescore' };
  const seen = new Set();
  for (let index = 0; index < args.length; index++) {
    const key = args[index]; if (seen.has(key)) throw new Error('参数重复'); seen.add(key);
    if (key === '--run') options.run = true;
    else if (key === '--plan') options.plan = true;
    else if (key === '--compare') {
      if (!args[index + 1] || !args[index + 2] || args[index + 1].startsWith('--') || args[index + 2].startsWith('--')) throw new Error('--compare需要baseline和candidate两个JSONL');
      options.compare = [args[++index], args[++index]];
    } else {
      if (!values[key] || !args[index + 1] || args[index + 1].startsWith('--')) throw new Error('未知参数或参数缺值');
      options[values[key]] = args[++index];
    }
  }
  if ([options.run, options.plan, !!options.report, !!options.compare, !!options.rescore].filter(Boolean).length > 1) throw new Error('--plan/--run/--report/--compare/--rescore不能混用');
  if (options.rescore && !options.output) throw new Error('--rescore需要新的output路径');
  if (options.rescore && ['--split', '--case-ids', '--phase'].some(key => seen.has(key))) throw new Error('--rescore保留原始题目范围与phase，不接受选择或改写');
  if (!['development', 'validation', 'all'].includes(options.split)) throw new Error('split仅支持development/validation/all');
  if (!['baseline', 'candidate'].includes(options.phase)) throw new Error('phase仅支持baseline/candidate');
  let base; try { base = new URL(options.base); } catch { throw new Error('base不是有效本机HTTP地址'); }
  if (base.protocol !== 'http:' || !['127.0.0.1', 'localhost', '[::1]'].includes(base.hostname) || base.username || base.password || base.pathname !== '/' || base.search || base.hash) throw new Error('base仅允许本机HTTP根地址');
  options.base = base.origin;
  if (options.run && (!options.authFile || !options.kbMapFile || !options.contextFile || !options.output)) throw new Error('--run需要认证auth-file、kb-map、context-file和新output');
  if (options.output) { options.output = resolve(options.output); if (!within(resolve(repositoryRoot, 'tmp'), options.output)) throw new Error('output必须位于仓库忽略的tmp目录'); }
  if (options.caseIDs) {
    options.caseIDs = options.caseIDs.split(',');
    if (options.caseIDs.some(id => !id) || new Set(options.caseIDs).size !== options.caseIDs.length) throw new Error('case-ids为空或重复');
  }
  return options;
}

export function createApiClient({ token, base = 'http://127.0.0.1:7001', fetchImpl = fetch, timeoutMs = 12_000 }) {
  if (typeof token !== 'string' || !token.trim()) throw new Error('认证令牌缺失');
  const counts = { total: 0, get: 0, post: 0, search: 0, analysis_submissions: 0, task_reads: 0, failed: 0 };
  const client = async (path, method = 'GET', body, requestOptions = {}) => {
    counts.total++; counts[method.toLowerCase()]++;
    if (path.endsWith('/search')) counts.search++;
    if (path === '/tasks/run') counts.analysis_submissions++;
    if (/^\/tasks\/\d+$/.test(path)) counts.task_reads++;
    try {
      let response;
      try { response = await fetchImpl(base + '/v1/sys/ai' + path, { method, headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(Math.max(1, Math.ceil(requestOptions.timeoutMs ?? timeoutMs))), redirect: 'error' }); }
      catch (error) { throw new Error(['TimeoutError', 'AbortError'].includes(error?.name) ? '请求超时或取消，结果未知；未自动重试' : '网络请求失败，结果未知；未自动重试'); }
      if (response.status !== 200) throw new Error(`HTTP ${response.status}，结果可能未知；未自动重试`);
      let envelope; try { envelope = await response.json(); } catch { throw new Error('响应JSON无效，结果未知；未自动重试'); }
      if (!envelope || envelope.code !== 200 || !envelope.result || typeof envelope.result !== 'object' || Array.isArray(envelope.result)) throw new Error(`业务响应失败 code=${Number.isInteger(envelope?.code) ? envelope.code : 'invalid'}；不计为空召回或错误回答`);
      return envelope.result;
    } catch (error) { counts.failed++; throw error; }
  };
  client.counts = counts; return client;
}

async function listDocuments(kbMap, kbKeys, client) {
  const byKb = new Map();
  for (const key of kbKeys) {
    const known = new Map(), kbID = kbMap[key]; let expectedTotal;
    for (let page = 1; page <= 1000; page++) {
      const response = await client(`/kbs/${kbID}/docs?page=${page}&page_size=100`);
      if (!nonnegative(response.total) || !Array.isArray(response.list) || (expectedTotal !== undefined && expectedTotal !== response.total)) throw new Error('文档分页格式或总数变化');
      expectedTotal = response.total;
      for (const doc of response.list) {
        if (!doc || !positiveID(doc.id) || doc.kb_id !== kbID || known.has(doc.id) || !positiveID(doc.version) || typeof doc.title !== 'string' || !doc.title.trim()
          || !['ready', 'pending', 'failed'].includes(doc.status) || typeof doc.embedding_model !== 'string') throw new Error('文档列表归属、版本或ID字段无效');
        known.set(doc.id, doc);
      }
      if (known.size === expectedTotal) break;
      if (known.size > expectedTotal || !response.list.length || page === 1000) throw new Error('文档分页数量不一致或超限');
    }
    byKb.set(key, known);
  }
  return byKb;
}
function inventoryOf(byKb) {
  return [...byKb].flatMap(([key, docs]) => [...docs.values()].map(doc => ({ kb: key, id: doc.id, kb_id: doc.kb_id, title: doc.title, version: doc.version, status: doc.status, embedding_model: doc.embedding_model })))
    .sort((a, b) => a.kb.localeCompare(b.kb) || a.id - b.id);
}
async function readBoundDocument(source, listed, client) {
  const response = await client(`/kbs/${listed.kb_id}/docs/${listed.id}`), doc = response.doc;
  if (!doc || doc.id !== listed.id || doc.kb_id !== listed.kb_id || doc.title !== source.title || doc.status !== 'ready' || doc.version !== listed.version || !positiveID(doc.version)
    || !doc.embedding_model || doc.embedding_model !== listed.embedding_model || typeof response.content !== 'string' || hash(response.content) !== source.sha256) throw new Error(`语料正文SHA256、ready、版本或模型不一致/变化：${source.key}`);
  return { ...source, doc_id: doc.id, kb_id: doc.kb_id, version: doc.version, status: doc.status, embedding_model: doc.embedding_model, content_sha256: hash(response.content) };
}

export function parseAnalysis(task) {
  const steps = task.steps;
  if (!Array.isArray(steps) || steps.length !== 1 || steps[0].index !== 0 || steps[0].tool !== 'kb_analyze' || steps[0].status !== 'succeeded' || typeof steps[0].output !== 'string') throw new Error('分析步骤记录格式无效，不能计作回答');
  let result; try { result = JSON.parse(steps[0].output); } catch { throw new Error('分析步骤输出不是有效JSON'); }
  return validateAnalysis(result);
}

export async function runEvaluation({ dataset, kbMap, client, onProgress = async () => {}, pollIntervalMs = 750, taskTimeoutMs = 100_000, now = () => performance.now() }) {
  if (!(taskTimeoutMs > 0 && taskTimeoutMs <= 100_000) || !(pollIntervalMs >= 0 && pollIntervalMs <= 5000)) throw new Error('任务超时须≤100秒，轮询间隔无效');
  const kbKeys = [...new Set(dataset.cases.map(item => item.kb))];
  if (kbKeys.some(key => !positiveID(kbMap[key])) || new Set(kbKeys.map(key => kbMap[key])).size !== kbKeys.length) throw new Error('知识库映射缺失、无效或重复');
  const knownByKb = await listDocuments(kbMap, kbKeys, client), inventory = inventoryOf(knownByKb), bindings = new Map();
  for (const source of dataset.documents.filter(doc => kbKeys.includes(doc.kb))) {
    const matches = [...knownByKb.get(source.kb).values()].filter(doc => doc.title === source.title);
    if (matches.length !== 1) throw new Error(`文档标题不唯一或缺失：${source.key}`);
    bindings.set(source.key, await readBoundDocument(source, matches[0], client));
  }
  await onProgress({ type: 'bindings', documents: [...bindings.values()], inventory });
  const results = [], calls = { search_requests: 0, analysis_submissions: 0, accepted_tasks: 0, task_reads: 0, completed_analyses: 0, known_chat_calls: 0, known_tool_search_calls: 0 };
  for (const item of dataset.cases) {
    const kbID = kbMap[item.kb], known = knownByKb.get(item.kb);
    await onProgress({ type: 'case_started', case_id: item.id, split: item.split, category: item.category, kb_id: kbID });
    const searchStarted = now(); calls.search_requests++;
    const response = await client(`/kbs/${kbID}/search`, 'POST', { query: item.query });
    const searchClient = now() - searchStarted;
    if (!nonnegative(response.elapsed_ms)) throw new Error('search elapsed_ms字段无效');
    const list = validateHits(response.list, kbID, known), retrieval = evaluateRetrieval(item, list, bindings);
    await onProgress({ type: 'search_result', case_id: item.id, kb_id: kbID, query: item.query, search_client_ms: searchClient, search_server_ms: response.elapsed_ms, list, retrieval });
    const requestID = randomUUID(), payload = { request_id: requestID, name: `eval-kb-quality-${item.id.slice(0, 40)}-${requestID}`,
      definition: JSON.stringify({ steps: [{ name: '知识库分析', tool: 'kb_analyze', input: { kb_id: kbID, query: item.query } }] }), input: '{}' };
    // 先sync JSONL再POST；一切失败只停止，禁止换UUID重发不可知的模型调用。
    await onProgress({ type: 'analysis_requested', case_id: item.id, request_id: requestID, payload });
    const started = now(); calls.analysis_submissions++;
    const accepted = await client('/tasks/run', 'POST', payload, { timeoutMs: Math.min(30_000, taskTimeoutMs) });
    if (!positiveID(accepted.task_id) || !['pending', 'running', 'succeeded', 'failed', 'cancelled', 'rejected', 'pending_approval'].includes(accepted.status)) throw new Error('任务接受响应无效，结果未知；不重发');
    calls.accepted_tasks++;
    await onProgress({ type: 'analysis_accepted', case_id: item.id, request_id: requestID, task_id: accepted.task_id, status: accepted.status });
    if (!['pending', 'running', 'succeeded'].includes(accepted.status)) throw new Error(`已接受任务状态${accepted.status}，停止且不重发`);
    let task, analysisElapsed;
    while (true) {
      const remaining = taskTimeoutMs - (now() - started);
      if (remaining <= 0) throw new Error(`任务轮询超时；保留原UUID和任务ID，不重发：${item.id}`);
      calls.task_reads++;
      const detail = await client(`/tasks/${accepted.task_id}`, 'GET', undefined, { timeoutMs: Math.min(12_000, remaining) }); task = detail.task;
      if (!task || task.id !== accepted.task_id || task.workflow_id !== 0 || task.request_id !== requestID) throw new Error('任务ID、临时任务或原UUID不一致');
      const elapsed = now() - started;
      if (elapsed > taskTimeoutMs) throw new Error(`任务轮询超时；不重发：${item.id}`);
      if (['succeeded', 'failed', 'rejected', 'cancelled'].includes(task.status)) {
        analysisElapsed = elapsed;
        await onProgress({ type: 'analysis_terminal', case_id: item.id, request_id: requestID, task_id: task.id, status: task.status });
        if (task.status !== 'succeeded') throw new Error(`分析任务终态${task.status}；不计为空召回或错误回答`);
        break;
      }
      if (!['pending', 'running'].includes(task.status)) throw new Error('任务状态异常或意外要求审批；停止且不重发');
      if (pollIntervalMs) await pause(Math.min(pollIntervalMs, Math.max(0, taskTimeoutMs - (now() - started))));
    }
    const analysis = parseAnalysis(task), answerScreen = screenAnswer(item, analysis, kbID, known, bindings);
    calls.completed_analyses++; calls.known_tool_search_calls++; if (analysis.status !== 'no_matches') calls.known_chat_calls++;
    const result = { type: 'result', case_id: item.id, split: item.split, category: item.category, kb_id: kbID, query: item.query,
      request_id: requestID, task_id: task.id, search_client_ms: searchClient, search_server_ms: response.elapsed_ms, analysis_elapsed_ms: analysisElapsed,
      list, retrieval, analysis, answer_screen: answerScreen };
    results.push(result); await onProgress(result);
  }
  const finalKnown = await listDocuments(kbMap, kbKeys, client);
  if (JSON.stringify(inventoryOf(finalKnown)) !== JSON.stringify(inventory)) throw new Error('运行期间知识库文档库存、版本或状态变化；本轮不可比较');
  for (const binding of bindings.values()) {
    const latest = await readBoundDocument(binding, finalKnown.get(binding.kb).get(binding.doc_id), client);
    if (JSON.stringify(latest) !== JSON.stringify(binding)) throw new Error('运行期间语料版本变化；本轮不可比较');
  }
  await onProgress({ type: 'corpus_verified', verified_at: new Date().toISOString(), document_count: bindings.size });
  calls.http = { ...(client.counts ?? {}) };
  calls.note = 'search_requests与analysis_submissions为实际POST次数；known_tool_search_calls/known_chat_calls只计完整成功分析；每个正常题最多2次查询Embedding，故障时具体供应商调用数未知';
  return { results, bindings: [...bindings.values()], summary: summarize(results, calls) };
}

async function assertOutputPath(filename) {
  const parent = await realpath(dirname(filename)), tmpRoot = await realpath(resolve(repositoryRoot, 'tmp'));
  if (parent !== tmpRoot && !within(tmpRoot, parent)) throw new Error('输出父目录符号链接越出tmp');
}
async function emitMarkdown(markdown, output) {
  if (!output) { console.log(markdown); return; }
  await assertOutputPath(output);
  try { await writeFile(output, markdown, { flag: 'wx', mode: 0o600 }); } catch (error) { throw new Error(error.code === 'EEXIST' ? '报告已存在，拒绝覆盖' : '无法创建Markdown报告'); }
  console.log(`已写入报告 ${relative(repositoryRoot, output)}`);
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.rescore) {
    const sourceBytes = await readFile(options.rescore), records = parseRecords(new TextDecoder('utf-8', { fatal: true }).decode(sourceBytes));
    const validation = { dataset: await loadDataset(options.cases), datasetSha256: hash(await readFile(options.cases)) };
    validateContext(records.find(row => row.type === 'meta')?.context);
    const rescored = rescoreRecords(records, { ...validation, sourceSha256: hash(sourceBytes) });
    await assertOutputPath(options.output); const output = await openReport(options.output);
    try { for (const row of rescored) await output.append(row); } finally { await output.close(); }
    console.log(JSON.stringify({ mode: 'offline_rescore', http_requests: 0, cases: rescored.filter(row => row.type === 'result').length,
      source_sha256: hash(sourceBytes), output: relative(repositoryRoot, options.output) })); return;
  }
  if (options.report) {
    const validation = { dataset: await loadDataset(options.cases), datasetSha256: hash(await readFile(options.cases)) };
    await emitMarkdown(renderReport(await readRecords(options.report), validation), options.output); return;
  }
  if (options.compare) {
    const validation = { dataset: await loadDataset(options.cases), datasetSha256: hash(await readFile(options.cases)) };
    const comparison = compareRuns(await readRecords(options.compare[0]), await readRecords(options.compare[1]), validation);
    await emitMarkdown(renderComparison(comparison), options.output); if (!comparison.comparable) process.exitCode = 1; return;
  }
  const dataset = await loadDataset(options.cases);
  if (options.caseIDs?.some(id => !dataset.cases.some(item => item.id === id))) throw new Error('case-ids包含未知题目');
  dataset.cases = dataset.cases.filter(item => (options.split === 'all' || item.split === options.split) && (!options.caseIDs || options.caseIDs.includes(item.id)));
  if (!dataset.cases.length || options.caseIDs?.some(id => !dataset.cases.some(item => item.id === id))) throw new Error('所选split与case-ids不匹配或没有题目');
  if (!options.run) {
    console.log(JSON.stringify({ mode: 'plan', http_requests: 0, phase: options.phase, split: options.split, case_count: dataset.cases.length,
      by_split: Object.fromEntries(['development', 'validation'].map(key => [key, dataset.cases.filter(item => item.split === key).length])), case_ids: dataset.cases.map(item => item.id),
      max_analysis_submissions: dataset.cases.length, max_query_embedding_calls: dataset.cases.length * 2,
      documents: dataset.documents, usage: '--run --auth-file <login.result JSON> --kb-map <JSON> --context-file <sanitised JSON> --output tmp/<new>.jsonl；--report <JSONL>；--compare <baseline JSONL> <candidate JSONL>' }, null, 2)); return;
  }
  const kbMap = await readJSON(options.kbMapFile, 'kb-map'), context = validateContext(await readJSON(options.contextFile, 'context')), auth = await readJSON(options.authFile, '认证文件');
  if (auth.expires_at !== undefined && (!positiveID(auth.expires_at) || auth.expires_at * 1000 <= Date.now() + 1000)) throw new Error('认证已过期；请正常登录后显式执行，不自动刷新');
  const client = createApiClient({ token: auth.access_token, base: options.base });
  await assertOutputPath(options.output); const report = await openReport(options.output); let activeCase = null;
  try {
    await report.append({ type: 'meta', version: 1, phase: options.phase, split: options.split, started_at: new Date().toISOString(), base_url: options.base,
      dataset_sha256: hash(await readFile(options.cases)), case_ids: dataset.cases.map(item => item.id), kb_map: kbMap, context });
    const result = await runEvaluation({ dataset, kbMap, client, onProgress: async event => {
      if (event.type === 'case_started') activeCase = event.case_id;
      await report.append(event);
      if (event.type === 'result') console.log(`case=${event.case_id} split=${event.split} task_id=${event.task_id} rule_flags=${event.answer_screen.reasons.join(',') || 'none'} analysis_ms=${Math.round(event.analysis_elapsed_ms)}`);
    } });
    await report.append({ type: 'summary', completed: true, completed_at: new Date().toISOString(), ...result.summary });
    console.log(JSON.stringify({ completed: true, n: result.results.length, flagged: result.summary.answer_rule_screen.flagged_count, calls: result.summary.calls, usage: result.summary.usage }));
  } catch (error) {
    await report.append({ type: 'error', case_id: activeCase, stopped_at: new Date().toISOString(), message: error.message, retry: false, comparable: false, calls: { ...client.counts } }); throw error;
  } finally { await report.close(); }
}

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) main().catch(error => { console.error(error.message); process.exitCode = 1; });
