import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { createServer } from 'vite';
import { QueryClient } from '@tanstack/react-query';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const document = { key: 'rules', kb: 'project_ops', source: 'ai_document/ai-rag/business-docs/rules.md', title: '发布规则', sha256: 'a'.repeat(64) };
const item = { id: 'case-1', kb: 'project_ops', split: 'development', category: 'risk', query: '发布前有什么要求？', rationale: '核对来源', expected_documents: ['rules'], evidence: [{ document: 'rules', groups: [['先备份']] }], answer: { kind: 'answerable', required_groups: [['先备份']], forbidden_phrases: [], absence_policy: 'forbidden' }, checks: [{ label: '先备份', kind: 'condition', required_group: 0 }] };
const dataset = { version: 1, documents: [document], cases: [item, { ...item, id: 'case-2', query: '上线前怎样处理？' }] };
const raw = JSON.stringify(dataset), sha = createHash('sha256').update(raw).digest('hex');
const source = { index: 1, chunk_id: 11, kb_id: 7, doc_id: 9, doc_title: document.title, snippet: '先备份，再发布。', score: 0.9 };
const analysis = { version: 1, type: 'kb_analysis', status: 'completed', analysis: '先备份，再发布。[1]', sources: [source], usage: { scope: 'chat', prompt_tokens: 20, completion_tokens: 10, total_tokens: 30, complete: true } };
const binding = { ...document, kb_id: 7, doc_id: 9, version: 2, status: 'ready', embedding_model: 'embedding-test', content_sha256: document.sha256 };
const context = { binary_sha: 'b'.repeat(64), model: { provider: 'local-test', name: 'test' }, embedding: { provider: 'local-test', model: 'embedding-test', dimension: 16 }, config: { max_context_tokens: 32000, analysis_answer_token_cap: 2048, recall_top_k: 20, return_top_k: 5, analysis_temperature: 0 } };
const observation = { case_id: item.id, request_id: '76d38d38-caa4-4f2f-b24b-55259fb02411', task_id: 12, status: 'succeeded', stage: 'done', query: item.query, list: [{ ...source, index: undefined }], search_client_ms: 20, search_server_ms: 10, analysis_elapsed_ms: 50, analysis, error: '' };
const run = { id: 1, kb_id: 7, request_id: '177654b5-56f1-4a44-afb4-f39ccb5e261f', dataset_key: 'test', dataset_sha256: sha, group_key: 'project_ops', status: 'succeeded', case_count: 1, completed_count: 1, failed_count: 0, stop_requested: false, corpus_verified: true, error: '', snapshot: raw, bindings: JSON.stringify({ documents: [binding], inventory: [{ id: 9, kb: 'project_ops', kb_id: 7, title: document.title, version: 2, status: 'ready', embedding_model: 'embedding-test' }] }), context: JSON.stringify(context), observations: JSON.stringify([observation]), created_at: 1, updated_at: 2, started_at: 1, completed_at: 2 };
function storage() { const data = new Map(); return { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, v), removeItem: k => data.delete(k) }; }

