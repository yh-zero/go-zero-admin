import { readFile } from 'node:fs/promises';
import { summarize, evaluateRetrieval, screenAnswer, validateAnalysis } from './eval-kb-quality-metrics.mjs';
import { validateHits } from './eval-kb-recall.mjs';
import { validateContext } from './eval-kb-quality-context.mjs';

export function parseRecords(source) {
  let rows;
  try { rows = source.split('\n').filter(line => line.trim()).map(JSON.parse); }
  catch { throw new Error('JSONL不可读或包含损坏行；不能把截断报告标为完成'); }
  if (!rows.length || rows.some(row => !row || typeof row !== 'object' || typeof row.type !== 'string')) throw new Error('JSONL记录格式无效');
  return rows;
}
export async function readRecords(filename) {
  let source; try { source = await readFile(filename, 'utf8'); } catch { throw new Error('JSONL不可读'); }
  return parseRecords(source);
}

const stable = value => JSON.stringify(canonical(value));
function canonical(value) {
  if (Array.isArray(value)) return value.map(canonical);
  if (value && typeof value === 'object') return Object.fromEntries(Object.keys(value).sort().map(key => [key, canonical(value[key])]));
  return value;
}
function state(records) {
  const meta = records.find(row => row.type === 'meta'), bindings = records.find(row => row.type === 'bindings');
  const results = records.filter(row => row.type === 'result'), summary = records.findLast(row => row.type === 'summary');
  const ids = results.map(row => row.case_id);
  return { meta, bindings, results, summary,
    complete: records.filter(row => row.type === 'meta').length === 1 && records.filter(row => row.type === 'bindings').length === 1
      && !!meta?.context && Array.isArray(meta.case_ids) && meta.case_ids.length > 0
      && !records.some(row => row.type === 'error') && !!records.find(row => row.type === 'corpus_verified')
      && summary?.completed === true && ids.length === meta.case_ids.length && new Set(ids).size === ids.length
      && stable([...ids].sort()) === stable([...meta.case_ids].sort()) };
}
const fixedContext = context => Object.fromEntries(Object.entries(context ?? {}).filter(([key]) => !['binary_sha', 'source_revision', 'experiment'].includes(key)));
const corpus = bindings => (bindings?.documents ?? []).map(row => ({ key: row.key, doc_id: row.doc_id, kb_id: row.kb_id,
  version: row.version, content_sha256: row.content_sha256, embedding_model: row.embedding_model, title: row.title, status: row.status })).sort((a, b) => a.key.localeCompare(b.key));

