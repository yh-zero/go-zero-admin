import { createHash } from 'node:crypto';
import { readFile, open, realpath } from 'node:fs/promises';
import { dirname, resolve, relative, isAbsolute, sep } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { performance } from 'node:perf_hooks';

// 默认仅展示计划。--run 才会通过正式API调用Embedding，不执行聊天或自动重试。
// 用法（从仓库根目录执行，auth文件内容是正常登录的result）：
// node go-zero-admin/data/scripts/eval-kb-recall.mjs --plan
// node go-zero-admin/data/scripts/eval-kb-recall.mjs --run --auth-file tmp/auth-private.json \
//   --manifest tmp/kb-experience-20261106/manifest.json --output tmp/recall-20261107.jsonl
// 也可使用 --kb-map tmp/kb-map.json，内容为 {"stocks":1,"ops":2}；无导入manifest时，
// 只能按正式列表中的唯一标题绑定ID，报告会注明不能验证服务端正文哈希。
// --case-ids id1,id2 显式选题；失败后检查旧JSONL，再用新输出文件显式选择尚未运行题。
const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const defaultCases = resolve(repositoryRoot, 'ai_document/ai-rag/recall-cases.json');
const ks = [1, 3, 5];
const positiveID = value => Number.isSafeInteger(value) && value > 0;
const nonnegative = value => Number.isSafeInteger(value) && value >= 0;
const normalized = text => text.normalize('NFKC').replace(/\s+/gu, '');
const rounded = number => Number(number.toFixed(3));

export async function loadDataset(filename = defaultCases) {
  let dataset;
  try { dataset = JSON.parse(await readFile(filename, 'utf8')); }
  catch { throw new Error('题集文件不可读或不是有效JSON'); }
  if (dataset.version !== 1 || !Array.isArray(dataset.documents) || !Array.isArray(dataset.cases)
      || dataset.cases.length === 0 || dataset.cases.length > 100) throw new Error('题集结构或数量无效');
  const documents = new Map();
  const sourceRoot = await realpath(resolve(repositoryRoot, 'ai_document/ai-rag'));
  for (const document of dataset.documents) {
    if (!document || typeof document.key !== 'string' || documents.has(document.key)
        || typeof document.kb !== 'string' || !document.kb || typeof document.title !== 'string' || !document.title.trim()
        || !/^[a-f0-9]{64}$/.test(document.sha256) || typeof document.source !== 'string'
        || isAbsolute(document.source)) throw new Error('题集素材字段无效');
    let bytes;
    try {
      const sourcePath = await realpath(resolve(repositoryRoot, document.source));
      const pathWithin = relative(sourceRoot, sourcePath);
      if (!pathWithin || isAbsolute(pathWithin) || pathWithin.startsWith('..') || pathWithin.includes(`..${sep}`)) throw new Error('outside');
      bytes = await readFile(sourcePath);
    } catch { throw new Error(`素材路径不可读或越界：${document.key}`); }
    if (createHash('sha256').update(bytes).digest('hex') !== document.sha256) throw new Error(`素材SHA256发生变化：${document.key}`);
    documents.set(document.key, { ...document, fullText: new TextDecoder('utf-8', { fatal: true }).decode(bytes) });
  }
  const caseIDs = new Set();
  for (const item of dataset.cases) {
    if (!item || typeof item.id !== 'string' || !item.id || caseIDs.has(item.id) || typeof item.kb !== 'string'
        || typeof item.query !== 'string' || !item.query.trim() || Buffer.byteLength(item.query) > 2000
        || !Array.isArray(item.evidence_groups)) throw new Error('题目字段、ID或query字节数无效');
    caseIDs.add(item.id);
    if (item.expected_document === null) {
      if (item.evidence_groups.length !== 0) throw new Error(`无答案题不能标注证据：${item.id}`);
      continue;
    }
    const document = documents.get(item.expected_document);
    if (!document || document.kb !== item.kb || item.evidence_groups.length === 0) throw new Error(`预期文档或证据缺失：${item.id}`);
    for (const group of item.evidence_groups) {
      if (!Array.isArray(group) || !group.length || group.some(value => typeof value !== 'string' || !value.trim())
          || !group.some(value => normalized(document.fullText).includes(normalized(value)))) throw new Error(`证据组不在标注原文中：${item.id}`);
    }
  }
  return dataset;
}

