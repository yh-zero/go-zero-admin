import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { createApiClient, evaluateRetrieval, loadDataset, runEvaluation, screenAnswer } from './eval-kb-quality.mjs';

const exec = promisify(execFile);
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const casesPath = resolve(root, 'ai_document/ai-rag/business-failure-regression-cases.json');
const fixturesPath = resolve(root, 'ai_document/ai-rag/business-failure-fixtures.json');
const sha = value => createHash('sha256').update(value).digest('hex');

test('新增14题为独立开发回归，保留旧题与原资料，不冒充新保留集', async () => {
  assert.ok(existsSync(casesPath), '缺少针对178/190的独立失败回归题集');
  const dataset = await loadDataset(casesPath);
  const original = JSON.parse(await readFile(resolve(root, 'ai_document/ai-rag/business-cases.json'), 'utf8'));
  assert.equal(dataset.cases.length, 14);
  assert.ok(dataset.cases.every(item => item.split === 'development'));
  assert.equal(dataset.cases.filter(item => item.answer.kind === 'answerable').length, 13);
  assert.equal(dataset.cases.filter(item => item.answer.kind === 'unanswerable').length, 1);
  assert.equal(dataset.cases.filter(item => item.answer.absence_policy === 'forbidden').length, 11);
  assert.equal(dataset.cases.filter(item => item.answer.absence_policy === 'required').length, 3);
  assert.deepEqual(dataset.documents, original.documents);
  assert.equal(dataset.annotation.human_signed, false);
  assert.equal(dataset.annotation.actual_user_history, false);
  assert.equal(dataset.annotation.independent_holdout, false);
  for (const sourceId of ['business-dev-access-instruction', 'business-val-model-access']) {
    const source = original.cases.find(item => item.id === sourceId);
    assert.equal(dataset.cases.filter(item => item.query === source.query).length, 2, sourceId);
  }
  assert.ok(dataset.cases.some(item => item.expected_documents.length === 3));
  const quotedUnknown = dataset.cases.find(item => item.id === 'business-failure-quoted-request-unknown');
  assert.equal(quotedUnknown.answer.kind, 'unanswerable');
  assert.match(quotedUnknown.query, /请回答“.+？”。/u);
  const wholeQuote = dataset.cases.find(item => item.id === 'business-failure-whole-quoted-question');
  assert.ok(wholeQuote.query.startsWith('“') && wholeQuote.query.endsWith('？”'));
  for (const [name, hash] of Object.entries({
    'quality-cases.json': '538f6de8322da47a77bb51db7bea76e2aaa9a3ae8de34767824a61698dcb5b4c',
    'answer-regression-cases.json': '844a999cfa03179bba7370299a6bf8e3ad2fd6dc463611c8d980b5104e586ee4',
    'business-cases.json': 'f3a4016eabc10a974e3acffc6d7347477976d00923c196c608de55a6db99538b',
  })) assert.equal(sha(await readFile(resolve(root, 'ai_document/ai-rag', name))), hash, name);
});