function inspectRun(records, { dataset, datasetSha256 } = {}, { checkDerived = true } = {}) {
  const current = state(records), reasons = [];
  const require = (condition, message) => { if (!condition) throw new Error(message); };
  const same = (left, right) => stable(left) === stable(right);
  const positive = value => Number.isSafeInteger(value) && value > 0;
  try {
    require(current.complete, '运行失败或不完整，缺少全部结果/结束语料复核/完成标记');
    require(current.meta.version === 1, 'JSONL版本不受当前评测器支持');
    validateContext(current.meta.context);
    require(dataset && datasetSha256 && current.meta.dataset_sha256 === datasetSha256, '需要同SHA256的冻结题集复算，当前题集摘要不匹配或未提供');
    require(records[0] === current.meta && records.at(-1) === current.summary && records.filter(row => row.type === 'summary').length === 1
      && records.filter(row => row.type === 'corpus_verified').length === 1, '运行记录顺序、结束标记或summary数量无效');
    const ids = current.meta.case_ids, cases = new Map(dataset.cases.map(item => [item.id, item]));
    require(ids.every(id => typeof id === 'string' && cases.has(id)) && new Set(ids).size === ids.length, '所选题ID无效、重复或不在冻结题集中');
    const kbKeys = [...new Set(ids.map(id => cases.get(id).kb))], kbMap = current.meta.kb_map;
    require(kbMap && kbKeys.every(key => positive(kbMap[key])) && new Set(kbKeys.map(key => kbMap[key])).size === kbKeys.length, '知识库映射缺失或不一致');
    const boundRows = current.bindings.documents, inventory = current.bindings.inventory;
    require(Array.isArray(boundRows) && boundRows.length > 0 && Array.isArray(inventory), '缺少实际语料绑定或知识库库存');
    const bindings = new Map(boundRows.map(row => [row.key, row]));
    const expectedSources = dataset.documents.filter(row => kbKeys.includes(row.kb));
    require(bindings.size === boundRows.length && bindings.size === expectedSources.length, '绑定文档缺失或重复');
    for (const source of expectedSources) {
      const bound = bindings.get(source.key);
      require(bound && positive(bound.doc_id) && bound.kb_id === kbMap[source.kb] && bound.kb === source.kb && positive(bound.version) && bound.status === 'ready'
        && bound.title === source.title && bound.source === source.source && bound.sha256 === source.sha256 && bound.content_sha256 === source.sha256 && typeof bound.embedding_model === 'string' && bound.embedding_model,
      '实际语料绑定与冻结题集不一致');
      require(inventory.some(doc => doc.id === bound.doc_id && doc.kb_id === bound.kb_id && doc.kb === bound.kb && doc.title === bound.title && doc.version === bound.version && doc.status === bound.status && doc.embedding_model === bound.embedding_model), '绑定文档缺少对应库存版本');
    }
    require(new Set(inventory.map(doc => doc.id)).size === inventory.length && inventory.every(doc => positive(doc.id) && positive(doc.version) && doc.kb_id === kbMap[doc.kb]
      && typeof doc.title === 'string' && ['ready', 'pending', 'failed'].includes(doc.status) && typeof doc.embedding_model === 'string'), '知识库库存格式或归属无效');
    const caseTypes = ['case_started', 'search_result', 'analysis_requested', 'analysis_accepted', 'analysis_terminal', 'result'];
    const expectedEvents = [{ type: 'meta' }, { type: 'bindings' }, ...ids.flatMap(case_id => caseTypes.map(type => ({ type, case_id }))),
      { type: 'corpus_verified' }, { type: 'summary' }];
    require(records.length === expectedEvents.length && records.every((row, index) => row.type === expectedEvents[index].type && row.case_id === expectedEvents[index].case_id),
      'v1事件必须按meta、bindings、case_ids逐题串行、corpus_verified、summary排序，不能交错或包含未知事件');
    require(records.filter(row => caseTypes.includes(row.type)).every(row => ids.includes(row.case_id)), '记录包含所选范围以外题目');
    const requests = new Set(), taskIDs = new Set(), results = [];
    for (const row of current.results) {
      const item = cases.get(row.case_id), kbID = kbMap[item.kb];
      require(row.query === item.query && row.split === item.split && row.category === item.category && row.kb_id === kbID, '题目query、split、category或知识库与冻结题集不一致');
      require(['search_client_ms', 'search_server_ms', 'analysis_elapsed_ms'].every(key => typeof row[key] === 'number' && Number.isFinite(row[key]) && row[key] >= 0), '逐题耗时缺失或无效');
      require(Number.isSafeInteger(row.search_server_ms) && row.analysis_elapsed_ms <= 100_000, '检索服务端耗时或分析100秒边界无效');
      require(typeof row.request_id === 'string' && /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/iu.test(row.request_id)
        && !requests.has(row.request_id) && positive(row.task_id) && !taskIDs.has(row.task_id), '逐题原UUID或任务ID缺失/重复');
      requests.add(row.request_id); taskIDs.add(row.task_id);
      const known = new Map(inventory.filter(doc => doc.kb_id === kbID).map(doc => [doc.id, doc]));
      validateHits(row.list, kbID, known); validateAnalysis(row.analysis);
      const retrieval = evaluateRetrieval(item, row.list, bindings), answerScreen = screenAnswer(item, row.analysis, kbID, known, bindings);
      if (checkDerived) require(same(row.retrieval, retrieval) && same(row.answer_screen, answerScreen), '保存的派生指标与当前冻结规则重算不一致；可能规则版本不同或记录有修改，需核对');
      const chain = caseTypes.map(type => records.filter(event => event.type === type && event.case_id === row.case_id));
      require(chain.every(events => events.length === 1), '逐题请求、接受、终态或检索记录缺失/重复');
      const [started, search, requested, accepted, terminal] = chain.map(events => events[0]);
      const indices = chain.map(events => records.indexOf(events[0]));
      require(indices.every((index, offset) => offset === 0 ? index > records.indexOf(current.bindings) : index > indices[offset - 1]), '逐题请求与终态记录顺序不一致');
      require(started.kb_id === kbID && started.split === item.split && started.category === item.category
        && search.kb_id === kbID && search.query === item.query && same(search.list, row.list) && (!checkDerived || same(search.retrieval, retrieval))
        && search.search_client_ms === row.search_client_ms && search.search_server_ms === row.search_server_ms, '原始检索记录与逐题结果不一致');
      const payload = requested.payload;
      require(requested.request_id === row.request_id && payload?.request_id === row.request_id && typeof payload.name === 'string' && payload.name.startsWith('eval-kb-quality-')
        && payload.input === '{}' && same(JSON.parse(payload.definition), { steps: [{ name: '知识库分析', tool: 'kb_analyze', input: { kb_id: kbID, query: item.query } }] }), '原UUID、载荷与冻结查询不一致');
      require(['pending', 'running', 'succeeded'].includes(accepted.status), '已接受任务状态缺失或不允许进入成功链');
      require(accepted.request_id === row.request_id && accepted.task_id === row.task_id && terminal.request_id === row.request_id && terminal.task_id === row.task_id && terminal.status === 'succeeded', '原UUID、已接受任务或终态不一致');
      results.push({ ...row, retrieval, answer_screen: answerScreen });
    }
    const verified = records.find(row => row.type === 'corpus_verified');
    require(verified.document_count === bindings.size && records.indexOf(verified) > Math.max(...current.results.map(row => records.indexOf(row))), '结束语料复核缺失或发生在逐题结果前');
    const calls = current.summary.calls, n = results.length, chatCalls = results.filter(row => row.analysis.status !== 'no_matches').length;
    require(calls && ['search_requests', 'analysis_submissions', 'accepted_tasks', 'completed_analyses', 'known_tool_search_calls'].every(key => calls[key] === n)
      && calls.known_chat_calls === chatCalls && Number.isSafeInteger(calls.task_reads) && calls.task_reads >= n, '调用计数与成功分析分母不一致');
    const http = calls.http;
    require(http && http.search === n && http.analysis_submissions === n && http.post === 2 * n && http.task_reads === calls.task_reads && http.failed === 0
      && Number.isSafeInteger(http.get) && http.get >= calls.task_reads + 2 * bindings.size + 2 * kbKeys.length && http.total === http.get + http.post, 'HTTP计数与已保存调用链不一致');
    const summary = summarize(results, calls);
    if (checkDerived) require(Object.keys(summary).every(key => same(summary[key], current.summary[key])), '保存的summary与逐题原始数据重算不一致；需核对评测器版本或记录');
    return { ...current, reasons, results, recomputed_summary: summary };
  } catch (error) { reasons.push(error instanceof SyntaxError ? '原始请求definition不是有效JSON' : error.message); }
  return { ...current, reasons, complete: false };
}