export function parseArgs(args) {
  const options = { run: false, base: 'http://127.0.0.1:7001', cases: defaultCases };
  const values = { '--auth-file': 'authFile', '--manifest': 'manifestFile', '--kb-map': 'kbMapFile', '--output': 'output', '--base': 'base', '--cases': 'cases', '--case-ids': 'caseIDs' };
  const seen = new Set();
  for (let index = 0; index < args.length; index++) {
    const key = args[index];
    if (seen.has(key)) throw new Error('命令参数重复');
    seen.add(key);
    if (key === '--run') options.run = true;
    else if (key !== '--plan') {
      if (!values[key] || !args[index + 1] || args[index + 1].startsWith('--')) throw new Error('未知参数或参数缺值');
      options[values[key]] = args[++index];
    }
  }
  if (seen.has('--plan') && options.run) throw new Error('--plan与--run不能同时使用');
  let base;
  try { base = new URL(options.base); } catch { throw new Error('base不是有效的本机HTTP服务地址'); }
  if (base.protocol !== 'http:' || !['127.0.0.1', 'localhost', '[::1]'].includes(base.hostname)
      || base.username || base.password || base.pathname !== '/' || base.search || base.hash) throw new Error('base仅允许本机HTTP服务根地址');
  options.base = base.origin;
  if (options.run && (!options.authFile || !options.output || (!options.manifestFile && !options.kbMapFile))) throw new Error('--run需要认证auth-file、manifest或kb-map，以及新的output路径');
  if (options.output) {
    options.output = resolve(options.output);
    const location = relative(resolve(repositoryRoot, 'tmp'), options.output);
    if (!location || isAbsolute(location) || location.startsWith('..') || resolve(repositoryRoot, 'tmp', location) !== options.output) throw new Error('结果output必须位于仓库被忽略的tmp目录');
  }
  if (options.caseIDs) {
    options.caseIDs = options.caseIDs.split(',');
    if (options.caseIDs.some(id => !id) || new Set(options.caseIDs).size !== options.caseIDs.length) throw new Error('case-ids重复或为空');
  }
  return options;
}

export function createApiClient({ token, base = 'http://127.0.0.1:7001', fetchImpl = fetch, timeoutMs = 12_000 }) {
  if (typeof token !== 'string' || !token.trim()) throw new Error('认证令牌缺失');
  return async (path, method = 'GET', body) => {
    let response;
    try {
      response = await fetchImpl(base + '/v1/sys/ai' + path, { method, headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(timeoutMs), redirect: 'error' });
    } catch (error) {
      if (['TimeoutError', 'AbortError'].includes(error?.name)) throw new Error('请求超时或取消，结果未知；未自动重试');
      throw new Error('网络请求失败，结果未知；未自动重试');
    }
    if (response.status !== 200) throw new Error(`HTTP ${response.status}；未自动重试`);
    let envelope;
    try { envelope = await response.json(); } catch { throw new Error('服务端未返回有效JSON'); }
    if (!envelope || envelope.code !== 200 || !envelope.result || typeof envelope.result !== 'object' || Array.isArray(envelope.result)) {
      throw new Error(`业务响应失败 code=${Number.isInteger(envelope?.code) ? envelope.code : 'invalid'}；不能计为空召回`);
    }
    return envelope.result;
  };
}

async function bindDocuments(dataset, kbMap, manifest, client) {
  if (manifest && !Array.isArray(manifest.documents)) throw new Error('导入manifest缺少素材文档数组');
  const kbKeys = [...new Set(dataset.cases.map(item => item.kb))];
  if (kbKeys.some(key => !positiveID(kbMap[key])) || new Set(kbKeys.map(key => kbMap[key])).size !== kbKeys.length) throw new Error('知识库映射ID缺失、无效或重复');
  const knownByKb = new Map();
  for (const key of kbKeys) {
    const kbID = kbMap[key], known = new Map();
    for (let page = 1; page <= 1000; page++) {
      const data = await client(`/kbs/${kbID}/docs?page=${page}&page_size=100`);
      if (!nonnegative(data.total) || !Array.isArray(data.list)) throw new Error('文档列表分页格式无效');
      for (const document of data.list) {
        if (!positiveID(document.id) || document.kb_id !== kbID || known.has(document.id)
            || typeof document.title !== 'string' || !document.title.trim()
            || !['ready', 'pending', 'failed'].includes(document.status)) throw new Error('文档列表归属或ID字段无效');
        known.set(document.id, document);
      }
      if (known.size === data.total) break;
      if (known.size > data.total || !data.list.length || page === 1000) throw new Error('文档分页数量不一致或超出上限');
    }
    knownByKb.set(key, known);
  }
  const bindings = new Map();
  const referenced = new Set(dataset.cases.map(item => item.expected_document).filter(Boolean));
  for (const source of dataset.documents.filter(document => referenced.has(document.key))) {
    const kbID = kbMap[source.kb], known = knownByKb.get(source.kb);
    let document;
    if (manifest) {
      const imported = manifest.documents?.filter(item => item.source === source.source) ?? [];
      if (imported.length !== 1 || imported[0].sha256 !== source.sha256 || imported[0].title !== source.title
          || imported[0].kb_id !== kbID || !positiveID(imported[0].id)) throw new Error(`导入manifest来源/哈希/知识库不一致：${source.key}`);
      document = known.get(imported[0].id);
    } else {
      const matches = [...known.values()].filter(item => item.title === source.title);
      if (matches.length !== 1) throw new Error(`标题不唯一或缺失，不能绑定同名文档：${source.key}`);
      [document] = matches;
    }
    if (!document || document.title !== source.title || document.status !== 'ready') throw new Error(`预期文档不存在或未就绪：${source.key}`);
    bindings.set(source.key, { ...source, doc_id: document.id, kb_id: kbID });
  }
  return { bindings, knownByKb };
}

