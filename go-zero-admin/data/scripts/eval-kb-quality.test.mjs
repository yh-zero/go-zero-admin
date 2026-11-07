import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtemp, mkdir, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const quality = await import('./eval-kb-quality.mjs').catch(error => {
  if (error.code === 'ERR_MODULE_NOT_FOUND') return {};
  throw error;
});
const feature = name => { assert.equal(typeof quality[name], 'function', `缺少质量评测能力 ${name}`); return quality[name]; };
const text = '事实甲。事实乙。';
const sha = createHash('sha256').update(text).digest('hex');
const runtimeContext = () => ({ binary_sha: 'a'.repeat(64), source_revision: 'aaa', model: { provider: 'fixture', name: 'fixture' },
  embedding: { provider: 'fixture', model: 'fixture', dimension: 3 },
  config: { max_context_tokens: 32000, analysis_answer_token_cap: 2048, recall_top_k: 20, return_top_k: 5 }, experiment: { label: 'baseline' } });
const document = (id, title, version = 1) => ({ id, kb_id: 7, title, version, status: 'ready', embedding_model: 'embed-fixture' });
const source = (key, title) => ({ key, kb: 'stocks', title, sha256: sha, source: `ai_document/ai-rag/${key}.txt` });
const sources = [source('a', '甲'), source('b', '乙')];
const positive = { id: 'multi', split: 'development', category: 'multi_document', kb: 'stocks', query: '两个事实？', expected_documents: ['a', 'b'], evidence: [{ document: 'a', groups: [['事实甲']] }, { document: 'b', groups: [['事实乙']] }], answer: { kind: 'answerable', required_groups: [['事实甲'], ['事实乙']], forbidden_phrases: [] }, rationale: '两个来源' };
const negative = { ...positive, id: 'negative', split: 'validation', category: 'near_miss', expected_documents: [], evidence: [], answer: { kind: 'unanswerable', required_groups: [], forbidden_phrases: ['审批人是张三'] } };
const bindings = new Map([['a', { doc_id: 11 }], ['b', { doc_id: 12 }]]);
const hit = (chunk, doc, title, snippet, score = 0.9) => ({ chunk_id: chunk, kb_id: 7, doc_id: doc, doc_title: title, snippet, score });
const hits = [hit(1, 11, '甲', '事实甲'), hit(2, 12, '乙', '事实乙', 0.8)];
const analysis = (answer = '事实甲[1]，事实乙[2]。') => ({ version: 1, type: 'kb_analysis', status: 'completed', analysis: answer, sources: hits.map((item, index) => ({ ...item, index: index + 1 })), usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true } });
const envelope = (result, status = 200) => new Response(JSON.stringify({ code: 200, result }), { status });
function fixture({ changed = false, submitFailure = false, pending = false, badAnswer, analysisOutput, failSecondSearch = false, acceptedStatus = 'running', onTaskRead = () => {} } = {}) {
  const calls = [], events = []; let reads = 0, searches = 0, submissions = 0;
  const fetchImpl = async (url, options) => {
    const parsed = new URL(url), path = parsed.pathname.replace('/v1/sys/ai', '');
    const body = options.body ? JSON.parse(options.body) : undefined;
    calls.push({ path, method: options.method, body, query: parsed.search });
    if (path === '/kbs/7/docs') {
      const page = Number(parsed.searchParams.get('page'));
      return envelope({ total: 2, list: [page === 1 ? document(11, '甲') : document(12, '乙')] });
    }
    if (/\/docs\/\d+$/.test(path)) {
      reads++;
      const id = Number(path.split('/').at(-1));
      return envelope({ doc: document(id, id === 11 ? '甲' : '乙', changed && reads > 2 ? 2 : 1), content: text });
    }
    if (path.endsWith('/search')) {
      if (failSecondSearch && ++searches === 2) return envelope({}, 500);
      return envelope({ list: hits, elapsed_ms: 3 });
    }
    if (path === '/tasks/run') {
      assert.ok(events.find(event => event.type === 'analysis_requested' && event.request_id === body.request_id), 'UUID与原载荷须在POST前持久化');
      if (submitFailure) throw new DOMException('private-access', 'TimeoutError');
      return envelope({ task_id: 51 + submissions++, status: acceptedStatus });
    }
    if (/^\/tasks\/\d+$/u.test(path)) {
      onTaskRead();
      const id = Number(path.split('/').at(-1)), submitted = calls.filter(call => call.path === '/tasks/run')[id - 51];
      return envelope({ task: { id, workflow_id: 0, request_id: submitted.body.request_id, status: pending ? 'running' : 'succeeded', steps: pending ? [] : [{ index: 0, tool: 'kb_analyze', status: 'succeeded', output: JSON.stringify(analysisOutput ?? analysis(badAnswer)) }] } });
    }
    throw new Error('未知桩路径');
  };
  return { calls, events, client: feature('createApiClient')({ token: 'fixture-access', fetchImpl }), args: { dataset: { documents: sources, cases: [positive] }, kbMap: { stocks: 7 }, onProgress: async event => events.push(event), pollIntervalMs: 0 } };
}
async function recordedRun(cases = [positive], fixtureOptions) {
  const f = fixture(fixtureOptions); f.args.dataset = { ...f.args.dataset, cases };
  const result = await feature('runEvaluation')({ ...f.args, client: f.client });
  const records = [{ type: 'meta', version: 1, phase: 'baseline', dataset_sha256: sha, case_ids: cases.map(item => item.id), kb_map: { stocks: 7 },
    context: runtimeContext() },
    ...f.events, { type: 'summary', completed: true, ...result.summary }];
  return { records, validation: { dataset: f.args.dataset, datasetSha256: sha } };
}

test('默认plan、显式run及选择边界，不允许远程或非tmp输出', () => {
  const parse = feature('parseArgs');
  assert.equal(parse([]).run, false);
  assert.equal(parse(['--split', 'validation', '--phase', 'candidate']).split, 'validation');
  assert.throws(() => parse(['--run']), /auth|认证/);
  assert.throws(() => parse(['--base', 'http://example.test']), /本机/);
  assert.throws(() => parse(['--split', 'test']), /split/);
  assert.throws(() => parse(['--output', 'outside.jsonl']), /tmp/);
  assert.throws(() => parse(['--plan', '--run']), /plan|run/);
});