test('两个实测坏答保留实际来源和指纹，旧遗漏可由新规则与证据发现', async t => {
  const dataset = await loadDataset(casesPath);
  const fixtures = JSON.parse(await readFile(fixturesPath, 'utf8'));
  assert.equal(fixtures.version, 1);
  assert.equal(fixtures.examples.length, 2);
  assert.equal(fixtures.provenance.regression_dataset_sha256, sha(await readFile(casesPath)));
  assert.equal(fixtures.provenance.base_dataset_sha256, 'f3a4016eabc10a974e3acffc6d7347477976d00923c196c608de55a6db99538b');
  assert.equal(fixtures.provenance.source_log_sha256, '57571c3b7baa28f2d0e23ec5e5714edf380c7f629b15559a80810a26ca41fe7c');
  assert.equal(fixtures.provenance.baseline_binary_sha256, '86698e4d8a9d8d0cb93cf706fed7317e7c136fa8e6673f3042d75d54056551c2');
  assert.equal(fixtures.provenance.raw_provider_structure_observed, false);
  const binding = new Map(fixtures.bound_documents.map(row => [row.key, row]));
  const known = new Map(fixtures.bound_documents.map(row => [row.doc_id, { id: row.doc_id, title: row.title, status: row.status }]));
  for (const doc of dataset.documents) {
    assert.equal(binding.get(doc.key).content_sha256, doc.sha256);
    assert.equal(binding.get(doc.key).title, doc.title);
  }
  for (const example of fixtures.examples) {
    assert.equal(sha(JSON.stringify(example.observed)), example.observed_sha256);
    assert.equal(example.review.human_signed, false);
    const item = dataset.cases.find(row => row.id === example.linked_regression_case_ids[0]);
    assert.equal(item.query, example.observed.query);
    const screen = screenAnswer(item, example.observed.analysis, example.observed.kb_id, known, binding);
    assert.ok(screen.reasons.includes('additional_absence_claim_needs_review'));
    if (example.observed.task_id === 178) {
      assert.equal(example.observed.answer_screen.flagged, false, '保存旧规则没有发现该歧义的事实');
      assert.ok(!screen.reasons.includes('missing_required_facts'));
      assert.equal(example.review.expected_missing_documents.length, 0);
    } else {
      assert.equal(example.observed.task_id, 190);
      const missing = example.review.expected_missing_documents[0];
      assert.equal(missing.key, 'ai-access');
      assert.equal(missing.actual_source_index, null);
      assert.equal(missing.doc_id, binding.get('ai-access').doc_id);
      assert.ok(!example.observed.analysis.sources.some(source => source.doc_id === missing.doc_id));
      assert.ok(!example.observed.list.some(source => source.doc_id === missing.doc_id));
      assert.ok(screen.reasons.includes('missing_required_facts'));
      assert.ok(screen.reasons.includes('missing_expected_citations'));
      assert.ok(screen.reasons.includes('cited_evidence_incomplete'));
      const retrieval = evaluateRetrieval(item, example.observed.list, binding);
      assert.equal(retrieval.recall_at[5], 0.5);
      assert.equal(retrieval.evidence_at[5].complete, false);
    }
  }
  function inspect(value) {
    if (!value || typeof value !== 'object') return;
    for (const [key, child] of Object.entries(value)) {
      assert.ok(!['access_token', 'refresh_token', 'session_id', 'api_key', 'password', 'authorization'].includes(key.toLowerCase()), '夹具不保存认证字段');
      inspect(child);
    }
  }
  inspect(fixtures);
  const originalLog = resolve(root, fixtures.provenance.source_log);
  if (existsSync(originalLog)) {
    const bytes = await readFile(originalLog);
    assert.equal(sha(bytes), fixtures.provenance.source_log_sha256);
    const records = bytes.toString('utf8').split('\n').filter(Boolean).map(JSON.parse);
    for (const example of fixtures.examples) {
      const row = records.find(row => row.type === 'result' && row.task_id === example.observed.task_id);
      assert.equal(sha(JSON.stringify(row)), example.source_record_sha256);
      for (const [key, value] of Object.entries(example.observed)) assert.deepEqual(value, row[key], key);
    }
  } else t.diagnostic('当前环境没有忽略目录中的原始日志，仅验证自包含夹具与素材指纹，不声称重新读取原始运行。');
});

test('14题及定向选择的plan阻断fetch仍为0 HTTP，validation没有伪造新保留题', async () => {
  const blockFetch = 'globalThis.fetch=()=>{throw new Error("失败回归plan不允许调用HTTP")};';
  const cli = resolve(root, 'go-zero-admin/data/scripts/eval-kb-quality.mjs');
  for (const [args, count] of [[[], 14], [['--split', 'development'], 14], [['--case-ids', 'business-failure-access-repeat-1,business-failure-quoted-request-unknown'], 2]]) {
    const { stdout } = await exec(process.execPath, ['--import', `data:text/javascript,${encodeURIComponent(blockFetch)}`, cli, '--plan', '--cases', casesPath, ...args], { cwd: root });
    const plan = JSON.parse(stdout);
    assert.equal(plan.mode, 'plan');
    assert.equal(plan.http_requests, 0);
    assert.equal(plan.case_count, count);
    assert.equal(plan.by_split.validation, 0);
    assert.equal(plan.max_analysis_submissions, count);
    assert.equal(plan.max_query_embedding_calls, count * 2);
  }
  await assert.rejects(exec(process.execPath, ['--import', `data:text/javascript,${encodeURIComponent(blockFetch)}`, cli, '--plan', '--cases', casesPath, '--split', 'validation'], { cwd: root }), /没有|为空|选题|题目|题集/u);
});

