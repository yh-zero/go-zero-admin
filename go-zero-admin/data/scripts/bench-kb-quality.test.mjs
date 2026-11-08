import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtemp, readFile, rm, writeFile, symlink, realpath, stat } from 'node:fs/promises';
import { resolve, join, dirname, basename } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { createApiClient } from './eval-kb-quality.mjs';

const bench = await import('./bench-kb-quality.mjs').catch(error => {
  if (error.code === 'ERR_MODULE_NOT_FOUND') return {};
  throw error;
});
const feature = name => { assert.equal(typeof bench[name], 'function', `缺少小规模性能基线能力 ${name}`); return bench[name]; };
const sha = value => createHash('sha256').update(value).digest('hex');
const text = '事实甲。';
const document = { key: 'a', kb: 'project', title: '项目资料', source: 'ai_document/ai-rag/project.md', sha256: sha(text) };
const item = { id: 'business-dev-example', split: 'development', category: 'direct', kb: 'project', query: '事实是什么？',
  expected_documents: ['a'], evidence: [{ document: 'a', groups: [['事实甲']] }],
  answer: { kind: 'answerable', required_groups: [['事实甲']], forbidden_phrases: [] }, rationale: '桩资料' };
const context = () => ({ binary_sha: 'a'.repeat(64), model: { provider: 'fixture', name: 'fixture' },
  embedding: { provider: 'fixture', model: 'fixture', dimension: 3 },
  config: { max_context_tokens: 32000, analysis_answer_token_cap: 2048, recall_top_k: 20, return_top_k: 5 } });
const source = { chunk_id: 101, kb_id: 7, doc_id: 11, doc_title: '项目资料', snippet: text, score: 0.9 };
const analysis = { version: 1, type: 'kb_analysis', status: 'completed', analysis: '事实甲 [1]',
  sources: [{ ...source, index: 1 }], usage: { scope: 'chat', prompt_tokens: 10, completion_tokens: 5, total_tokens: 15, complete: true } };
const envelope = result => new Response(JSON.stringify({ code: 200, result }), { status: 200 });
const delay = ms => new Promise(resolvePromise => setTimeout(resolvePromise, ms));
const execFileAsync = promisify(execFile);
async function cleanupTemporaryFolder(folder, expectedRoot, prefix) {
  const actual = await realpath(folder), root = await realpath(expectedRoot);
  assert.equal(dirname(actual), root, '清理只能操作本轮指定根目录下的独立临时子目录');
  assert.ok(basename(actual).startsWith(prefix));
  await rm(folder, { recursive: true });
  await assert.rejects(stat(folder), error => error.code === 'ENOENT');
}

function fixture({ failure, duplicateTask = false, drift = false, postflightDrift = false, analysisOutput } = {}) {
  const logs = new Map(), manifest = [], http = [], requested = new Set(), accepted = new Map();
  const documentLists = new Map();
  let nextTask = 100, active = 0, peak = 0;
  const matches = (lane, stage) => failure?.concurrency === lane.concurrency && failure?.lane === lane.lane && failure?.stage === stage;
  const args = {
    dataset: { version: 1, documents: [document], cases: [item] }, datasetSha256: 'b'.repeat(64), caseID: item.id,
    kbMap: { project: 7 }, context: context(), pollIntervalMs: 0,
    onManifest: async event => manifest.push(event),
    openLane: async lane => {
      const events = [], key = `${lane.concurrency}:${lane.lane}`;
      logs.set(key, events);
      return { path: `tmp/bench-fixture/c${lane.concurrency}-l${lane.lane}.jsonl`, append: async event => {
        events.push(event);
        if (event.type === 'analysis_requested') requested.add(event.request_id);
      }, close: async () => {} };
    },
    createClient: lane => createApiClient({ token: 'fixture-access', fetchImpl: async (url, options) => {
      const parsed = new URL(url), path = parsed.pathname.replace('/v1/sys/ai', '');
      const body = options.body ? JSON.parse(options.body) : undefined;
      http.push({ ...lane, path, method: options.method, body });
      const key = `${lane.concurrency}:${lane.lane}`;
      if (path === '/kbs/7/docs') documentLists.set(key, (documentLists.get(key) ?? 0) + 1);
      const doc = { id: 11, kb_id: 7, title: '项目资料', status: 'ready', version: (drift && lane.concurrency > 1) || (postflightDrift && documentLists.get(key) > 1) ? 2 : 1, embedding_model: 'fixture-embed' };
      if (path === '/kbs/7/docs') {
        if (matches(lane, 'preflight')) throw new Error('桩预检故障');
        if (matches(lane, 'postflight') && documentLists.get(key) > 1) throw new Error('桩结束复核故障');
        return envelope({ list: [doc], total: 1 });
      }
      if (path === '/kbs/7/docs/11') return envelope({ doc, content: text });
      if (path === '/kbs/7/search') return envelope({ list: [source], elapsed_ms: 2 });
      if (path === '/tasks/run') {
        assert.ok(requested.has(body.request_id), '原UUID和载荷须先写入lane日志');
        if (matches(lane, 'submit')) throw new DOMException('fixture-timeout', 'TimeoutError');
        const id = duplicateTask && lane.concurrency === 3 ? 200 : nextTask++;
        accepted.set(id, { request_id: body.request_id, lane });
        active++; peak = Math.max(peak, active);
        return envelope({ task_id: id, status: 'running' });
      }
      if (/^\/tasks\/\d+$/u.test(path)) {
        await delay(8);
        const id = Number(path.split('/').at(-1)), submitted = accepted.get(id);
        active--;
        return envelope({ task: { id, workflow_id: 0, request_id: submitted.request_id,
          status: matches(lane, 'terminal') ? 'failed' : 'succeeded',
          steps: [{ index: 0, tool: 'kb_analyze', status: 'succeeded', output: JSON.stringify(analysisOutput ?? analysis) }] } });
      }
      throw new Error(`未知桩路径 ${path}`);
    } }),
  };
  return { args, logs, manifest, http, peak: () => peak };
}