test('多文档Recall分母独立，原始片段排名产生Hit/MRR@1/3/5及按文档证据覆盖', () => {
  const result = feature('evaluateRetrieval')(positive, [hits[0], { ...hits[0], chunk_id: 3 }], new Map([['a', { doc_id: 11 }], ['b', { doc_id: 12 }]]));
  assert.deepEqual(result.hit_at, { 1: 1, 3: 1, 5: 1 });
  assert.deepEqual(result.recall_at, { 1: 0.5, 3: 0.5, 5: 0.5 });
  assert.deepEqual(result.mrr_at, { 1: 1, 3: 1, 5: 1 });
  assert.equal(result.evidence_at[5].coverage, 0.5);
  assert.equal(feature('evaluateRetrieval')(negative, hits, new Map()).kind, 'negative');
});

test('规则筛查同时识别缺少事实、非法/缺少编号、来源越界和混合拒答错答', () => {
  const screen = feature('screenAnswer'), known = new Map([[11, document(11, '甲')], [12, document(12, '乙')]]);
  const good = screen(positive, analysis(), 7, known);
  assert.equal(good.flagged, false);
  assert.equal(good.requires_human_review, true);
  const wrong = screen(positive, { ...analysis('事实甲[9]'), sources: [{ ...analysis().sources[0], kb_id: 8 }] }, 7, known);
  assert.ok(wrong.reasons.includes('invalid_citation'));
  assert.ok(wrong.reasons.includes('source_scope_mismatch'));
  assert.ok(wrong.reasons.includes('missing_required_facts'));
  assert.ok(screen(positive, analysis('事实甲事实乙'), 7, known).reasons.includes('missing_citation'));
  const mixed = screen(negative, analysis('资料不足，但审批人是张三[1]'), 7, known);
  assert.ok(mixed.reasons.includes('forbidden_phrase'));
  assert.equal(mixed.flagged, true, '出现资料不足不能让混合错答通过');
  assert.ok(screen(negative, analysis('答案是张三[1]'), 7, known).reasons.includes('missing_insufficiency'));
  assert.ok(screen(positive, analysis('事实甲[1,2]事实乙[2]'), 7, known).reasons.includes('malformed_citation'));
});

test('分页唯一标题+服务端全文SHA/version绑定，原UUID持久化，正式search和draft分析各一次', async () => {
  const f = fixture();
  const result = await feature('runEvaluation')({ ...f.args, client: f.client });
  assert.equal(result.results.length, 1);
  assert.equal(result.bindings[0].version, 1);
  assert.equal(result.bindings[0].content_sha256, sha);
  assert.equal(f.calls.filter(call => call.path === '/kbs/7/docs').length, 4, '开始分页与结束分页均须核对');
  assert.equal(f.calls.filter(call => call.path === '/tasks/run').length, 1);
  assert.equal(f.calls.filter(call => call.path.endsWith('/search')).length, 1);
  const sent = f.calls.find(call => call.path === '/tasks/run').body;
  assert.match(sent.request_id, /^[a-f0-9-]{36}$/);
  assert.match(sent.name, /^eval-kb-quality-/);
  assert.deepEqual(JSON.parse(sent.definition), { steps: [{ name: '知识库分析', tool: 'kb_analyze', input: { kb_id: 7, query: positive.query } }] });
  assert.equal(sent.input, '{}');
  assert.equal(result.summary.calls.analysis_submissions, 1);
  assert.equal(result.summary.calls.known_chat_calls, 1);
  assert.equal(result.summary.usage.known_total_tokens, 15);
  assert.equal(result.summary.latency.analysis_submit_to_terminal.n, 1);
});

test('运行结束文档版本变化必须失败，不产出可比较summary', async () => {
  const f = fixture({ changed: true });
  await assert.rejects(feature('runEvaluation')({ ...f.args, client: f.client }), /变化|version|版本/);
  assert.equal(f.events.filter(event => event.type === 'result').length, 1);
  assert.equal(f.events.filter(event => event.type === 'corpus_verified').length, 0);
});

test('未知POST不重发，原UUID/payload保留且不伪装为错误回答或空召回', async () => {
  const f = fixture({ submitFailure: true });
  await assert.rejects(feature('runEvaluation')({ ...f.args, client: f.client }), /超时|未知/);
  assert.equal(f.calls.filter(call => call.path === '/tasks/run').length, 1);
  assert.equal(f.events.filter(event => event.type === 'analysis_requested').length, 1);
  assert.equal(f.events.filter(event => event.type === 'result').length, 0);
  assert.equal(f.events.filter(event => event.type === 'search_result').length, 1);
});

test('GET轮询超时保留已接受task ID，不重新提交、不继续下一题', async () => {
  const f = fixture({ pending: true }); let tick = 0;
  await assert.rejects(feature('runEvaluation')({ ...f.args, dataset: { documents: sources, cases: [positive, negative] }, client: f.client, now: () => tick++ * 10, taskTimeoutMs: 30 }), /超时/);
  assert.equal(f.calls.filter(call => call.path === '/tasks/run').length, 1);
  assert.equal(f.events.find(event => event.type === 'analysis_accepted').task_id, 51);
  assert.equal(f.events.filter(event => event.type === 'result').length, 0);
});

test('100秒内观察到终态后固定耗时，终态日志持久化延迟不使完整运行失效', async () => {
  for (const terminalAt of [99999, 100000]) {
    let clock = 0;
    const f = fixture({ onTaskRead: () => { clock = terminalAt; } });
    const result = await feature('runEvaluation')({ ...f.args, client: f.client, now: () => clock, onProgress: async event => {
      await f.args.onProgress(event);
      if (event.type === 'analysis_terminal') clock += 20;
    } });
    assert.equal(result.results[0].analysis_elapsed_ms, terminalAt);
    const records = [{ type: 'meta', version: 1, phase: 'baseline', dataset_sha256: sha, case_ids: [positive.id], kb_map: { stocks: 7 }, context: runtimeContext() },
      ...f.events, { type: 'summary', completed: true, ...result.summary }];
    const candidate = structuredClone(records); candidate[0].phase = 'candidate';
    assert.equal(feature('compareRuns')(records, candidate, { dataset: f.args.dataset, datasetSha256: sha }).comparable, true);
  }
});

test('超过100秒才返回终态仍停止且不保存成功题或重发', async () => {
  let clock = 0;
  const f = fixture({ onTaskRead: () => { clock = 100001; } });
  await assert.rejects(feature('runEvaluation')({ ...f.args, client: f.client, now: () => clock }), /超时/u);
  assert.equal(f.client.counts.analysis_submissions, 1); assert.equal(f.client.counts.task_reads, 1);
  assert.equal(f.events.some(row => row.type === 'result' || row.type === 'analysis_terminal'), false);
});