test('14题走实际评测编排与全响应桩，三文档、引号问项和不足目标保持', async () => {
  const dataset = await loadDataset(casesPath);
  const content = new Map(await Promise.all(dataset.documents.map(async row => [row.key, await readFile(resolve(root, row.source), 'utf8')])));
  const kbMap = { project_ops: 701, project_ai: 702 };
  const docs = dataset.documents.map((row, index) => ({ ...row, id: 801 + index, kb_id: kbMap[row.kb], version: 1, status: 'ready', embedding_model: 'local-fixture' }));
  const tasks = new Map(), events = [];
  let requests = 0, submissions = 0;
  const envelope = result => new Response(JSON.stringify({ code: 200, result }));
  const hits = item => item.expected_documents.map((key, index) => {
    const doc = docs.find(row => row.key === key);
    return { chunk_id: 901 + doc.id, kb_id: doc.kb_id, doc_id: doc.id, doc_title: doc.title, snippet: content.get(key), score: 0.9 - index / 100 };
  });
  const fetchImpl = async (url, options) => {
    requests++;
    const parsed = new URL(url), path = parsed.pathname.replace('/v1/sys/ai', '');
    const body = options.body === undefined ? undefined : JSON.parse(options.body);
    const list = path.match(/^\/kbs\/(\d+)\/docs$/u);
    if (list) {
      const own = docs.filter(row => row.kb_id === Number(list[1]));
      return envelope({ list: own, total: own.length });
    }
    const detail = path.match(/^\/kbs\/(\d+)\/docs\/(\d+)$/u);
    if (detail) {
      const doc = docs.find(row => row.id === Number(detail[2]));
      assert.equal(doc.kb_id, Number(detail[1]));
      return envelope({ doc, content: content.get(doc.key) });
    }
    const search = path.match(/^\/kbs\/(\d+)\/search$/u);
    if (search) {
      const item = dataset.cases.find(row => row.query === body.query);
      assert.equal(kbMap[item.kb], Number(search[1]));
      return envelope({ list: hits(item), elapsed_ms: 2 });
    }
    if (path === '/tasks/run') {
      const input = JSON.parse(body.definition).steps[0].input;
      const item = dataset.cases.find(row => row.query === input.query);
      assert.equal(input.kb_id, kbMap[item.kb]);
      assert.ok(events.some(row => row.type === 'analysis_requested' && row.request_id === body.request_id));
      const sources = hits(item).map((row, index) => ({ ...row, index: index + 1 }));
      const answerable = item.answer.kind === 'answerable';
      const analysis = { version: 1, type: 'kb_analysis', status: answerable ? 'completed' : 'no_matches',
        analysis: answerable ? item.answer.required_groups.map(group => group[0]).join('；') + sources.map(row => ` [${row.index}]`).join('')
          + (item.answer.absence_policy === 'required' ? '；本次片段未能确定问题中明确询问的实例实际ID。' : '') : '本次片段资料不足，无法回答该分析目标。',
        sources, usage: { scope: 'chat', prompt_tokens: answerable ? 10 : 0, completion_tokens: answerable ? 5 : 0, total_tokens: answerable ? 15 : 0, complete: answerable } };
      const id = 1001 + submissions++;
      tasks.set(id, { id, workflow_id: 0, request_id: body.request_id, status: 'succeeded', steps: [{ index: 0, tool: 'kb_analyze', status: 'succeeded', output: JSON.stringify(analysis) }] });
      return envelope({ task_id: id, status: 'running' });
    }
    const task = path.match(/^\/tasks\/(\d+)$/u);
    if (task) return envelope({ task: tasks.get(Number(task[1])) });
    throw new Error(`未知本地响应桩路径：${path}`);
  };
  const client = createApiClient({ token: 'local-regression-fixture', fetchImpl });
  const result = await runEvaluation({ dataset, kbMap, client, onProgress: async event => events.push(event), pollIntervalMs: 0 });
  assert.equal(result.results.length, 14);
  assert.equal(submissions, 14);
  assert.equal(new Set(events.filter(row => row.type === 'analysis_requested').map(row => row.request_id)).size, 14);
  assert.equal(result.summary.positive_count, 13);
  assert.equal(result.summary.negative_count, 1);
  assert.equal(result.summary.answer_rule_screen.flagged_count, 0);
  assert.equal(result.summary.calls.known_chat_calls, 13);
  assert.equal(requests, client.counts.total);
  assert.equal(events.filter(row => row.type === 'corpus_verified').length, 1);
  assert.equal(result.results.find(row => row.case_id === 'business-failure-cross-three-docs').analysis.sources.length, 3);
});