test('默认plan为0HTTP计划，run必须显式指定单一题目和本地输入，不接受改并发等级', () => {
  const parse = feature('parseBenchArgs');
  assert.equal(parse([]).run, false);
  assert.throws(() => parse(['--run']), /run|认证|case-id/u);
  assert.throws(() => parse(['--base', 'https://example.test']), /本机/u);
  assert.throws(() => parse(['--levels', '100']), /参数/u);
  assert.throws(() => parse(['--case-id', 'a,b']), /case-id/u);
  assert.throws(() => parse(['--output', 'elsewhere.jsonl']), /tmp/u);
  assert.throws(() => parse(['--run', '--plan']), /混用|互斥/u);
});

test('真实CLI默认plan在禁用fetch的进程内成功，准确展示9次分析和18次逻辑Embedding预算', async () => {
  const filename = fileURLToPath(new URL('./bench-kb-quality.mjs', import.meta.url));
  const { stdout } = await execFileAsync(process.execPath, ['--import', 'data:text/javascript,globalThis.fetch=()=>{throw new Error("HTTP_DISABLED")}', filename]);
  const result = JSON.parse(stdout);
  assert.equal(result.mode, 'plan');
  assert.equal(result.http_requests, 0);
  assert.deepEqual(result.levels, [1, 3, 5]);
  assert.equal(result.max_analysis_submissions, 9);
  assert.equal(result.max_logical_query_embeddings, 18);
  assert.ok(result.eligible_development_cases.length);
});

test('1/3/5波次固定同一开发题，共9次分析，峰值5且每lane保留原v1串行日志', async () => {
  const f = fixture(), result = await feature('runBenchmark')(f.args);
  assert.equal(result.completed, true);
  assert.deepEqual(result.waves.map(wave => wave.target_concurrency), [1, 3, 5]);
  assert.deepEqual(result.waves.map(wave => wave.latency.analysis_submit_to_terminal.n), [1, 3, 5]);
  assert.deepEqual(result.waves.map(wave => wave.observed_active_lanes_peak), [1, 3, 5]);
  assert.equal(f.peak(), 5);
  assert.equal(result.summary.analysis_submissions, 9);
  assert.equal(result.summary.search_requests, 9);
  assert.equal(result.summary.metadata_http_requests, 36);
  assert.equal(result.summary.http.total, 63);
  assert.equal(result.summary.known_chat_total_tokens, 135);
  assert.equal(result.summary.unique_cases, 1);
  const requests = f.http.filter(call => call.path === '/tasks/run');
  assert.equal(new Set(requests.map(call => call.body.request_id)).size, 9);
  const taskIDs = result.waves.flatMap(wave => wave.lanes.map(lane => lane.task_id));
  assert.equal(new Set(taskIDs).size, 9);
  for (const events of f.logs.values()) {
    assert.deepEqual(events.map(event => event.type), ['meta', 'bindings', 'case_started', 'search_result', 'analysis_requested', 'analysis_accepted', 'analysis_terminal', 'result', 'corpus_verified', 'summary']);
    assert.equal(events[0].version, 1);
    assert.deepEqual(events[0].case_ids, [item.id]);
    assert.equal(events.at(-1).completed, true);
  }
  for (const wave of result.waves) for (const lane of wave.lanes) {
    const events = f.logs.get(`${wave.target_concurrency}:${lane.lane}`);
    assert.equal(lane.log_sha256, sha(events.map(event => JSON.stringify(event) + '\n').join('')));
  }
  assert.equal(f.manifest[0].type, 'benchmark_meta');
  assert.equal(f.manifest.at(-1).type, 'benchmark_summary');
  assert.match(feature('renderBenchmarkReport')(result), /不代表生产容量/u);
});