test('质量验收的冻结题集、实际观测和原请求恢复', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const helpers = await server.ssrLoadModule('/src/pages/ai/knowledgeQuality.ts').catch(() => ({}));
    await t.test('分析8来源可解码展示，第9来源拒绝，普通检索仍最多5项', async () => {
      const expanded = count => ({ ...analysis, sources: Array.from({ length: count }, (_, i) => ({ ...source, index: i + 1, chunk_id: 11 + i, snippet: `先备份。扩展片段${i + 1}` })) });
      const decode = (result, list = observation.list) => helpers.decodeQualityEvaluation({ ...run, observations: JSON.stringify([{ ...observation, list, analysis: result }]) }, 7);
      for (const count of [5, 8]) assert.equal((await decode(expanded(count))).rows[0].analysis.sources.length, count);
      await assert.rejects(decode(expanded(9)));
      await assert.rejects(decode(expanded(8), Array.from({ length: 6 }, (_, i) => ({ ...source, index: undefined, chunk_id: 11 + i }))));
      const view = await decode(expanded(8));
      const originalWindow = globalThis.window;
      globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
      try {
        const { QualityAnswer } = await server.ssrLoadModule('/src/pages/ai/KnowledgeQualityDetails.tsx');
        const html = renderToStaticMarkup(createElement(QualityAnswer, { row: view.rows[0], scope: ['fixture', 1, 1], guard: { writing: false, can: () => false } }));
        for (const index of [6, 7, 8]) assert.ok(html.includes(`扩展片段${index}`));
      } finally { globalThis.window = originalWindow; }
    });
    await t.test('冻结题集必须核对原字节SHA，拒重复ID与重复JSON键', async () => {
      assert.equal(typeof helpers.parseQualityDataset, 'function');
      const loaded = await helpers.parseQualityDataset({ key: 'test', name: '验收', sha256: sha, dataset: raw });
      assert.equal(loaded.dataset.cases[0].query, item.query);
      await assert.rejects(helpers.parseQualityDataset({ key: 'test', name: '验收', sha256: 'f'.repeat(64), dataset: raw }));
      const duplicate = JSON.stringify({ ...dataset, cases: [item, item] });
      await assert.rejects(helpers.parseQualityDataset({ key: 'test', name: '验收', sha256: createHash('sha256').update(duplicate).digest('hex'), dataset: duplicate }));
      assert.throws(() => helpers.parseQualityJson('{"status":"failed","status":"succeeded"}'));
      assert.throws(() => helpers.parseQualityJson('{"x":1,"\\u0078":2}'));
    });
    await t.test('正负例规则完整且预期文档与证据一一对应', async () => {
      const parse = value => { const content = JSON.stringify(value); return helpers.parseQualityDataset({ key: 'test', name: '验收', sha256: createHash('sha256').update(content).digest('hex'), dataset: content }); };
      for (const changed of [
        { evidence: [], answer: { ...item.answer, required_groups: [] }, checks: undefined },
        { evidence: [] },
        { answer: { ...item.answer, required_groups: [] }, checks: undefined },
        { evidence: [item.evidence[0], item.evidence[0]] },
        { evidence: [{ ...item.evidence[0], groups: [] }] },
        { expected_documents: [], evidence: [] },
        { answer: { ...item.answer, kind: 'unanswerable' }, expected_documents: [], evidence: [], checks: undefined },
      ]) await assert.rejects(parse({ ...dataset, cases: [{ ...item, ...changed }] }), '缺失或相互矛盾的规则不能派生质量分数');
      const secondDoc = { ...document, key: 'other', title: '补充规则' };
      await assert.rejects(parse({ ...dataset, documents: [document, secondDoc], cases: [{ ...item, expected_documents: ['rules', 'other'], evidence: [item.evidence[0], item.evidence[0]] }] }), '等长重复证据也不能替代另一份预期文档');
      const negative = { ...item, expected_documents: [], evidence: [], answer: { ...item.answer, kind: 'unanswerable', required_groups: [] }, checks: undefined };
      assert.equal((await parse({ ...dataset, cases: [negative] })).dataset.cases[0].answer.kind, 'unanswerable');
    });
    await t.test('耗时必须来自完整观测，成功项边界与等待零值不能伪造', async () => {
      const decode = obs => helpers.decodeQualityEvaluation({ ...run, observations: JSON.stringify([obs]) }, 7);
      for (const name of ['search_client_ms', 'search_server_ms', 'analysis_elapsed_ms']) {
        const missing = { ...observation }; delete missing[name];
        await assert.rejects(decode(missing), `${name}缺失不能补成0并进入比较`);
        for (const value of [-1, null, 100_001]) await assert.rejects(decode({ ...observation, [name]: value }));
      }
      for (const name of ['search_server_ms', 'analysis_elapsed_ms']) await assert.rejects(decode({ ...observation, [name]: 1.5 }));
      const missingList = { ...observation }; delete missingList.list;
      await assert.rejects(decode(missingList), '缺失检索观测不能当成真实空召回');
      await assert.rejects(decode({ ...observation, list: null }));
      const zero = await decode({ ...observation, search_client_ms: 0, search_server_ms: 0, analysis_elapsed_ms: 0 });
      assert.equal(zero.summary.latency.analysis_submit_to_terminal.p50_ms, 0, '明确记录的真实0毫秒仍是有效观测');
      const waiting = { ...observation, task_id: 0, status: 'waiting', stage: 'waiting', list: [], search_client_ms: 0, search_server_ms: 0, analysis_elapsed_ms: 0, analysis: undefined };
      const pending = { ...run, status: 'pending', completed_count: 0, corpus_verified: false };
      const view = await helpers.decodeQualityEvaluation({ ...pending, observations: JSON.stringify([waiting]) }, 7);
      assert.equal(view.summary.n, 0);
      assert.equal(view.summary.latency.analysis_submit_to_terminal, null);
      await assert.rejects(helpers.decodeQualityEvaluation({ ...pending, observations: JSON.stringify([{ ...waiting, analysis_elapsed_ms: 50 }]) }, 7));
      const unknown = { ...run, status: 'unknown', completed_count: 0, failed_count: 1, corpus_verified: false };
      const timedOut = { ...observation, status: 'unknown', analysis: undefined, analysis_elapsed_ms: 100_001, error: '等待超时' };
      assert.equal((await helpers.decodeQualityEvaluation({ ...unknown, observations: JSON.stringify([timedOut]) }, 7)).rows[0].analysisMs, 100_001, '真实超时诊断允许调度余量，但不进入成功统计');
      await assert.rejects(helpers.decodeQualityEvaluation({ ...unknown, observations: JSON.stringify([{ ...timedOut, analysis_elapsed_ms: 110_001 }]) }, 7));
    });
    await t.test('评分复用现有纯函数，执行失败不伪造成质量警报或零用量', async () => {
      const view = await helpers.decodeQualityEvaluation(run, 7);
      assert.equal(view.rows[0].screen.flagged, false);
      assert.equal(view.rows[0].checks[0].matched, true);
      assert.equal(view.summary.usage.known_total_tokens, 30);
      const failure = { ...run, status: 'failed', completed_count: 0, failed_count: 1, corpus_verified: false, observations: JSON.stringify([{ ...observation, status: 'failed', analysis: undefined, error: '模型调用未完成' }]) };
      const bad = await helpers.decodeQualityEvaluation(failure, 7);
      assert.equal(bad.rows[0].executionFailed, true);
      assert.equal(bad.rows[0].screen, null);
      assert.equal(bad.rows[0].analysis, null);
      assert.equal(bad.summary.n, 0);
      assert.equal(bad.summary.latency.analysis_submit_to_terminal, null);
      const originalWindow = globalThis.window;
      globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
      try {
        const { KnowledgeQualityResults } = await server.ssrLoadModule('/src/pages/ai/KnowledgeQualityDetails.tsx');
        assert.doesNotThrow(() => renderToStaticMarkup(createElement(KnowledgeQualityResults, { view: bad, scope: ['fixture', 1, 1], guard: { writing: false, can: () => false }, filter: 'all', canRepeat: () => false, onRepeat() {}, onTask() {} })));
      } finally { globalThis.window = originalWindow; }
    });
    await t.test('错知识库、伪造问题、来源和重复案例均拒绝', async () => {
      await assert.rejects(helpers.decodeQualityEvaluation(run, 8));
      for (const obs of [[{ ...observation, query: '已被改动' }], [observation, observation], [{ ...observation, analysis: { ...analysis, sources: [{ ...source, kb_id: 8 }] } }]]) {
        await assert.rejects(helpers.decodeQualityEvaluation({ ...run, observations: JSON.stringify(obs) }, 7));
      }
    });
    await t.test('只比较完整同题同语料同预算，构建变化单列', async () => {
      const first = await helpers.decodeQualityEvaluation(run, 7);
      assert.equal(helpers.compareQualityRuns(first, first).comparable, false);
      const next = await helpers.decodeQualityEvaluation({ ...run, id: 2, context: JSON.stringify({ ...context, binary_sha: 'c'.repeat(64) }) }, 7);
      assert.equal(helpers.compareQualityRuns(first, next).comparable, true);
      for (const changed of [{ status: 'unknown' }, { corpus_verified: false }, { group_key: 'other' }, { context: JSON.stringify({ ...context, model: { ...context.model, name: 'changed' } }) }, { bindings: JSON.stringify({ documents: [{ ...binding, version: 3 }], inventory: [{ id: 9, kb: 'project_ops', kb_id: 7, title: document.title, version: 3, status: 'ready', embedding_model: 'embedding-test' }] }) }]) {
        let second; try { second = await helpers.decodeQualityEvaluation({ ...run, ...changed }, 7); } catch { continue; }
        assert.equal(helpers.compareQualityRuns(first, second).comparable, false);
      }
    });
    await t.test('显式新请求必须同组，待确认请求按会话/知识库保存且不换UUID', async () => {
      const catalogue = await helpers.parseQualityDataset({ key: 'test', name: '验收', sha256: sha, dataset: raw });
      const attempt = helpers.createQualityAttempt(catalogue, 'project_ops', [item.id], run.request_id);
      assert.equal(attempt.request_id, run.request_id);
      assert.throws(() => helpers.createQualityAttempt(catalogue, 'project_ops', [item.id, item.id], run.request_id));
      assert.throws(() => helpers.createQualityAttempt(catalogue, 'missing', [item.id], run.request_id));
      const local = storage(), key = helpers.qualityAttemptKey(['session-a', 1, 1], null, 7);
      helpers.writeQualityAttempt(local, key, attempt);
      assert.deepEqual(helpers.readQualityAttempt(local, key), attempt);
      assert.equal(helpers.readQualityAttempt(local, helpers.qualityAttemptKey(['session-b', 1, 1], null, 7)), null);
      assert.throws(() => helpers.writeQualityAttempt({ setItem() { throw new Error('quota'); } }, key, attempt));
      assert.throws(() => helpers.assertQualityAttemptResult({ ...run, dataset_sha256: 'c'.repeat(64) }, attempt));
      helpers.assertQualityAttemptResult(run, attempt);
      local.setItem(key, JSON.stringify({ ...attempt, unexpected: '不得转发' }));
      assert.equal(Object.hasOwn(helpers.readQualityAttempt(local, key), 'unexpected'), false);
    });
    await t.test('页面重载与部门返回后按实际身份恢复，版本变化不能丢失未知请求', async () => {
      const catalogue = await helpers.parseQualityDataset({ key: 'test', name: '验收', sha256: sha, dataset: raw });
      const attempt = helpers.createQualityAttempt(catalogue, 'project_ops', [item.id], run.request_id), local = storage();
      const key = helpers.qualityAttemptKey(['session-a', 1, 1], 23, 7);
      helpers.writeQualityAttempt(local, key, attempt);
      assert.deepEqual(helpers.readQualityAttempt(local, helpers.qualityAttemptKey(['session-a', 0, 0], 23, 7)), attempt, 'F5后同身份和实际部门仍可恢复原UUID');
      assert.deepEqual(helpers.readQualityAttempt(local, helpers.qualityAttemptKey(['session-a', 1, 4], 23, 7)), attempt, '切部门再返回时恢复原标识');
      for (const other of [helpers.qualityAttemptKey(['session-b', 1, 1], 23, 7), helpers.qualityAttemptKey(['session-a', 1, 1], 24, 7), helpers.qualityAttemptKey(['session-a', 1, 1], 23, 8)]) assert.equal(helpers.readQualityAttempt(local, other), null);
    });
    await t.test('单题复测只允许已明确终止或已完成的执行，未知结果保留原任务', () => {
      assert.equal(typeof helpers.canRepeatQualityRow, 'function');
      for (const parent of ['pending', 'running', 'succeeded', 'failed', 'cancelled', 'unknown']) {
        for (const status of ['waiting', 'running', 'succeeded', 'failed', 'unknown']) {
          const expected = status === 'succeeded' || status === 'failed' || status === 'waiting' && ['failed', 'cancelled'].includes(parent);
          assert.equal(helpers.canRepeatQualityRow(parent, { status, taskId: status === 'waiting' ? 0 : 12 }), expected, `${parent}/${status}`);
        }
        assert.equal(helpers.canRepeatQualityRow(parent, { status: 'waiting', taskId: 12 }), false, '有已提交任务的题目不能当作尚未开始');
      }
    });
    await t.test('执行失败或未知详情保留可复制原UUID，原任务入口仍指向原任务', async () => {
      const originalWindow = globalThis.window;
      globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
      try {
        const { QualityAnswer, KnowledgeQualityResults } = await server.ssrLoadModule('/src/pages/ai/KnowledgeQualityDetails.tsx');
        assert.equal(typeof QualityAnswer, 'function');
        for (const status of ['failed', 'unknown']) {
          const obs = { ...observation, status, analysis: undefined, error: '原结果待核对' };
          const view = await helpers.decodeQualityEvaluation({ ...run, status, completed_count: 0, failed_count: 1, corpus_verified: false, observations: JSON.stringify([obs]) }, 7);
          const props = { scope: ['fixture', 1, 1], guard: { writing: false, can: () => true } };
          const answer = renderToStaticMarkup(createElement(QualityAnswer, { row: view.rows[0], ...props }));
          assert.ok(answer.includes(observation.request_id));
          assert.ok(answer.includes('data-icon="copy"'), '原UUID只读且提供复制入口');
          assert.ok(answer.includes('本题请求标识'));
          const table = renderToStaticMarkup(createElement(KnowledgeQualityResults, { view, ...props, filter: 'all', canRepeat: row => helpers.canRepeatQualityRow(view.run.status, row), onRepeat() {}, onTask() {} }));
          const retest = table.match(/<button\b[^>]*>(?:(?!<\/button>)[\s\S])*?单题复测(?:(?!<\/button>)[\s\S])*?<\/button>/u)?.[0];
          assert.ok(retest);
          assert.equal(/\bdisabled=/u.test(retest), status === 'unknown');
          assert.ok(table.includes('运行记录'), '已存在的原任务保留查看入口');
        }
      } finally { globalThis.window = originalWindow; }
    });
  } finally { await server.close(); }
});

