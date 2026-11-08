import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { createApiClient, loadDataset, runEvaluation } from './eval-kb-quality.mjs';

const exec = promisify(execFile);
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const casesPath = resolve(root, 'ai_document/ai-rag/business-cases.json');
const provenancePath = resolve(root, 'ai_document/ai-rag/business-corpus-provenance.json');
const sha = value => createHash('sha256').update(value).digest('hex');
const revision = 'fc226fbc8d1cd196453617fa7ff2fcdb548ba945';
const originalHashes = {
  'quality-cases.json': '538f6de8322da47a77bb51db7bea76e2aaa9a3ae8de34767824a61698dcb5b4c',
  'answer-regression-cases.json': '844a999cfa03179bba7370299a6bf8e3ad2fd6dc463611c8d980b5104e586ee4',
  'recall-cases.json': 'ea15baf4bed18fcba7193ae3a7f45e078f90e924d280a8b04b22430d6e93fb50',
};

test('实际项目业务题集保留32个新问题及明确的来源与人工复核边界', async () => {
  const dataset = await loadDataset(casesPath);
  assert.equal(dataset.cases.length, 32);
  assert.equal(dataset.cases.filter(row => row.split === 'development').length, 20);
  assert.equal(dataset.cases.filter(row => row.split === 'validation').length, 12);
  assert.equal(dataset.cases.filter(row => row.answer.kind === 'answerable').length, 24);
  assert.equal(dataset.cases.filter(row => row.answer.kind === 'unanswerable').length, 8);
  assert.deepEqual([...new Set(dataset.documents.map(row => row.kb))].sort(), ['project_ai', 'project_ops']);
  assert.equal(dataset.documents.length, 8);
  assert.equal(new Set(dataset.cases.map(row => row.query)).size, 32);
  assert.ok(new Set(dataset.cases.map(row => row.category)).size >= 6);
  assert.equal(dataset.cases.filter(row => row.answer.absence_policy === 'required').length, 2);
  assert.match(dataset.description, /AI辅助/);
  assert.match(dataset.description, /不是真实用户历史问题/);
  assert.equal(dataset.annotation.human_signed, false);
  assert.equal(dataset.annotation.actual_user_history, false);
  for (const doc of dataset.documents) {
    assert.ok(dataset.cases.some(row => row.expected_documents.includes(doc.key)), doc.key);
    const bytes = await readFile(resolve(root, doc.source));
    assert.ok(bytes.length > 0 && bytes.length <= 65536, doc.key);
    assert.ok(!bytes.includes(13), `${doc.key}必须固定LF`);
  }
});

test('冻结摘录SHA与来源记录一致，源提交可用时逐字核对', async t => {
  const dataset = await loadDataset(casesPath);
  const provenance = JSON.parse(await readFile(provenancePath, 'utf8'));
  assert.equal(provenance.source_revision, revision);
  assert.equal(provenance.documents.length, dataset.documents.length);
  assert.equal(provenance.material_kind, 'actual_project_document_excerpts');
  let gitAvailable = true;
  try { await exec('git', ['cat-file', '-e', `${revision}^{commit}`], { cwd: root }); }
  catch { gitAvailable = false; }
  for (const doc of dataset.documents) {
    const source = provenance.documents.find(row => row.key === doc.key);
    assert.ok(source, doc.key);
    assert.equal(source.sha256, doc.sha256);
    assert.equal(source.source, doc.source);
    assert.ok(source.excerpts.length > 0);
    const parts = [];
    for (const excerpt of source.excerpts) {
      assert.match(excerpt.origin_sha256, /^[a-f0-9]{64}$/u);
      assert.ok(['ai_document/dev-guide.md', 'ai_document/api-contract.md', 'README.md'].includes(excerpt.origin_path));
      assert.ok(excerpt.start_line > 0 && excerpt.end_line_exclusive > excerpt.start_line);
      if (!gitAvailable) continue;
      const { stdout } = await exec('git', ['show', `${revision}:${excerpt.origin_path}`], { cwd: root, encoding: 'buffer', maxBuffer: 1024 * 1024 });
      assert.equal(sha(stdout), excerpt.origin_sha256, excerpt.origin_path);
      const lines = stdout.toString('utf8').split('\n');
      const content = lines.slice(excerpt.start_line - 1, excerpt.end_line_exclusive - 1).join('\n') + '\n';
      assert.equal(sha(content), excerpt.excerpt_sha256, doc.key);
      parts.push(content);
    }
    if (gitAvailable) assert.equal(parts.join('\n'), await readFile(resolve(root, doc.source), 'utf8'), doc.key);
  }
  if (!gitAvailable) t.diagnostic('浅克隆未包含冻结源提交：已核对素材SHA，未执行Git原文逐字核对；不自动联网补取。');
});