test('提交超时保留原UUID与载荷，不自动重放；等待同波次已开始lane结束并停止下一波', async () => {
  const f = fixture({ failure: { concurrency: 3, lane: 2, stage: 'submit' } });
  const result = await feature('runBenchmark')(f.args);
  assert.equal(result.completed, false);
  assert.equal(result.waves.length, 2);
  assert.equal(result.summary.analysis_submissions, 4);
  assert.equal(result.summary.failed_lanes, 1);
  assert.equal(result.summary.succeeded_lanes, 3);
  const failed = result.waves[1].lanes.find(lane => lane.lane === 2);
  assert.ok(failed.request_id);
  assert.equal(failed.task_id, null);
  assert.equal(failed.outcome, 'unknown');
  assert.equal(failed.retry, false);
  const events = f.logs.get('3:2');
  assert.equal(events.filter(event => event.type === 'analysis_requested').length, 1);
  assert.equal(events.at(-1).type, 'error');
  assert.equal(f.http.filter(call => call.path === '/tasks/run' && call.body.request_id === failed.request_id).length, 1);
  assert.equal(f.http.some(call => call.concurrency === 5), false);
});

test('失败终态保留taskID，不计成功/空召回/零费用且其他lane照常收束', async () => {
  const f = fixture({ failure: { concurrency: 3, lane: 1, stage: 'terminal' } });
  const result = await feature('runBenchmark')(f.args), failed = result.waves[1].lanes.find(lane => lane.lane === 1);
  assert.equal(failed.status, 'failed');
  assert.equal(failed.outcome, 'failed');
  assert.ok(Number.isSafeInteger(failed.task_id));
  assert.equal(result.summary.known_chat_total_tokens, 45);
  assert.equal(result.summary.unknown_usage_lanes, 1);
  assert.equal(result.waves[1].latency.analysis_submit_to_terminal.n, 2);
});

test('有效结果已观察后结束语料GET失败仍保留已知调用和tokens，证据失败不混入成功耗时', async () => {
  const f = fixture({ failure: { concurrency: 1, lane: 1, stage: 'postflight' } });
  const result = await feature('runBenchmark')(f.args), lane = result.waves[0].lanes[0];
  assert.equal(result.completed, false);
  assert.equal(result.waves.length, 1);
  assert.equal(lane.status, 'succeeded');
  assert.equal(lane.outcome, 'succeeded');
  assert.equal(lane.evidence_status, 'invalid');
  assert.equal(lane.analysis_observed, true);
  assert.deepEqual(lane.usage, analysis.usage);
  assert.equal(result.summary.known_chat_calls, 1);
  assert.equal(result.summary.known_chat_total_tokens, 15);
  assert.equal(result.summary.unknown_usage_lanes, 0);
  assert.equal(result.summary.observed_analyses, 1);
  assert.equal(result.summary.succeeded_lanes, 0);
  assert.equal(result.waves[0].latency.analysis_submit_to_terminal.n, 0);
  assert.equal(f.http.filter(call => call.path === '/tasks/run').length, 1);
  assert.equal(f.logs.get('1:1').find(event => event.type === 'result').analysis.usage.total_tokens, 15);
});