test('质量验收API与Query遵守统一认证、取消和作用域', async t => {
  const originalWindow = globalThis.window;
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  const client = new QueryClient();
  try {
    const api = await server.ssrLoadModule('/src/api/ai.ts');
    const queries = await server.ssrLoadModule('/src/pages/ai/knowledgeQualityQueries.ts');
    const { request, setRequestErrorReporter } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const scope = ['quality-session', 1, 1];
    useAuthStore.setState({ token: 'local-quality-fixture', sessionId: scope[0], sessionVersion: 1, departmentVersion: 1, expiresAt: Date.now() / 1000 + 3600,
      permissionsReady: true, isLoggingOut: false, permCodes: ['sys:ai:kb:list', 'sys:ai:kb:evaluate'] });
    setRequestErrorReporter(() => {});
    const response = (config, result, code = 200) => ({ data: { code, result, msg: code === 409 ? '题集已更新' : '' }, status: 200, statusText: 'OK', headers: {}, config });
    await t.test('五个接口保持请求原UUID与选择，执行失败不自动重试', async () => {
      const calls = [];
      request.defaults.adapter = async config => { calls.push(config); return response(config, { evaluation: run, list: [], total: 0 }); };
      await api.fetchAiKbEvaluationDatasets(7);
      const attempt = { request_id: run.request_id, dataset_key: run.dataset_key, dataset_sha256: sha, group_key: run.group_key, case_ids: [item.id] };
      await api.startAiKbEvaluation(7, attempt);
      await api.fetchAiKbEvaluations(7, { page: 2, page_size: 10, status: 'failed', request_id: run.request_id });
      await api.fetchAiKbEvaluation(7, 1);
      await api.stopAiKbEvaluation(7, 1);
      assert.deepEqual(calls.map(c => `${c.method} ${c.url}`), ['get /sys/ai/kbs/7/evaluation-datasets', 'post /sys/ai/kbs/7/evaluations', 'get /sys/ai/kbs/7/evaluations', 'get /sys/ai/kbs/7/evaluations/1', 'post /sys/ai/kbs/7/evaluations/1/stop']);
      assert.deepEqual(JSON.parse(calls[1].data), attempt);
      assert.ok(calls.every(c => c.headers.Authorization === 'Bearer local-quality-fixture'));
      assert.deepEqual(calls[2].params, { page: 2, page_size: 10, status: 'failed', request_id: run.request_id });
      request.defaults.adapter = async config => { calls.push(config); return response(config, {}, 409); };
      await assert.rejects(api.startAiKbEvaluation(7, attempt), e => e.code === 409);
      assert.equal(calls.length, 6);
    });
    await t.test('分页/身份/知识库隔离，取消后迟到响应不能回填', async () => {
      const options = queries.qualityListOptions(scope, 7, { page: 2, page_size: 10 });
      assert.equal(options.retry, false);
      assert.notDeepEqual(options.queryKey, queries.qualityListOptions(scope, 7, { page: 1, page_size: 10 }).queryKey);
      assert.notDeepEqual(options.queryKey, queries.qualityListOptions(['other-session', 1, 1], 7, { page: 2, page_size: 10 }).queryKey);
      assert.notDeepEqual(options.queryKey, queries.qualityListOptions(scope, 8, { page: 2, page_size: 10 }).queryKey);
      let release, started;
      const invoked = new Promise(resolve => { started = resolve; });
      request.defaults.adapter = async config => { started(); await new Promise(resolve => { release = resolve; }); return response(config, { list: [run], total: 1 }); };
      const result = client.fetchQuery(options); await invoked;
      await client.cancelQueries({ queryKey: options.queryKey, exact: true }); release();
      await assert.rejects(result); assert.equal(client.getQueryData(options.queryKey), undefined);
    });
    setRequestErrorReporter(undefined);
  } finally { client.clear(); await server.close(); globalThis.window = originalWindow; }
});