export function rescoreRecords(records, { sourceSha256, at = new Date().toISOString(), ...validation } = {}) {
  if (!/^[a-f0-9]{64}$/u.test(sourceSha256 ?? '') || typeof at !== 'string' || !Number.isFinite(Date.parse(at))) throw new Error('离线复算需要有效源SHA256和时间');
  // 只豁免旧派生指标，UUID、原始检索、来源、事件、绑定、题集与调用计数仍完整校验。
  const source = inspectRun(records, validation, { checkDerived: false });
  if (!source.complete) throw new Error(`离线复算拒绝原始记录：${source.reasons.join('；')}`);
  const rescored = structuredClone(records), results = new Map(source.results.map(row => [row.case_id, row]));
  for (const row of rescored) {
    if (row.type === 'meta') {
      const previous = row.derived_from;
      row.derived_from = { source_sha256: sourceSha256, at, operation: 'offline_rescore' };
      if (previous) row.derived_from.previous = previous;
    } else if (row.type === 'search_result') row.retrieval = structuredClone(results.get(row.case_id).retrieval);
    else if (row.type === 'result') {
      row.retrieval = structuredClone(results.get(row.case_id).retrieval);
      row.answer_screen = structuredClone(results.get(row.case_id).answer_screen);
    } else if (row.type === 'summary') Object.assign(row, structuredClone(source.recomputed_summary));
  }
  const verified = inspectRun(rescored, validation);
  if (!verified.complete) throw new Error(`离线复算副本校验失败：${verified.reasons.join('；')}`);
  return rescored;
}

