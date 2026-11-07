import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, readFile, unlink, rmdir } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const recall = await import('./eval-kb-recall.mjs').catch(error => {
  if (error.code === 'ERR_MODULE_NOT_FOUND') return {};
  throw error;
});
function feature(name) {
  assert.equal(typeof recall[name], 'function', `缺少召回评测能力 ${name}`);
  return recall[name];
}
const source = { key: 'rules', kb: 'stocks', source: 'ai_document/ai-rag/rules.txt', title: '规则', sha256: 'a'.repeat(64) };
const positive = { id: 'rules-query', kb: 'stocks', query: '亏损时如何估值？', expected_document: 'rules', evidence_groups: [['PE为负无意义'], ['PB或PS', 'PB/PS']] };
const negative = { id: 'no-answer', kb: 'stocks', query: '发布审批人是谁？', expected_document: null, evidence_groups: [] };
const manifest = { kbs: [{ key: 'stocks', id: 7 }], documents: [{ source: source.source, title: source.title, sha256: source.sha256, kb_id: 7, id: 11 }] };
const doc = (id, title = '规则') => ({ id, kb_id: 7, title, status: 'ready' });
const hit = (chunkId, docId, snippet, score, title = '规则') => ({ chunk_id: chunkId, kb_id: 7, doc_id: docId, doc_title: title, snippet, score });
const envelope = (result, code = 200, status = 200) => new Response(JSON.stringify({ code, msg: 'fixture', result }), { status, headers: { 'Content-Type': 'application/json' } });
function fixture(search, docs = [doc(11), doc(12)]) {
  const calls = [];
  const fetchImpl = async (url, options) => {
    const parsed = new URL(url);
    calls.push({ path: parsed.pathname, method: options.method, query: parsed.search });
    assert.equal(options.headers.Authorization, 'Bearer fixture-access');
    if (parsed.pathname.endsWith('/docs')) return envelope({ list: docs, total: docs.length });
    assert.ok(parsed.pathname.endsWith('/search'), '不应调用聊天或写入文档接口');
    return search(calls.filter(call => call.path.endsWith('/search')).length, JSON.parse(options.body));
  };
  return { calls, client: feature('createApiClient')({ token: 'fixture-access', fetchImpl }) };
}

test('固定题集覆盖12个正例及2个无答案例，原文SHA256与证据标注有效', async () => {
  const dataset = await feature('loadDataset')();
  assert.equal(dataset.cases.filter(item => item.expected_document !== null).length, 12);
  assert.equal(dataset.cases.filter(item => item.expected_document === null).length, 2);
  assert.equal(dataset.documents.length, 11);
  assert.equal(new Set(dataset.cases.map(item => item.id)).size, 14);
  assert.ok(dataset.documents.every(item => /^[a-f0-9]{64}$/.test(item.sha256)));
  const options = feature('parseArgs')([]);
  assert.equal(options.run, false, '默认只能计划，不发真实请求');
  assert.equal(feature('parseArgs')(['--plan']).run, false);
  assert.throws(() => feature('parseArgs')(['--run']), /auth|认证/);
  assert.throws(() => feature('parseArgs')(['--base', 'http://example.test']), /本机/);
  assert.throws(() => feature('parseArgs')(['--output', 'outside-result.jsonl']), /tmp/);
});

test('统计使用前K片段和原始排名，只在正确文档中匹配NFKC/空白规范化后的证据', () => {
  const list = [hit(1, 12, 'PE为负无意义，PB/PS', 0.99), hit(2, 11, 'ＰＥ 为负无意义', 0.9), hit(3, 11, '改用ＰＢ ／ ＰＳ', 0.8)];
  const result = feature('evaluateCase')(positive, list, 11);
  assert.deepEqual(result.hit_at, { 1: 0, 3: 1, 5: 1 });
  assert.deepEqual(result.recall_at, result.hit_at);
  assert.equal(result.mrr_at_5, 0.5);
  assert.equal(result.evidence_at[1].coverage, 0, '其他同名文档的证据不能冒充目标');
  assert.equal(result.evidence_at[3].coverage, 1);
  assert.equal(result.evidence_at[3].complete, true);
});

test('MRR@5不计入第六个以后的目标，独立统计函数也保持截断口径', () => {
  const list = Array.from({ length: 6 }, (_, index) => hit(index + 1, index === 5 ? 11 : 12, 'PE为负无意义 PB/PS', 1 - index / 10));
  assert.equal(feature('evaluateCase')(positive, list, 11).mrr_at_5, 0);
});

test('空召回不冒充错误；无答案例单独记录候选与分数，不加入正例召回率', () => {
  const evaluate = feature('evaluateCase');
  const missed = evaluate(positive, [], 11);
  assert.equal(missed.hit_at[5], 0);
  assert.equal(missed.mrr_at_5, 0);
  assert.equal(missed.evidence_at[5].coverage, 0);
  const empty = evaluate(negative, [], null);
  assert.equal(empty.candidate_count, 0);
  assert.equal(empty.max_score, null);
  const candidates = evaluate(negative, [hit(1, 11, '无关投资规则', 0.3)], null);
  assert.equal(candidates.kind, 'negative');
  assert.equal(candidates.candidate_count, 1);
  assert.equal(candidates.max_score, 0.3);
  assert.equal('answer_correct' in candidates, false);
  const summary = feature('summarize')([{ evaluation: missed, client_elapsed_ms: 40, server_elapsed_ms: 30 }, { evaluation: candidates, client_elapsed_ms: 20, server_elapsed_ms: 10 }]);
  assert.equal(summary.positive_count, 1);
  assert.equal(summary.negative_count, 1);
  assert.equal(summary.positive.recall_at[5], 0);
  assert.equal(summary.negative.candidate_return_rate, 1);
  assert.equal(summary.latency.client.median_ms, 30);
});