for (const status of ['failed', 'cancelled', 'rejected', 'pending_approval']) {
  test(`POST已接受${status}先保存原任务再停止，不轮询或进入下一题`, async () => {
    const f = fixture({ acceptedStatus: status });
    await assert.rejects(feature('runEvaluation')({ ...f.args, dataset: { documents: sources, cases: [positive, negative] }, client: f.client }), /状态|终态|审批/u);
    assert.deepEqual(f.events.find(row => row.type === 'analysis_accepted'), {
      type: 'analysis_accepted', case_id: positive.id, request_id: f.events.find(row => row.type === 'analysis_requested').request_id, task_id: 51, status,
    });
    assert.equal(f.client.counts.analysis_submissions, 1);
    assert.equal(f.client.counts.task_reads, 0);
    assert.equal(f.events.filter(row => row.type === 'case_started').length, 1);
    assert.equal(f.events.some(row => row.type === 'result'), false);
  });

  test(`成功离线链拒绝已接受${status}后转为succeeded`, async () => {
    const { records, validation } = await recordedRun(), candidate = structuredClone(records);
    candidate[0].phase = 'candidate'; candidate.find(row => row.type === 'analysis_accepted').status = status;
    assert.equal(feature('compareRuns')(records, candidate, validation).comparable, false);
    assert.throws(() => feature('rescoreRecords')(candidate, { ...validation, sourceSha256: sha }), /状态|接受/u);
  });
}

test('第二题HTTP500保留第一题结果，不计入有效题分母', async () => {
  const f = fixture({ failSecondSearch: true });
  await assert.rejects(feature('runEvaluation')({ ...f.args, dataset: { documents: sources, cases: [positive, negative] }, client: f.client }), /500/);
  assert.equal(f.events.filter(event => event.type === 'result').length, 1);
  assert.equal(f.calls.filter(call => call.path === '/tasks/run').length, 1);
});

test('报告保留答案/来源和人工复核边界，比较要求同题同语料同模型上下文且拒绝不完整', async () => {
  const { records, validation } = await recordedRun();
  const markdown = feature('renderReport')(records);
  assert.match(markdown, /事实甲/); assert.match(markdown, /人工复核/); assert.match(markdown, /串行/);
  const candidate = structuredClone(records); candidate[0].phase = 'candidate'; candidate[0].context.binary_sha = 'b'.repeat(64);
  assert.equal(feature('compareRuns')(records, candidate, validation).comparable, true);
  candidate[0].context.config.max_context_tokens = 16000;
  assert.equal(feature('compareRuns')(records, candidate, validation).comparable, false);
  assert.equal(feature('compareRuns')(records, records.slice(0, -1), validation).comparable, false);
  const reordered = structuredClone(records); reordered[0].phase = 'candidate';
  const context = reordered[0].context;
  reordered[0].context = { experiment: context.experiment, config: context.config, source_revision: context.source_revision, embedding: context.embedding, model: context.model, binary_sha: 'b'.repeat(64) };
  assert.equal(feature('compareRuns')(records, reordered, validation).comparable, true, '对象字段顺序不改变可比性');
});

test('多文档必须实际引用每篇预期文档，且已引用片段覆盖对应证据组', () => {
  const screen = feature('screenAnswer'), known = new Map([[11, document(11, '甲')], [12, document(12, '乙')]]);
  const partial = screen(positive, analysis('事实甲[1]，事实乙。'), 7, known, bindings);
  assert.ok(partial.reasons.includes('missing_expected_citations'));
  assert.ok(partial.reasons.includes('cited_evidence_incomplete'));
  const wrongSnippet = analysis(); wrongSnippet.sources[1].snippet = '不相关事实';
  assert.ok(screen(positive, wrongSnippet, 7, known, bindings).reasons.includes('cited_evidence_incomplete'));
  assert.equal(screen(positive, analysis(), 7, known, bindings).flagged, false);
});

test('运行context敏感字段拒绝，预算token字段允许', () => {
  const context = runtimeContext();
  assert.deepEqual(feature('validateContext')(context), context);
  assert.throws(() => feature('validateContext')({ ...context, api_key: 'private' }), /敏感/);
  assert.throws(() => feature('validateContext')({ ...context, label: 'Bearer fixture-private' }), /凭据/);
});

test('context拒绝缺失模型与预算标签、无效构建摘要和越界数值', async () => {
  const validate = feature('validateContext'), { records, validation } = await recordedRun();
  const invalid = [
    context => context.binary_sha = 'aaa', context => context.binary_sha = 'g'.repeat(64),
    ...['model', 'embedding', 'config'].flatMap(key => [undefined, null, [], {}, 'fixture'].map(value => context => context[key] = value)),
    context => context.model.provider = ' ', context => delete context.model.name,
    context => context.embedding.provider = '', context => delete context.embedding.model,
    ...[0, -1, 1.5, '3', Number.MAX_SAFE_INTEGER + 1].map(value => context => context.embedding.dimension = value),
    ...['max_context_tokens', 'analysis_answer_token_cap', 'recall_top_k', 'return_top_k'].flatMap(key => [undefined, 0, -1, 1.5, '5', Number.MAX_SAFE_INTEGER + 1].map(value => context => context.config[key] = value)),
    context => context.config.analysis_answer_token_cap = 32001, context => context.config.return_top_k = 21,
    ...[null, '', '0', 'default', NaN, Infinity, -Infinity, -0.1].map(value => context => context.config.analysis_temperature = value),
  ];
  for (const mutate of invalid) {
    const context = runtimeContext(); mutate(context);
    assert.throws(() => validate(context), /context|模型|Embedding|预算|temperature|摘要/u);
    const candidate = structuredClone(records); candidate[0].phase = 'candidate'; candidate[0].context = context;
    assert.equal(feature('compareRuns')(records, candidate, validation).comparable, false);
    assert.throws(() => feature('rescoreRecords')(candidate, { ...validation, sourceSha256: sha }), /context|模型|Embedding|预算|temperature|摘要/u);
  }
});

test('context保留省略temperature的历史标签与数值0，运行标签差异拒绝混比', async () => {
  const validate = feature('validateContext'), oldContext = runtimeContext();
  assert.equal(validate(oldContext), oldContext); assert.equal('analysis_temperature' in oldContext.config, false);
  for (const value of [0, 0.5, 2, 2.1, 'provider_default']) {
    const context = runtimeContext(); context.config.analysis_temperature = value;
    assert.equal(validate(context).config.analysis_temperature, value);
  }
  const { records, validation } = await recordedRun(), candidate = structuredClone(records); candidate[0].phase = 'candidate';
  assert.equal(feature('compareRuns')(records, candidate, validation).comparable, true);
  candidate[0].context.config.analysis_temperature = 'provider_default';
  assert.equal(feature('compareRuns')(records, candidate, validation).comparable, false);
  const explicitDefault = structuredClone(candidate); explicitDefault[0].phase = 'baseline';
  candidate[0].context.config.analysis_temperature = 0;
  assert.equal(feature('compareRuns')(records, candidate, validation).comparable, false);
  assert.equal(feature('compareRuns')(explicitDefault, candidate, validation).comparable, false);
});