export function compareRuns(baselineRecords, candidateRecords, validation) {
  const baseline = inspectRun(baselineRecords, validation), candidate = inspectRun(candidateRecords, validation), reasons = [];
  reasons.push(...baseline.reasons.map(reason => `baseline：${reason}`), ...candidate.reasons.map(reason => `candidate：${reason}`));
  const allowedChanges = ['binary_sha', 'source_revision', 'experiment'].filter(key => stable(baseline.meta?.context?.[key]) !== stable(candidate.meta?.context?.[key]))
    .map(key => ({ field: key, baseline: baseline.meta?.context?.[key], candidate: candidate.meta?.context?.[key] }));
  if (reasons.length) return { comparable: false, reasons, allowed_changes: allowedChanges };
  if (baseline.meta?.phase !== 'baseline' || candidate.meta?.phase !== 'candidate') reasons.push('必须显式提供baseline与candidate阶段');
  if (baseline.meta?.dataset_sha256 !== candidate.meta?.dataset_sha256 || stable([...(baseline.meta?.case_ids ?? [])].sort()) !== stable([...(candidate.meta?.case_ids ?? [])].sort())) reasons.push('题集摘要或所选题目不同');
  if (stable(corpus(baseline.bindings)) !== stable(corpus(candidate.bindings)) || stable(baseline.bindings?.inventory) !== stable(candidate.bindings?.inventory)) reasons.push('文档ID、语料哈希、版本、Embedding或知识库库存不同');
  if (stable(fixedContext(baseline.meta?.context)) !== stable(fixedContext(candidate.meta?.context))) reasons.push('聊天/Embedding模型、预算、TopK或其余运行上下文不同');
  if (reasons.length) return { comparable: false, reasons, allowed_changes: allowedChanges };
  const before = baseline.recomputed_summary, after = candidate.recomputed_summary;
  const delta = (a, b) => typeof a === 'number' && typeof b === 'number' ? b - a : null;
  const groupDelta = (a, b) => ({ n: b.n, hit_at_5: delta(a.positive.hit_at[5], b.positive.hit_at[5]),
    recall_at_5: delta(a.positive.recall_at[5], b.positive.recall_at[5]), mrr_at_5: delta(a.positive.mrr_at[5], b.positive.mrr_at[5]),
    evidence_coverage_at_5: delta(a.positive.evidence_coverage_at[5], b.positive.evidence_coverage_at[5]),
    rule_flagged_count: b.answer_rule_screen.flagged_count - a.answer_rule_screen.flagged_count,
    analysis_p50_ms: delta(a.latency.analysis_submit_to_terminal?.p50_ms, b.latency.analysis_submit_to_terminal?.p50_ms),
    analysis_p95_ms: delta(a.latency.analysis_submit_to_terminal?.p95_ms, b.latency.analysis_submit_to_terminal?.p95_ms) });
  return { comparable: true, reasons: [], allowed_changes: allowedChanges, baseline: before, candidate: after,
    delta: groupDelta(before, after), by_split: Object.fromEntries(Object.keys(before.by_split).map(key => [key, groupDelta(before.by_split[key], after.by_split[key])])),
    by_category: Object.fromEntries(Object.keys(before.by_category).map(key => [key, groupDelta(before.by_category[key], after.by_category[key])])),
    cases: baseline.results.map(row => { const other = candidate.results.find(item => item.case_id === row.case_id);
      return { case_id: row.case_id, baseline_reasons: row.answer_screen.reasons, candidate_reasons: other.answer_screen.reasons,
        baseline_recall_at_5: row.retrieval.recall_at?.[5] ?? null, candidate_recall_at_5: other.retrieval.recall_at?.[5] ?? null }; }) };
}