test('结果之后语料变化、result落盘失败或close失败分别保留已知用量并停止后续波次', async () => {
  for (const mode of ['corpus', 'result-log', 'close']) {
    const f = fixture({ postflightDrift: mode === 'corpus' }), openLane = f.args.openLane;
    if (mode !== 'corpus') f.args.openLane = async descriptor => {
      const log = await openLane(descriptor);
      return { ...log, append: async event => { if (mode === 'result-log' && event.type === 'result') throw new Error('桩result落盘失败'); await log.append(event); },
        close: async () => { if (mode === 'close') throw new Error('桩close失败'); await log.close(); } };
    };
    const result = await feature('runBenchmark')(f.args), lane = result.waves[0].lanes[0];
    assert.equal(result.completed, false, mode);
    assert.equal(result.waves.length, 1, mode);
    assert.equal(lane.outcome, 'succeeded', mode);
    assert.equal(lane.evidence_status, 'invalid', mode);
    assert.equal(lane.usage.total_tokens, 15, mode);
    assert.equal(result.summary.known_chat_total_tokens, 15, mode);
    assert.equal(result.summary.unknown_usage_lanes, 0, mode);
    assert.equal(result.waves[0].latency.analysis_submit_to_terminal.n, 0, mode);
    assert.equal(f.http.filter(call => call.path === '/tasks/run').length, 1, mode);
    if (mode !== 'corpus') { assert.equal(lane.log_integrity, false, mode); assert.equal(lane.log_sha256, null, mode); }
  }
});

test('已知但不完整的聊天usage保留tokens并标未知，明确no_matches不冒充未知聊天调用', async () => {
  const partial = fixture({ analysisOutput: { ...analysis, usage: { ...analysis.usage, complete: false } } });
  const result = await feature('runBenchmark')(partial.args);
  assert.equal(result.completed, true);
  assert.equal(result.summary.known_chat_total_tokens, 135);
  assert.equal(result.summary.unknown_usage_lanes, 9);
  const empty = fixture({ analysisOutput: { ...analysis, status: 'no_matches', analysis: '本次资料不足。', sources: [],
    usage: { scope: 'chat', prompt_tokens: 0, completion_tokens: 0, total_tokens: 0, complete: false } } });
  const noMatches = await feature('runBenchmark')(empty.args);
  assert.equal(noMatches.completed, true);
  assert.equal(noMatches.summary.known_chat_calls, 0);
  assert.equal(noMatches.summary.known_chat_total_tokens, 0);
  assert.equal(noMatches.summary.unknown_usage_lanes, 0);
});

test('语料预检失败会释放同波次等待门，未放行search/分析且不死锁', async () => {
  const f = fixture({ failure: { concurrency: 3, lane: 2, stage: 'preflight' } });
  const result = await feature('runBenchmark')(f.args);
  assert.equal(result.completed, false);
  assert.equal(result.waves.length, 2);
  assert.equal(f.http.filter(call => call.concurrency === 3 && call.method === 'POST').length, 0);
  assert.equal(result.summary.analysis_submissions, 1);
  assert.equal(result.summary.unknown_usage_lanes, 0);
});

test('不同lane收到同一taskID会停止基线，保留原接收记录，不把重复任务算独立样本', async () => {
  const f = fixture({ duplicateTask: true }), result = await feature('runBenchmark')(f.args);
  assert.equal(result.completed, false);
  assert.equal(result.waves.length, 2);
  assert.ok(result.waves[1].lanes.some(lane => /重复/u.test(lane.error ?? '')));
  assert.ok([...f.logs.values()].flat().some(event => event.type === 'analysis_accepted' && event.task_id === 200));
  assert.ok(result.summary.unique_task_ids < result.summary.accepted_task_responses);
});

test('lane日志持续写失败也等待其他已提交工作收束，manifest保留原ID并标记日志不可校验', async () => {
  const f = fixture(), openLane = f.args.openLane;
  f.args.openLane = async descriptor => {
    const log = await openLane(descriptor); let broken = false;
    return { ...log, append: async event => {
      if (descriptor.concurrency === 3 && descriptor.lane === 2 && event.type === 'analysis_accepted') broken = true;
      if (broken) throw new Error('桩磁盘写失败');
      await log.append(event);
    } };
  };
  const result = await feature('runBenchmark')(f.args), failed = result.waves[1].lanes.find(lane => lane.lane === 2);
  assert.equal(result.completed, false);
  assert.ok(failed.request_id);
  assert.ok(Number.isSafeInteger(failed.task_id));
  assert.equal(failed.log_sha256, null);
  assert.equal(failed.log_integrity, false);
  assert.equal(result.summary.succeeded_lanes, 3);
  assert.equal(result.summary.analysis_submissions, 4);
  assert.equal(f.manifest.at(-1).type, 'benchmark_summary');
});