test('实际40题素材哈希与证据通过；负例不能附带必需事实或无依据正例', async () => {
  const load = feature('loadDataset'), dataset = await load();
  assert.equal(dataset.cases.length, 40);
  assert.equal(dataset.cases.filter(item => item.split === 'development').length, 24);
  assert.equal(dataset.cases.filter(item => item.split === 'validation').length, 16);
  assert.equal(dataset.documents.length, 11);
  const directory = await mkdtemp(join(tmpdir(), 'gozero-quality-cases-'));
  try {
    const path = join(directory, 'invalid.json'), invalid = structuredClone(dataset);
    invalid.cases = [invalid.cases.find(item => item.answer.kind === 'unanswerable')];
    invalid.cases[0].answer.required_groups = [['编造事实']];
    await writeFile(path, JSON.stringify(invalid));
    await assert.rejects(load(path), /负例/);
    invalid.cases = [structuredClone(dataset.cases.find(item => item.answer.kind === 'answerable'))];
    invalid.cases[0].answer.required_groups = [['该事实不在任何资料abcdef']];
    await writeFile(path, JSON.stringify(invalid));
    await assert.rejects(load(path), /必需|原文/);
  } finally { await rm(directory, { recursive: true, force: true }); }
});

test('比较允许显式实验/构建变化，但拒绝所选题、语料版本、模型与预算变化', async () => {
  const { records: run, validation } = await recordedRun();
  const candidate = structuredClone(run); candidate[0].phase = 'candidate'; candidate[0].context.experiment.label = 'candidate'; candidate[0].context.source_revision = 'b';
  const comparison = feature('compareRuns')(run, candidate, validation);
  assert.equal(comparison.comparable, true);
  assert.deepEqual(comparison.allowed_changes.map(row => row.field), ['source_revision', 'experiment']);
  for (const mutate of [rows => rows[0].case_ids.push('extra'), rows => rows[1].documents[0].version++, rows => rows[0].context.model.name = 'other', rows => rows[0].context.config.max_context_tokens = 16000]) {
    const changed = structuredClone(candidate); mutate(changed);
    assert.equal(feature('compareRuns')(run, changed, validation).comparable, false);
  }
});

test('比较从冻结题集及原始记录重算，指标/调用数/分组/事件缺失不允许假完成', async () => {
  const { records, validation } = await recordedRun();
  const candidate = structuredClone(records); candidate[0].phase = 'candidate';
  for (const mutate of [
    rows => rows.find(row => row.type === 'result').retrieval.recall_at[5] = 999,
    rows => delete rows.find(row => row.type === 'result').retrieval,
    rows => rows.find(row => row.type === 'result').analysis_elapsed_ms = null,
    rows => rows.find(row => row.type === 'result').category = 'fabricated',
    rows => rows.find(row => row.type === 'result').query = 'changed-query',
    rows => rows.find(row => row.type === 'result').answer_screen.flagged = true,
    rows => rows.find(row => row.type === 'summary').calls.known_chat_calls = 0,
    rows => rows.find(row => row.type === 'analysis_accepted').request_id = 'other',
    rows => rows.splice(rows.findIndex(row => row.type === 'analysis_terminal'), 1),
    rows => rows.push({ type: 'error', message: 'partial' }),
  ]) {
    const invalid = structuredClone(candidate); mutate(invalid);
    assert.equal(feature('compareRuns')(records, invalid, validation).comparable, false);
  }
  assert.equal(feature('compareRuns')(records, candidate, { ...validation, datasetSha256: 'b'.repeat(64) }).comparable, false);
});

test('v1完整日志必须按case_ids串行完成六事件，拒绝交错、乱序和未知事件', async () => {
  const { records, validation } = await recordedRun([positive, negative]);
  const good = structuredClone(records); good[0].phase = 'candidate';
  assert.equal(feature('compareRuns')(records, good, validation).comparable, true);
  const eventTypes = ['case_started', 'search_result', 'analysis_requested', 'analysis_accepted', 'analysis_terminal', 'result'];
  const changes = [
    rows => [rows[0], rows[1], ...eventTypes.flatMap(type => rows.filter(row => row.type === type)), ...rows.slice(-2)],
    rows => { rows[0].case_ids.reverse(); return rows; },
    rows => { rows.splice(-1, 0, { type: 'analysis_failed', case_id: positive.id, status: 'failed' }); return rows; },
    rows => { rows.splice(2, 0, { type: 'future_event' }); return rows; },
  ];
  for (const change of changes) {
    const invalid = change(structuredClone(good));
    assert.equal(feature('compareRuns')(records, invalid, validation).comparable, false);
    assert.throws(() => feature('rescoreRecords')(invalid, { ...validation, sourceSha256: sha }), /顺序|事件|串行/u);
    assert.match(feature('renderReport')(invalid, validation), /失败或不完整；不可比较/u);
  }
});

test('同库非bound来源、重复分块和非规范引用编号均进入待人工复核', () => {
  const known = new Map([[11, document(11, '甲')], [12, document(12, '乙')], [99, document(99, '额外资料')]]);
  const other = analysis(); other.sources.push({ ...hits[0], index: 3, doc_id: 99, chunk_id: 99, doc_title: '额外资料' });
  assert.ok(feature('screenAnswer')(positive, other, 7, known, bindings).reasons.includes('unbound_source'));
  const duplicate = analysis(); duplicate.sources[1].chunk_id = duplicate.sources[0].chunk_id;
  assert.ok(feature('screenAnswer')(positive, duplicate, 7, known, bindings).reasons.includes('duplicate_source_chunk'));
  assert.ok(feature('screenAnswer')(positive, analysis('事实甲[01]事实乙[2]'), 7, known, bindings).reasons.includes('malformed_citation'));
});