export function validateHits(list, kbID, knownDocuments) {
  if (!Array.isArray(list) || list.length > 5) throw new Error('召回片段数量无效');
  const chunks = new Set();
  let previous;
  for (const item of list) {
    if (!item || !positiveID(item.chunk_id) || !positiveID(item.doc_id) || chunks.has(item.chunk_id)) throw new Error('召回分块ID无效或重复');
    if (item.kb_id !== kbID) throw new Error('召回知识库越界');
    const document = knownDocuments.get(item.doc_id);
    if (!document || document.status !== 'ready' || item.doc_title !== document.title) throw new Error('召回文档ID、标题或就绪状态无效');
    if (typeof item.snippet !== 'string' || !item.snippet.trim() || typeof item.score !== 'number' || !Number.isFinite(item.score)) throw new Error('召回片段或分数字段无效');
    if (previous && (item.score > previous.score || (item.score === previous.score && item.chunk_id < previous.chunk_id))) throw new Error('召回排序或同分分块顺序不稳定');
    chunks.add(item.chunk_id); previous = item;
  }
  return list;
}

export function evaluateCase(item, list, expectedDocID) {
  if (item.expected_document === null) return { kind: 'negative', candidate_count: list.length,
    max_score: list.length ? Math.max(...list.map(hit => hit.score)) : null, scope_isolated: true,
    note: '非空候选不等于有答案；此处不评判生成答案或拒答正确率' };
  const first = list.slice(0, 5).findIndex(hit => hit.doc_id === expectedDocID);
  const result = { kind: 'positive', hit_at: {}, recall_at: {}, evidence_at: {}, mrr_at_5: first < 0 ? 0 : 1 / (first + 1) };
  for (const k of ks) {
    const snippets = list.slice(0, k).filter(hit => hit.doc_id === expectedDocID).map(hit => normalized(hit.snippet));
    const matched = item.evidence_groups.map(group => group.some(value => snippets.some(snippet => snippet.includes(normalized(value)))));
    result.hit_at[k] = first >= 0 && first < k ? 1 : 0;
    result.recall_at[k] = result.hit_at[k];
    result.evidence_at[k] = { covered: matched.filter(Boolean).length, total: matched.length,
      coverage: matched.filter(Boolean).length / matched.length, complete: matched.every(Boolean), matched_groups: matched };
  }
  return result;
}

export function summarize(results) {
  const positives = results.filter(item => item.evaluation.kind === 'positive');
  const negatives = results.filter(item => item.evaluation.kind === 'negative');
  const average = values => values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : null;
  const latency = values => {
    if (!values.length) return null;
    const sorted = [...values].sort((a, b) => a - b), middle = Math.floor(sorted.length / 2);
    return { n: sorted.length, min_ms: sorted[0], median_ms: rounded(sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2), max_ms: sorted.at(-1) };
  };
  return { positive_count: positives.length, negative_count: negatives.length,
    positive: { hit_at: Object.fromEntries(ks.map(k => [k, average(positives.map(item => item.evaluation.hit_at[k]))])),
      recall_at: Object.fromEntries(ks.map(k => [k, average(positives.map(item => item.evaluation.recall_at[k]))])),
      mrr_at_5: average(positives.map(item => item.evaluation.mrr_at_5)),
      evidence_coverage_at: Object.fromEntries(ks.map(k => [k, average(positives.map(item => item.evaluation.evidence_at[k].coverage))])),
      complete_evidence_rate_at: Object.fromEntries(ks.map(k => [k, average(positives.map(item => Number(item.evaluation.evidence_at[k].complete)))])) },
    negative: { candidate_return_rate: average(negatives.map(item => Number(item.evaluation.candidate_count > 0))),
      cases: negatives.map(item => ({ case_id: item.case_id, ...item.evaluation })) },
    latency: { client: latency(results.map(item => item.client_elapsed_ms)), server: latency(results.map(item => item.server_elapsed_ms)) },
    notes: ['每题只有一个相关文档，Hit@K与Recall@K同值；K为原始片段数，不先去重改变排名',
      'MRR@5为前五截断指标；证据覆盖是原文字符串匹配，不等同于语义答案准确率',
      '无答案题单列；相似度不是答案置信度，本小题集不用于直接选择通用阈值',
      '小样本串行调用，不是并发基准；不报告P95；素材或索引变化须另存历史结果'] };
}