test('默认与分组plan在阻断fetch时仍执行成功，且不更改历史题集', async () => {
  const blockFetch = 'globalThis.fetch=()=>{throw new Error("禁止业务题集plan调用HTTP")};';
  for (const [split, count] of [['all', 32], ['development', 20], ['validation', 12]]) {
    const { stdout } = await exec(process.execPath, ['--import', `data:text/javascript,${encodeURIComponent(blockFetch)}`,
      resolve(root, 'go-zero-admin/data/scripts/eval-kb-quality.mjs'), '--plan', '--cases', casesPath, '--split', split], { cwd: root });
    const plan = JSON.parse(stdout);
    assert.equal(plan.mode, 'plan');
    assert.equal(plan.http_requests, 0);
    assert.equal(plan.case_count, count);
    assert.equal(plan.max_analysis_submissions, count);
  }
  for (const [file, expected] of Object.entries(originalHashes)) {
    assert.equal(sha(await readFile(resolve(root, 'ai_document/ai-rag', file))), expected, file);
  }
});

test('新题集可走真实评测编排与全响应桩，两知识库归属和部分未知标记保持', async () => {
  const dataset = await loadDataset(casesPath);
  const content = new Map(await Promise.all(dataset.documents.map(async row => [row.key, await readFile(resolve(root, row.source), 'utf8')])));
  const kbMap = { project_ops: 701, project_ai: 702 };
  const docs = dataset.documents.map((row, index) => ({ ...row, id: 801 + index, kb_id: kbMap[row.kb], version: 1, status: 'ready', embedding_model: 'local-fixture' }));
  const tasks = new Map(), events = [];
  let networkRequests = 0, submissions = 0;
  const envelope = result => new Response(JSON.stringify({ code: 200, result }));
  const fetchImpl = async (url, options) => {
    networkRequests++;
    const parsed = new URL(url), path = parsed.pathname.replace('/v1/sys/ai', '');
    const body = options.body === undefined ? undefined : JSON.parse(options.body);
    const list = path.match(/^\/kbs\/(\d+)\/docs$/u);
    if (list) return envelope({ list: docs.filter(row => row.kb_id === Number(list[1])), total: docs.filter(row => row.kb_id === Number(list[1])).length });
    const detail = path.match(/^\/kbs\/(\d+)\/docs\/(\d+)$/u);
    if (detail) {
      const doc = docs.find(row => row.id === Number(detail[2]));
      assert.equal(doc.kb_id, Number(detail[1]));
      return envelope({ doc, content: content.get(doc.key) });
    }
    const hits = item => item.expected_documents.map((key, index) => {
      const doc = docs.find(row => row.key === key);
      return { chunk_id: 901 + doc.id, kb_id: doc.kb_id, doc_id: doc.id, doc_title: doc.title, snippet: content.get(key), score: 0.9 - index / 100 };
    });
    const search = path.match(/^\/kbs\/(\d+)\/search$/u);
    if (search) {
      const item = dataset.cases.find(row => row.query === body.query);
      assert.equal(kbMap[item.kb], Number(search[1]));
      return envelope({ list: hits(item), elapsed_ms: 2 });
    }
    if (path === '/tasks/run') {
      const input = JSON.parse(body.definition).steps[0].input, item = dataset.cases.find(row => row.query === input.query);
      assert.ok(item);
      assert.equal(input.kb_id, kbMap[item.kb]);
      assert.ok(events.some(row => row.type === 'analysis_requested' && row.request_id === body.request_id));
      const sources = hits(item).map((row, index) => ({ ...row, index: index + 1 }));
      const answerable = item.answer.kind === 'answerable';
      const analysis = { version: 1, type: 'kb_analysis', status: answerable ? 'completed' : 'no_matches',
        analysis: answerable ? item.answer.required_groups.map(group => group[0]).join('；') + sources.map(row => ` [${row.index}]`).join('')
          + (item.answer.absence_policy === 'required' ? '；本次片段未能确定问题中的实际实例值。' : '') : '本次片段资料不足，无法回答。', sources,
        usage: { scope: 'chat', prompt_tokens: answerable ? 10 : 0, completion_tokens: answerable ? 5 : 0, total_tokens: answerable ? 15 : 0, complete: answerable } };
      const id = 1001 + submissions++;
      tasks.set(id, { id, workflow_id: 0, request_id: body.request_id, status: 'succeeded', steps: [{ index: 0, tool: 'kb_analyze', status: 'succeeded', output: JSON.stringify(analysis) }] });
      return envelope({ task_id: id, status: 'running' });
    }
    const task = path.match(/^\/tasks\/(\d+)$/u);
    if (task) return envelope({ task: tasks.get(Number(task[1])) });
    throw new Error(`未知本地响应桩路径：${path}`);
  };
  const client = createApiClient({ token: 'local-fixture', fetchImpl });
  const result = await runEvaluation({ dataset, kbMap, client, onProgress: async event => events.push(event), pollIntervalMs: 0 });
  assert.equal(result.results.length, 32);
  assert.equal(submissions, 32);
  assert.equal(result.summary.positive_count, 24);
  assert.equal(result.summary.negative_count, 8);
  assert.equal(result.summary.answer_rule_screen.flagged_count, 0);
  assert.equal(result.summary.calls.known_chat_calls, 24);
  assert.equal(networkRequests, client.counts.total);
  assert.equal(events.filter(row => row.type === 'corpus_verified').length, 1);
});