test('损坏bindings/case_ids结构返回不可比及原因，不在二次比较时抛TypeError', async () => {
  const { records, validation } = await recordedRun();
  for (const mutate of [
    rows => rows.find(row => row.type === 'bindings').documents = {},
    rows => rows[0].context.config.api_key = 'private',
    rows => rows.find(row => row.type === 'bindings').documents = [null],
    rows => rows[0].case_ids = {},
  ]) {
    const candidate = structuredClone(records); candidate[0].phase = 'candidate'; mutate(candidate);
    const result = feature('compareRuns')(records, candidate, validation);
    assert.equal(result.comparable, false);
    assert.ok(result.reasons.some(reason => reason.startsWith('candidate：')));
  }
});

test('损坏语料绑定的离线CLI输出不可比原因并非零退出', async () => {
  const { records } = await recordedRun(), dataset = await feature('loadDataset')();
  const datasetBytes = await readFile(new URL('../../../ai_document/ai-rag/quality-cases.json', import.meta.url));
  records[0].dataset_sha256 = createHash('sha256').update(datasetBytes).digest('hex');
  records[0].case_ids = [dataset.cases[0].id];
  for (const row of records) if (row.case_id) row.case_id = dataset.cases[0].id;
  const candidate = structuredClone(records); candidate[0].phase = 'candidate'; candidate.find(row => row.type === 'bindings').documents = {};
  const directory = await mkdtemp(join(tmpdir(), 'gozero-quality-cli-'));
  try {
    const before = join(directory, 'baseline.jsonl'), after = join(directory, 'candidate.jsonl');
    await writeFile(before, records.map(JSON.stringify).join('\n') + '\n');
    await writeFile(after, candidate.map(JSON.stringify).join('\n') + '\n');
    await assert.rejects(promisify(execFile)(process.execPath, [fileURLToPath(new URL('./eval-kb-quality.mjs', import.meta.url)), '--compare', before, after]), error => {
      assert.equal(error.code, 1);
      assert.match(error.stdout, /不可比较/);
      assert.match(error.stdout, /candidate：缺少实际语料绑定或知识库库存/);
      assert.equal(error.stderr, '');
      return true;
    });
  } finally { await rm(directory, { recursive: true, force: true }); }
});

test('任务结果只读结构化steps输出，非法用量与缺少sources不能冒充成功', () => {
  const parse = feature('parseAnalysis');
  const task = output => ({ result: '[1/1] 知识库分析: 不是JSON', steps: [{ index: 0, status: 'succeeded', tool: 'kb_analyze', output: JSON.stringify(output) }] });
  assert.equal(parse(task(analysis())).analysis, '事实甲[1]，事实乙[2]。');
  assert.throws(() => parse({ ...task(analysis()), steps: [] }), /步骤/);
  assert.throws(() => parse(task({ ...analysis(), sources: [] })), /来源/);
  assert.throws(() => parse(task({ ...analysis(), usage: { ...analysis().usage, total_tokens: 99 } })), /用量/);
});

test('no_matches仅允许零用量且complete=false，运行及离线重算拒绝矛盾聊天计数', async () => {
  const empty = { ...analysis('知识库未检索到相关资料，无法完成分析。'), status: 'no_matches', sources: [],
    usage: { scope: 'chat', prompt_tokens: 0, completion_tokens: 0, total_tokens: 0, complete: false } };
  const { records, validation } = await recordedRun([negative], { analysisOutput: empty });
  assert.equal(records.at(-1).calls.known_chat_calls, 0); assert.equal(records.at(-1).usage.known_total_tokens, 0);
  assert.equal(records.at(-1).usage.complete_cases, 0);
  const candidate = structuredClone(records); candidate[0].phase = 'candidate';
  assert.equal(feature('compareRuns')(records, candidate, validation).comparable, true);
  for (const usage of [
    { ...empty.usage, prompt_tokens: 1 }, { ...empty.usage, completion_tokens: 1 }, { ...empty.usage, total_tokens: 1 },
    { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true },
  ]) {
    const malformed = { ...empty, usage }, f = fixture({ analysisOutput: malformed });
    await assert.rejects(feature('runEvaluation')({ ...f.args, client: f.client }), /用量/u);
    assert.equal(f.events.some(row => row.type === 'result'), false);
    const invalid = structuredClone(candidate); invalid.find(row => row.type === 'result').analysis.usage = usage;
    assert.throws(() => feature('rescoreRecords')(invalid, { ...validation, sourceSha256: sha }), /用量/u);
    const comparison = feature('compareRuns')(records, invalid, validation);
    assert.equal(comparison.comparable, false); assert.ok(comparison.reasons.some(reason => /用量/u.test(reason)));
  }
});

test('实际坏答fixture保留原case与来源SHA，显式forbidden把三条缺口断言列入复核', async () => {
  const fixture = JSON.parse(await readFile(new URL('../../../ai_document/ai-rag/answer-failure-fixtures.json', import.meta.url), 'utf8'));
  const dataset = await feature('loadDataset')(), datasetBytes = await readFile(new URL('../../../ai_document/ai-rag/quality-cases.json', import.meta.url));
  assert.equal(fixture.provenance.dataset_sha256, createHash('sha256').update(datasetBytes).digest('hex'));
  assert.equal(fixture.cases.length, 3);
  const known = new Map(fixture.bound_documents.map(doc => [doc.doc_id, { ...doc, id: doc.doc_id }]));
  const bound = new Map(fixture.bound_documents.map(doc => [doc.key, doc]));
  assert.equal(fixture.cases.filter(item => item.manual_review.outcome === 'incorrect_claim').length, 1);
  assert.equal(fixture.cases.filter(item => item.manual_review.outcome === 'scope_ambiguity').length, 2);
  for (const saved of fixture.cases) {
    const item = dataset.cases.find(item => item.id === saved.case_id);
    assert.ok(item); assert.equal(saved.query, item.query); assert.equal(saved.original_split, item.split);
    assert.equal(saved.original_sources.length, 5);
    for (const source of saved.original_sources) {
      const recordedHash = saved.source_hashes.find(row => row.index === source.index), material = known.get(source.doc_id);
      assert.equal(recordedHash.snippet_sha256, createHash('sha256').update(source.snippet).digest('hex'));
      assert.equal(recordedHash.document_sha256, material.sha256);
      const bytes = await readFile(new URL('../../../' + recordedHash.document_source, import.meta.url));
      assert.equal(createHash('sha256').update(bytes).digest('hex'), recordedHash.document_sha256);
    }
    const original = { status: 'completed', analysis: saved.original_answer, sources: saved.original_sources };
    const oldScreen = feature('screenAnswer')(item, original, saved.kb_id, known, bound);
    assert.deepEqual(oldScreen.reasons, [], '旧40题不添加策略时继续保持原筛查行为');
    const explicit = { ...item, answer: { ...item.answer, absence_policy: saved.absence_policy } };
    assert.deepEqual(feature('screenAnswer')(explicit, original, saved.kb_id, known, bound).reasons, [saved.expected_rule_reason]);
    assert.equal(feature('screenAnswer')(explicit, { ...original, analysis: saved.corrected_answer }, saved.kb_id, known, bound).flagged, false);
  }
  assert.doesNotMatch(JSON.stringify(fixture), /request_id|access_token|refresh_token|session_token|Bearer\s/iu);
});