test('跨波次语料库存/版本变化在提交前拒绝，不把不同语料当并发比较', async () => {
  const f = fixture({ drift: true }), result = await feature('runBenchmark')(f.args);
  assert.equal(result.completed, false);
  assert.equal(result.summary.analysis_submissions, 1);
  assert.ok(result.waves[1].lanes.every(lane => /语料|取消/u.test(lane.error)));
});

test('仅允许有效上下文、唯一开发正例和完整题集摘要，非法输入在HTTP前失败', async () => {
  const invalid = [
    args => ({ ...args, caseID: 'missing' }),
    args => ({ ...args, datasetSha256: '' }),
    args => ({ ...args, context: {} }),
    args => ({ ...args, dataset: { ...args.dataset, cases: [{ ...item, split: 'validation' }] } }),
    args => ({ ...args, dataset: { ...args.dataset, cases: [{ ...item, answer: { ...item.answer, kind: 'unanswerable' } }] } }),
    args => ({ ...args, dataset: { ...args.dataset, cases: [item, item] } }),
  ];
  for (const mutate of invalid) {
    const f = fixture();
    await assert.rejects(feature('runBenchmark')(mutate(f.args)));
    assert.equal(f.http.length, 0);
    assert.equal(f.manifest.length, 0);
  }
});

test('独立经验分位数保留N，零成功不生成伪P95', () => {
  const percentile = feature('performanceLatency');
  assert.deepEqual(percentile([]), { n: 0, p50_ms: null, p95_ms: null, min_ms: null, max_ms: null, method: 'nearest-rank' });
  assert.equal(percentile([20]).p95_ms, 20);
  assert.equal(percentile([30, 10, 20]).p50_ms, 20);
  assert.equal(percentile([30, 10, 20]).n, 3);
  assert.throws(() => percentile([NaN]));
  assert.throws(() => percentile([-1]));
});

test('Markdown报告把题目ID和日志路径作为文字，不能变成远程图片或表格结构', async () => {
  const f = fixture(), result = await feature('runBenchmark')(f.args);
  result.meta.case_id = '![远程](https://example.test/a)';
  result.waves[0].lanes[0].log_path = 'tmp/a|b![远程](https://example.test/a).jsonl';
  const markdown = feature('renderBenchmarkReport')(result);
  assert.equal(markdown.includes('![远程](https://example.test/a)'), false);
  assert.match(markdown, /a\\\|b/u);
});

test('预留输出先核对全部路径，拒绝历史文件且不覆盖；日志和Markdown均独占创建', async t => {
  const root = resolve('tmp'), folder = await mkdtemp(join(root, 'bench-quality-test-'));
  t.diagnostic(`本轮临时目录（结束精确清理）：${folder}`);
  try {
    const output = join(folder, 'benchmark.jsonl');
    await writeFile(join(folder, 'benchmark.c3.l2.jsonl'), '原始证据');
    await assert.rejects(feature('reserveBenchmarkOutputs')(output), /存在|覆盖/u);
    await assert.rejects(readFile(output));
    assert.equal(await readFile(join(folder, 'benchmark.c3.l2.jsonl'), 'utf8'), '原始证据');
    const second = join(folder, 'second.jsonl'), reports = await feature('reserveBenchmarkOutputs')(second);
    try {
      await reports.manifest.append({ type: 'fixture' });
      await reports.markdown.write('# 桩报告\n');
      assert.equal(reports.lanes.size, 9);
    } finally { await reports.close(); }
    assert.equal(await readFile(second, 'utf8'), '{"type":"fixture"}\n');
    assert.equal(await readFile(join(folder, 'second.md'), 'utf8'), '# 桩报告\n');
  } finally { await cleanupTemporaryFolder(folder, root, 'bench-quality-test-'); }
});

test('tmp内链接若指向外部目录，拒绝创建输出', async t => {
  const folder = await mkdtemp(join(resolve('tmp'), 'bench-quality-link-test-'));
  const outside = await mkdtemp(join(tmpdir(), 'bench-quality-outside-'));
  t.diagnostic(`本轮临时目录和链接目标（结束精确清理）：${folder}；${outside}`);
  try {
    await symlink(outside, join(folder, 'redirect'), process.platform === 'win32' ? 'junction' : 'dir');
    await assert.rejects(feature('reserveBenchmarkOutputs')(join(folder, 'redirect', 'benchmark.jsonl')), /tmp|越界/u);
  } finally {
    await cleanupTemporaryFolder(folder, resolve('tmp'), 'bench-quality-link-test-');
    await cleanupTemporaryFolder(outside, tmpdir(), 'bench-quality-outside-');
  }
});