test('拒绝重复chunk、顺序错乱、同分不稳定、跨库及非法片段字段', () => {
  const validate = feature('validateHits');
  const known = new Map([[11, doc(11)]]);
  const one = hit(1, 11, '资料', 0.9);
  assert.deepEqual(validate([one], 7, known), [one]);
  for (const invalid of [
    [one, { ...one }],
    [one, hit(2, 11, '资料', 0.95)],
    [hit(2, 11, '资料', 0.9), one],
    [{ ...one, kb_id: 8 }],
    [{ ...one, score: '0.9' }],
    [{ ...one, score: Infinity }],
    [{ ...one, snippet: '  ' }],
    [{ ...one, doc_title: '另一文档' }],
    [{ ...one, doc_id: 99 }],
    [{ ...one, chunk_id: Number.MAX_SAFE_INTEGER + 1 }],
    Array.from({ length: 6 }, (_, index) => ({ ...one, chunk_id: index + 1 })),
  ]) assert.throws(() => validate(invalid, 7, known), /召回|片段|排序|分块|知识库|文档|字段/);
});

test('通过manifest精确绑定素材/ID，同名其他文档不会被计为预期命中', async () => {
  const { client, calls } = fixture(() => envelope({ list: [hit(1, 12, 'PE为负无意义 PB/PS', 0.95), hit(2, 11, 'PE为负无意义 PB/PS', 0.8)], elapsed_ms: 12 }));
  const progress = [];
  const run = await feature('runEvaluation')({ dataset: { documents: [source], cases: [positive, negative] }, kbMap: { stocks: 7 }, manifest, client, onProgress: async value => progress.push(value) });
  assert.equal(run.results.length, 2);
  assert.equal(run.results[0].expected_document.doc_id, 11);
  assert.equal(run.results[0].evaluation.hit_at[1], 0);
  assert.equal(run.results[0].evaluation.hit_at[3], 1);
  assert.equal(run.results[0].evaluation.mrr_at_5, 0.5);
  assert.equal(calls.filter(call => call.path.endsWith('/search')).length, 2);
  assert.equal(calls.filter(call => call.path.endsWith('/docs')).length, 1);
  assert.equal(progress.filter(item => item.type === 'result').length, 2);
  assert.ok(progress.find(item => item.type === 'case_started'));
});

test('没有manifest时拒绝歧义标题，有manifest时拒绝来源哈希不一致，均不执行付费检索', async () => {
  const { client, calls } = fixture(() => { throw new Error('不应检索'); });
  const run = feature('runEvaluation');
  const args = { dataset: { documents: [source], cases: [positive] }, kbMap: { stocks: 7 }, client };
  await assert.rejects(run(args), /同名|唯一|歧义/);
  await assert.rejects(run({ ...args, manifest: { ...manifest, documents: [{ ...manifest.documents[0], sha256: 'b'.repeat(64) }] } }), /哈希|来源/);
  assert.equal(calls.filter(call => call.path.endsWith('/search')).length, 0);
});

test('HTTP/业务503及超时停止，不算空召回，不自动重试或继续后续题', async () => {
  for (const response of [() => envelope({}, 503), () => envelope({}, 200, 503), () => { throw new DOMException('private-network-details', 'TimeoutError'); }]) {
    const { client, calls } = fixture(response, [doc(11)]);
    const progress = [];
    await assert.rejects(feature('runEvaluation')({ dataset: { documents: [source], cases: [positive, negative] }, kbMap: { stocks: 7 }, manifest, client, onProgress: async value => progress.push(value) }), /503|超时/);
    assert.equal(calls.filter(call => call.path.endsWith('/search')).length, 1);
    assert.equal(progress.filter(item => item.type === 'result').length, 0);
  }
});

test('第二题跨库响应中止时，已成功第一题结果仍逐题保留', async () => {
  const { client, calls } = fixture(index => envelope({ list: [index === 1 ? hit(1, 11, 'PE为负无意义 PB/PS', 0.9) : { ...hit(2, 11, '不应跨库', 0.9), kb_id: 8 }], elapsed_ms: 1 }), [doc(11)]);
  const progress = [];
  await assert.rejects(feature('runEvaluation')({ dataset: { documents: [source], cases: [positive, negative, { ...negative, id: 'third' }] }, kbMap: { stocks: 7 }, manifest, client, onProgress: async value => progress.push(value) }), /知识库/);
  assert.equal(calls.filter(call => call.path.endsWith('/search')).length, 2);
  assert.equal(progress.filter(item => item.type === 'result').length, 1);
  assert.equal(progress.find(item => item.type === 'result').case_id, positive.id);
});

test('结果文件追加持久化，已存在的历史文件拒绝覆盖', async () => {
  const open = feature('openReport');
  const directory = await mkdtemp(join(tmpdir(), 'gozero-recall-eval-'));
  const filename = join(directory, 'result.jsonl');
  try {
    const report = await open(filename);
    try {
      await report.append({ type: 'result', case_id: 'one' });
      await report.append({ type: 'error', message: '请求停止' });
    } finally { await report.close(); }
    await assert.rejects(open(filename), /存在|覆盖/);
    const rows = (await readFile(filename, 'utf8')).trim().split('\n').map(JSON.parse);
    assert.deepEqual(rows.map(row => row.type), ['result', 'error']);
  } finally {
    await unlink(filename).catch(error => { if (error.code !== 'ENOENT') throw error; });
    await rmdir(directory);
  }
});