test('可选absence_policy枚举严格校验，不向旧题插入默认字段', async () => {
  const load = feature('loadDataset'), dataset = await load(), directory = await mkdtemp(join(tmpdir(), 'gozero-quality-policy-'));
  const originalAnswerKeys = Object.keys(dataset.cases[0].answer);
  assert.equal(originalAnswerKeys.includes('absence_policy'), false);
  try {
    const path = join(directory, 'cases.json'), candidate = structuredClone(dataset); candidate.cases = [candidate.cases.find(item => item.answer.kind === 'answerable')];
    for (const value of ['forbidden', 'required']) {
      candidate.cases[0].answer.absence_policy = value; await writeFile(path, JSON.stringify(candidate));
      assert.equal((await load(path)).cases[0].answer.absence_policy, value);
    }
    for (const value of [null, 'optional', '', [], {}]) {
      candidate.cases[0].answer.absence_policy = value; await writeFile(path, JSON.stringify(candidate));
      await assert.rejects(load(path), /absence_policy/);
    }
    assert.deepEqual(Object.keys((await load()).cases[0].answer), originalAnswerKeys);
  } finally { await rm(directory, { recursive: true, force: true }); }
});

test('required仅检查部分缺失题是否表达不足；正常全负拒答及旧输出结构保持兼容', () => {
  const known = new Map([[11, document(11, '甲')], [12, document(12, '乙')]]), screen = feature('screenAnswer');
  const partial = { ...positive, answer: { ...positive.answer, absence_policy: 'required' } };
  assert.deepEqual(screen(partial, analysis(), 7, known, bindings).reasons, ['missing_scoped_insufficiency']);
  assert.equal(screen(partial, analysis('事实甲[1]，事实乙[2]。资料未提供审批人姓名，无法确定。'), 7, known, bindings).flagged, false);
  assert.equal(screen(negative, analysis('资料不足，无法确定。'), 7, known, bindings).flagged, false);
  const oldOutput = screen(positive, analysis(), 7, known, bindings);
  assert.deepEqual(oldOutput, { label: '规则筛查，非语义准确率或人工评审', flagged: false, requires_human_review: true, reasons: [],
    required_groups_matched: [true, true], forbidden_phrases_matched: [], citation_indices: [1, 2], cited_evidence_groups_matched: [true, true] });
});

test('absence_policy只做词法提示：引用/否定也可能误报，required命中不证明未知范围正确', () => {
  const known = new Map([[11, document(11, '甲')], [12, document(12, '乙')]]), screen = feature('screenAnswer');
  const forbidden = { ...positive, answer: { ...positive.answer, absence_policy: 'forbidden' } };
  for (const tail of ['不要写“资料不足”这句话。', '并非资料不足，所问事实均已提供。']) {
    const result = screen(forbidden, analysis(`事实甲[1]，事实乙[2]。${tail}`), 7, known, bindings);
    assert.deepEqual(result.reasons, ['additional_absence_claim_needs_review'], '引用与否定仍会触发词法复核，不能据此自动判错');
    assert.equal(result.requires_human_review, true); assert.equal('answer_correct' in result, false);
  }
  const required = { ...positive, answer: { ...positive.answer, absence_policy: 'required' } };
  assert.equal(screen(required, analysis('事实甲[1]，事实乙[2]。并非资料不足。'), 7, known, bindings).flagged, false,
    '否定句也会命中不足词，未标记不能证明已说明缺失目标，仍需人工复核');
});

test('v1实测未涉及尾注有独立provenance并触发复核，正答不触发', async () => {
  const fixture = JSON.parse(await readFile(new URL('../../../ai_document/ai-rag/answer-failure-fixtures.json', import.meta.url), 'utf8'));
  assert.match(fixture.description, /AI辅助.*无真人签署/);
  assert.equal(fixture.cases.length, 3); assert.equal(fixture.variants.length, 1);
  const variant = fixture.variants[0];
  assert.equal(variant.case_id, 'answer-regression-second-line-repeat-1');
  assert.equal(variant.source_case_id, 'quality-dev-incident-second-line');
  assert.equal(variant.provenance.candidate_binary_sha256, '72d435004ce5535196d9e750e25555eec0c1b095419a945509989b446dd19dc7');
  assert.equal(variant.provenance.source_log_sha256, '881a99e7d0e4b4480233f72942d9b8da33589a9e8073316ee92a65a86ecc9a07');
  const dataset = await feature('loadDataset')(new URL('../../../ai_document/ai-rag/answer-regression-cases.json', import.meta.url));
  const item = dataset.cases.find(item => item.id === variant.case_id);
  const known = new Map(fixture.bound_documents.map(doc => [doc.doc_id, { ...doc, id: doc.doc_id }]));
  const bound = new Map(fixture.bound_documents.map(doc => [doc.key, doc]));
  for (const source of variant.original_sources) {
    const recorded = variant.source_hashes.find(row => row.index === source.index);
    assert.equal(recorded.snippet_sha256, createHash('sha256').update(source.snippet).digest('hex'));
    assert.equal(recorded.document_sha256, known.get(source.doc_id).sha256);
  }
  const original = { status: 'completed', analysis: variant.original_answer, sources: variant.original_sources };
  assert.deepEqual(feature('screenAnswer')(item, original, variant.kb_id, known, bound).reasons, [variant.expected_rule_reason]);
  assert.equal(feature('screenAnswer')(item, { ...original, analysis: variant.corrected_answer }, variant.kb_id, known, bound).flagged, false);
});

test('未涉及/未包含/未能确定同时支持缺口复核与服务端固定部分缺失文案', () => {
  const known = new Map([[11, document(11, '甲')], [12, document(12, '乙')]]), screen = feature('screenAnswer');
  const forbidden = { ...positive, answer: { ...positive.answer, absence_policy: 'forbidden' } };
  const required = { ...positive, answer: { ...positive.answer, absence_policy: 'required' } };
  for (const clause of ['本次片段未涉及其他升级时限。', '本次片段未包含其他升级时限。', '本次片段未能确定“审批人姓名”。']) {
    const output = analysis(`事实甲[1]，事实乙[2]。${clause}`);
    assert.deepEqual(screen(forbidden, output, 7, known, bindings).reasons, ['additional_absence_claim_needs_review']);
    assert.equal(screen(required, output, 7, known, bindings).flagged, false);
  }
  assert.equal(screen(negative, analysis('本次片段资料不足，无法回答该分析目标。'), 7, known, bindings).flagged, false);
});