export async function runEvaluation({ dataset, kbMap, manifest, client, onProgress = async () => {} }) {
  const { bindings, knownByKb } = await bindDocuments(dataset, kbMap, manifest, client);
  const results = [];
  for (const item of dataset.cases) {
    const kbID = kbMap[item.kb];
    await onProgress({ type: 'case_started', case_id: item.id, kb_id: kbID, query: item.query, at: new Date().toISOString() });
    const started = performance.now();
    const response = await client(`/kbs/${kbID}/search`, 'POST', { query: item.query });
    const clientElapsed = rounded(performance.now() - started);
    if (!nonnegative(response.elapsed_ms)) throw new Error('召回elapsed_ms字段无效');
    const list = validateHits(response.list, kbID, knownByKb.get(item.kb));
    const expected = bindings.get(item.expected_document) ?? null;
    const result = { type: 'result', case_id: item.id, kb_id: kbID, query: item.query, expected_document: expected,
      client_elapsed_ms: clientElapsed, server_elapsed_ms: response.elapsed_ms, list,
      evaluation: evaluateCase(item, list, expected?.doc_id ?? null) };
    results.push(result); await onProgress(result);
  }
  return { results, summary: summarize(results) };
}

export async function openReport(filename) {
  let handle;
  try { handle = await open(filename, 'wx', 0o600); }
  catch (error) { throw new Error(error.code === 'EEXIST' ? '结果文件已存在，拒绝覆盖历史结果' : '无法创建结果文件；先创建父目录'); }
  return { append: async value => { await handle.writeFile(JSON.stringify(value) + '\n'); await handle.sync(); }, close: () => handle.close() };
}

async function readJSON(filename, label) {
  try { return JSON.parse(await readFile(filename, 'utf8')); }
  catch { throw new Error(`${label}不可读或不是有效JSON`); }
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const dataset = await loadDataset(options.cases);
  if (options.caseIDs) {
    if (options.caseIDs.some(id => !dataset.cases.some(item => item.id === id))) throw new Error('case-ids包含未知题目');
    dataset.cases = dataset.cases.filter(item => options.caseIDs.includes(item.id));
  }
  if (!options.run) {
    console.log(JSON.stringify({ mode: 'plan', http_requests: 0, documents: dataset.documents, cases: dataset.cases,
      usage: '--run --auth-file <本地login.result JSON> --manifest <导入manifest JSON> --output tmp/<新的结果文件>.jsonl；也可提供--kb-map' }, null, 2));
    return;
  }
  const manifest = options.manifestFile ? await readJSON(options.manifestFile, '导入manifest') : undefined;
  const kbMap = options.kbMapFile ? await readJSON(options.kbMapFile, 'kb-map') : Object.fromEntries((manifest?.kbs ?? []).map(kb => [kb.key, kb.id]));
  const auth = await readJSON(options.authFile, '认证文件');
  if (auth.expires_at !== undefined && (!positiveID(auth.expires_at) || auth.expires_at * 1000 <= Date.now() + 1000)) throw new Error('认证凭据已过期；请正常登录后再显式执行，不自动刷新');
  const client = createApiClient({ token: auth.access_token, base: options.base });
  // 在任何HTTP前排除旧输出文件；追加JSONL确保中止时先前结果仍可读取。
  const report = await openReport(options.output);
  let activeCase = null;
  try {
    await report.append({ type: 'meta', version: 1, started_at: new Date().toISOString(), base_url: options.base,
      dataset_sha256: createHash('sha256').update(await readFile(options.cases)).digest('hex'), case_ids: dataset.cases.map(item => item.id), kb_map: kbMap,
      binding: manifest ? '导入manifest来源/SHA256/ID + 正式文档列表复核；不直接读取服务端正文' : '仅正式列表唯一标题绑定；服务端正文哈希未验证' });
    const result = await runEvaluation({ dataset, kbMap, manifest, client, onProgress: async item => {
      if (item.type === 'case_started') activeCase = item.case_id;
      await report.append(item);
      if (item.type === 'result') console.log(`case=${item.case_id} kind=${item.evaluation.kind} candidates=${item.list.length} elapsed_ms=${item.server_elapsed_ms}`);
    } });
    await report.append({ type: 'summary', completed_at: new Date().toISOString(), ...result.summary });
    console.log(JSON.stringify(result.summary, null, 2));
  } catch (error) {
    await report.append({ type: 'error', case_id: activeCase, stopped_at: new Date().toISOString(), message: error.message, retry: false });
    throw error;
  } finally { await report.close(); }
}

if (process.argv[1] && pathToFileURL(resolve(process.argv[1])).href === import.meta.url) {
  main().catch(error => { console.error(error.message); process.exitCode = 1; });
}