// 所有模型文本以缩进代码块呈现，避免把资料中的Markdown/HTML作为报告指令或远程媒体执行。
const codeBlock = value => String(value).split('\n').map(line => `    ${line}`).join('\n');
const decimal = value => value === null || value === undefined ? '—' : typeof value === 'number' ? String(Number(value.toFixed(4))) : String(value);
function groupTable(summary) {
  const rows = [['全部成功题', summary], ...Object.entries(summary.by_split).map(([key, value]) => [`split:${key}`, value]), ...Object.entries(summary.by_category).map(([key, value]) => [`category:${key}`, value])];
  return ['| 范围 | N(正/负) | Hit@1/3/5 | Recall@1/3/5 | MRR@1/3/5 | 证据覆盖@1/3/5 | 规则标记 |', '|---|---:|---|---|---|---|---:|',
    ...rows.map(([name, group]) => `| ${name} | ${group.n}(${group.positive_count}/${group.negative_count}) | ${[1,3,5].map(k => decimal(group.positive.hit_at[k])).join('/')} | ${[1,3,5].map(k => decimal(group.positive.recall_at[k])).join('/')} | ${[1,3,5].map(k => decimal(group.positive.mrr_at[k])).join('/')} | ${[1,3,5].map(k => decimal(group.positive.evidence_coverage_at[k])).join('/')} | ${group.answer_rule_screen.flagged_count} |`) ].join('\n');
}
export function renderReport(records, validation) {
  const current = validation ? inspectRun(records, validation) : state(records), errors = records.filter(row => row.type === 'error');
  const summary = current.recomputed_summary ?? summarize(current.results, current.summary?.calls ?? records.findLast(row => row.calls)?.calls ?? {});
  const lines = ['# 知识库质量评测', '', `状态：${current.complete ? '完整；结束语料已复核' : '失败或不完整；不可比较'}。成功完成 ${current.results.length}/${current.meta?.case_ids?.length ?? '?'} 题。`, '',
    '回答指标仅为规则筛查，全部答案需人工复核；不代表语义准确率或人工评审通过率。失败题不记为空召回或错误回答。', '',
    '## 运行上下文', '', codeBlock(JSON.stringify(current.meta ?? {}, null, 2)), '', '## 语料绑定', '', codeBlock(JSON.stringify(current.bindings ?? {}, null, 2)), '',
    '## 检索与规则筛查', '', groupTable(summary), '', '## 耗时与调用', '', '以下为串行样本经验P50/P95（nearest-rank），N为成功题样本量；不代表生产并发或容量。', '',
    codeBlock(JSON.stringify({ latency: summary.latency, by_split: Object.fromEntries(Object.entries(summary.by_split).map(([key, group]) => [key, group.latency])),
      by_category: Object.fromEntries(Object.entries(summary.by_category).map(([key, group]) => [key, group.latency])), calls: summary.calls, usage: summary.usage }, null, 2)), '', '## 逐题人工复核', ''];
  if (current.meta?.derived_from?.operation === 'offline_rescore') lines.splice(4, 0, '本报告为原始运行日志的离线规则重算副本；没有重新请求服务或模型，也没有重新核对当前语料。来源、耗时和调用计数保留原运行记录。', '');
  for (const row of current.results) {
    lines.push(`### ${row.case_id}`, '', `split=${row.split}；category=${row.category}；规则${row.answer_screen.flagged ? '标记' : '未标记'}，仍需人工复核。`, '',
      codeBlock(JSON.stringify({ query: row.query, retrieval: row.retrieval, answer_screen: row.answer_screen, task_id: row.task_id,
        request_id: row.request_id, analysis: row.analysis }, null, 2)), '');
  }
  if (errors.length) lines.push('## 错误与未覆盖', '', codeBlock(JSON.stringify(errors, null, 2)), '');
  if (current.reasons?.length) lines.push('## 记录核对', '', codeBlock(JSON.stringify(current.reasons, null, 2)), '');
  const finished = new Set(current.results.map(row => row.case_id));
  const incomplete = records.filter(row => ['analysis_requested', 'analysis_accepted', 'search_result'].includes(row.type) && !finished.has(row.case_id));
  if (incomplete.length) lines.push('## 中止前已保存记录', '', '保留原UUID、载荷、任务ID和检索结果；未知请求不得产生新UUID自动重发。', '', codeBlock(JSON.stringify(incomplete, null, 2)), '');
  lines.push('## 测量边界', '', ...summary.notes.map(note => `- ${note}`), '');
  return lines.join('\n');
}

export function renderComparison(comparison) {
  return ['# 知识库评测比较', '', comparison.comparable ? '同题、同语料与同模型上下文核对通过；差值为candidate减baseline。' : '不可比较，不计算质量或耗时提升。', '',
    '仅为规则筛查，所有答案仍需人工复核；串行经验分位数不代表生产容量。', '', codeBlock(JSON.stringify(comparison, null, 2)), ''].join('\n');
}