test('离线rescore只重算派生字段，保留原始事件/模型/UUID/来源/计时且同hash严格复验', async () => {
  const { records, validation } = await recordedRun(), source = structuredClone(records);
  source.find(row => row.type === 'search_result').retrieval.recall_at[5] = 999;
  source.find(row => row.type === 'result').answer_screen.reasons = ['obsolete_rule'];
  source.find(row => row.type === 'summary').answer_rule_screen.flagged_count = 123;
  const sourceBytes = JSON.stringify(source), sourceSha256 = createHash('sha256').update(sourceBytes).digest('hex');
  const rescored = feature('rescoreRecords')(source, { ...validation, sourceSha256, at: '2026-11-07T00:00:00.000Z' });
  assert.equal(JSON.stringify(source), sourceBytes, '源记录不得原地修改');
  assert.deepEqual(rescored[0].derived_from, { source_sha256: sourceSha256, at: '2026-11-07T00:00:00.000Z', operation: 'offline_rescore' });
  assert.equal(rescored.find(row => row.type === 'search_result').retrieval.recall_at[5], 1);
  assert.deepEqual(rescored.find(row => row.type === 'result').answer_screen.reasons, []);
  const immutable = rows => rows.map(row => {
    const copy = structuredClone(row);
    if (copy.type === 'meta') delete copy.derived_from;
    if (copy.type === 'search_result') delete copy.retrieval;
    if (copy.type === 'result') { delete copy.retrieval; delete copy.answer_screen; }
    if (copy.type === 'summary') { for (const key of Object.keys(feature('summarize')([], {}))) if (key !== 'calls') delete copy[key]; }
    return copy;
  });
  assert.deepEqual(immutable(rescored), immutable(source));
  const candidate = structuredClone(rescored); candidate[0].phase = 'candidate';
  assert.equal(feature('compareRuns')(rescored, candidate, validation).comparable, true);
  assert.match(feature('renderReport')(rescored, validation), /离线.*重算/);
  for (const mutate of [
    rows => rows.find(row => row.type === 'analysis_accepted').request_id = 'wrong',
    rows => rows.find(row => row.type === 'summary').calls.known_chat_calls = 0,
    rows => rows[0].context.config.api_key = 'private',
    rows => rows.find(row => row.type === 'bindings').documents = {},
    rows => rows.push({ type: 'error', message: 'partial' }),
  ]) { const invalid = structuredClone(source); mutate(invalid); assert.throws(() => feature('rescoreRecords')(invalid, { ...validation, sourceSha256 }), /离线|原始|复算|不完整/); }
  assert.throws(() => feature('rescoreRecords')(source, { ...validation, sourceSha256, datasetSha256: 'b'.repeat(64) }), /摘要|SHA256/);
});

test('rescore CLI模式必须新tmp输出且不能混用run/选择题目', () => {
  const parse = feature('parseArgs');
  assert.throws(() => parse(['--rescore', 'source.jsonl']), /output/);
  assert.equal(parse(['--rescore', 'source.jsonl', '--output', 'tmp/rescored.jsonl']).rescore, 'source.jsonl');
  assert.throws(() => parse(['--rescore', 'source.jsonl', '--output', 'tmp/rescored.jsonl', '--run']), /混用|同时/);
  assert.throws(() => parse(['--rescore', 'source.jsonl', '--output', 'tmp/rescored.jsonl', '--case-ids', 'multi']), /范围|选择|case-ids/);
});

for (const [label, status] of [['缺失', undefined], ['非法', 'not-valid']]) {
  test(`已接受任务status${label}时rescore拒绝、compare不可比`, async () => {
    const { records, validation } = await recordedRun(), invalid = structuredClone(records);
    const accepted = invalid.find(row => row.type === 'analysis_accepted');
    if (status === undefined) delete accepted.status; else accepted.status = status;
    const sourceSha256 = createHash('sha256').update(JSON.stringify(invalid)).digest('hex');
    assert.throws(() => feature('rescoreRecords')(invalid, { ...validation, sourceSha256 }), /状态|接受/);
    invalid[0].phase = 'candidate';
    const comparison = feature('compareRuns')(records, invalid, validation);
    assert.equal(comparison.comparable, false);
    assert.ok(comparison.reasons.some(reason => /状态|接受/u.test(reason)));
  });
}

test('离线rescore CLI在禁止网络时roundtrip，拒绝覆写、坏事件与异题集SHA且不改源文件', async () => {
  const dataset = await feature('loadDataset')(), selectedCases = dataset.cases.slice(0, 2), item = selectedCases[0], progress = [];
  const materials = await Promise.all(dataset.documents.filter(doc => doc.kb === item.kb).map(async (source, index) => ({ source,
    doc: document(index + 1, source.title), content: await readFile(new URL('../../../' + source.source, import.meta.url), 'utf8') })));
  const target = materials.find(row => row.source.key === item.expected_documents[0]);
  const actualHit = hit(1, target.doc.id, target.doc.title, target.content);
  const output = { ...analysis(), analysis: item.answer.required_groups.map(group => group[0]).join('；') + '[1]', sources: [{ ...actualHit, index: 1 }] };
  let taskID = 50; const requestIDs = new Map();
  const client = feature('createApiClient')({ token: 'fixture-access', fetchImpl: async (url, options) => {
    const path = new URL(url).pathname.replace('/v1/sys/ai', '');
    if (path === '/kbs/7/docs') return envelope({ list: materials.map(row => row.doc), total: materials.length });
    if (/\/docs\/\d+$/u.test(path)) { const row = materials.find(row => row.doc.id === Number(path.split('/').at(-1))); return envelope({ doc: row.doc, content: row.content }); }
    if (path.endsWith('/search')) return envelope({ list: [actualHit], elapsed_ms: 3 });
    if (path === '/tasks/run') { taskID++; requestIDs.set(taskID, JSON.parse(options.body).request_id); return envelope({ task_id: taskID, status: 'running' }); }
    if (/^\/tasks\/\d+$/u.test(path)) {
      const id = Number(path.split('/').at(-1));
      return envelope({ task: { id, workflow_id: 0, request_id: requestIDs.get(id), status: 'succeeded', steps: [{ index: 0, tool: 'kb_analyze', status: 'succeeded', output: JSON.stringify(output) }] } });
    }
    throw new Error('非法桩路径');
  } });
  const run = await feature('runEvaluation')({ dataset: { ...dataset, cases: selectedCases }, kbMap: { [item.kb]: 7 }, client, pollIntervalMs: 0, onProgress: async row => progress.push(row) });
  const caseBytes = await readFile(new URL('../../../ai_document/ai-rag/quality-cases.json', import.meta.url));
  const rows = [{ type: 'meta', version: 1, phase: 'baseline', dataset_sha256: createHash('sha256').update(caseBytes).digest('hex'), case_ids: selectedCases.map(item => item.id), kb_map: { [item.kb]: 7 },
    context: runtimeContext() }, ...progress, { type: 'summary', completed: true, ...run.summary }];
  rows.find(row => row.type === 'result').answer_screen.reasons = ['obsolete_rule'];
  const repositoryTmp = fileURLToPath(new URL('../../../tmp/', import.meta.url)); await mkdir(repositoryTmp, { recursive: true });
  const directory = await mkdtemp(join(repositoryTmp, 'quality-rescore-test-'));
  const invoke = (...args) => promisify(execFile)(process.execPath, ['--import', 'data:text/javascript,' + encodeURIComponent('globalThis.fetch=()=>{throw Error("unexpected HTTP")};'),
    fileURLToPath(new URL('./eval-kb-quality.mjs', import.meta.url)), ...args]);
  try {
    const source = join(directory, 'source.jsonl'), destination = join(directory, 'rescored.jsonl'), text = rows.map(JSON.stringify).join('\n') + '\n';
    await writeFile(source, text);
    const result = await invoke('--rescore', source, '--output', destination);
    assert.equal(JSON.parse(result.stdout).http_requests, 0); assert.equal(JSON.parse(result.stdout).mode, 'offline_rescore'); assert.equal(result.stderr, '');
    assert.equal(await readFile(source, 'utf8'), text);
    const rescored = await feature('readRecords')(destination);
    assert.equal(rescored[0].derived_from.source_sha256, createHash('sha256').update(text).digest('hex'));
    assert.deepEqual(rescored.find(row => row.type === 'result').answer_screen.reasons, []);
    assert.deepEqual(rescored.find(row => row.type === 'result').analysis, rows.find(row => row.type === 'result').analysis);
    await assert.rejects(invoke('--rescore', source, '--output', destination), error => error.code === 1 && /存在|覆盖/u.test(error.stderr));
    const invalid = structuredClone(rows); invalid.find(row => row.type === 'analysis_accepted').request_id = 'wrong';
    const badSource = join(directory, 'bad-source.jsonl'), badOutput = join(directory, 'bad-output.jsonl'); await writeFile(badSource, invalid.map(JSON.stringify).join('\n') + '\n');
    await assert.rejects(invoke('--rescore', badSource, '--output', badOutput), error => error.code === 1 && /UUID|复算/u.test(error.stderr));
    await assert.rejects(readFile(badOutput), error => error.code === 'ENOENT');
    for (const status of [undefined, 'not-valid', 'failed', 'cancelled', 'rejected', 'pending_approval']) {
      const invalidStatus = structuredClone(rows), accepted = invalidStatus.find(row => row.type === 'analysis_accepted');
      if (status === undefined) delete accepted.status; else accepted.status = status;
      await writeFile(badSource, invalidStatus.map(JSON.stringify).join('\n') + '\n');
      await assert.rejects(invoke('--rescore', badSource, '--output', badOutput), error => error.code === 1 && /状态|接受/u.test(error.stderr));
      await assert.rejects(readFile(badOutput), error => error.code === 'ENOENT');
    }
    const eventTypes = ['case_started', 'search_result', 'analysis_requested', 'analysis_accepted', 'analysis_terminal', 'result'];
    for (const [mutate, reason] of [
      [rows => [rows[0], rows[1], ...eventTypes.flatMap(type => rows.filter(row => row.type === type)), ...rows.slice(-2)], /串行|事件/u],
      [rows => { rows.splice(-1, 0, { type: 'unexpected_event' }); return rows; }, /事件/u],
      [rows => { rows[0].context.model = {}; return rows; }, /模型/u],
      [rows => { const result = rows.find(row => row.type === 'result'); result.analysis.status = 'no_matches'; result.analysis.sources = []; return rows; }, /用量/u],
    ]) {
      const invalidRows = mutate(structuredClone(rows)); invalidRows[0].phase = 'candidate';
      await writeFile(badSource, invalidRows.map(JSON.stringify).join('\n') + '\n');
      await assert.rejects(invoke('--rescore', badSource, '--output', badOutput), error => error.code === 1 && reason.test(error.stderr));
      await assert.rejects(readFile(badOutput), error => error.code === 'ENOENT');
      await assert.rejects(invoke('--compare', destination, badSource), error => error.code === 1 && /不可比较/u.test(error.stdout) && reason.test(error.stdout));
      const report = await invoke('--report', badSource);
      assert.match(report.stdout, /失败或不完整；不可比较/u); assert.match(report.stdout, reason);
    }
    const otherCases = join(directory, 'other-cases.json'); await writeFile(otherCases, JSON.stringify({ ...dataset, description: '另一份合法题集，摘要不同' }));
    await assert.rejects(invoke('--rescore', source, '--cases', otherCases, '--output', badOutput), error => error.code === 1 && /摘要|SHA256/u.test(error.stderr));
    await assert.rejects(readFile(badOutput), error => error.code === 'ENOENT');
    assert.equal(await readFile(source, 'utf8'), text);
  } finally { await rm(directory, { recursive: true, force: true }); }
});

test('JSONL报告读取拒绝破损尾行，历史报告创建拒绝覆盖', async () => {
  const dir = await mkdtemp(join(tmpdir(), 'gozero-quality-'));
  try {
    const path = join(dir, 'partial.jsonl'); await writeFile(path, '{"type":"meta"}\n{"type":');
    await assert.rejects(feature('readRecords')(path), /JSONL|损坏/);
    const output = join(dir, 'report.jsonl'), handle = await feature('openReport')(output);
    await handle.append({ type: 'error', retry: false }); await handle.close();
    await assert.rejects(feature('openReport')(output), /存在|覆盖/);
    assert.equal(JSON.parse((await readFile(output, 'utf8')).trim()).retry, false);
  } finally { await rm(dir, { recursive: true, force: true }); }
});
