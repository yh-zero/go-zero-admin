import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { QueryClient, QueryClientProvider, QueryObserver } from '@tanstack/react-query';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const storage = () => { const values = new Map(); return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }; };

test('知识库操作保留上下文并减少无效读取', async t => {
  const originalWindow = globalThis.window;
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {}, removeEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  try {
    const queries = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
    const { request } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const scope = ['knowledge-optimization', 1, 1];
    useAuthStore.setState({ token: 'local-fixture', sessionId: scope[0], sessionVersion: 1, departmentVersion: 1, permissionsReady: true, isLoggingOut: false,
      expiresAt: Date.now() / 1000 + 3600, permCodes: ['sys:ai:kb:list'] });
    const calls = [];
    request.defaults.adapter = async config => { calls.push(config); return { config, status: 200, statusText: 'OK', headers: {}, data: { code: 200, result: config.params?.choices_only ? { choices: [], total: 0 } : { list: [], total: 0, pending_count: 1, ready_count: 0, failed_count: 0 } } }; };

    await t.test('资料页只在签名或库统计变化后补读当前完整页', async () => {
      const previous = request.defaults.adapter;
      let revision = 'v1', pending = 1;
      const projections = [];
      request.defaults.adapter = async config => {
        const meta = !!config.params.meta_only;
        projections.push(meta);
        const result = { list: meta ? [] : [{ id: 9, status: pending ? 'pending' : 'ready', stage: revision }], total: 1, pending_count: pending, ready_count: 1 - pending, failed_count: 0, page_signature: revision };
        return { config, status: 200, headers: {}, data: { code: 200, result } };
      };
      try {
        const opts = queries.kbDocListOptions(scope, 7, { page: 2, page_size: 10, query: 'same' });
        const first = await client.fetchQuery(opts), same = await client.fetchQuery(opts);
        assert.deepEqual(same, first);
        revision = 'v2';
        assert.equal((await client.fetchQuery(opts)).list[0].stage, 'v2');
        pending = 0;
        const end = await client.fetchQuery(opts);
        assert.equal(end.list[0].status, 'ready');
        assert.equal(opts.refetchInterval({ state: { data: end } }), false);
        assert.deepEqual(projections, [false, true, true, false, true, false]);
      } finally { request.defaults.adapter = previous; client.clear(); }
    });
    await t.test('旧服务的选项列表移除管理字段，跨页选中值按原 ID 读取', async () => {
      const { fetchAiKbChoices } = await server.ssrLoadModule('/src/api/ai.ts');
      const previous = request.defaults.adapter, paths = [];
      request.defaults.adapter = async config => {
        paths.push(config.url);
        const result = config.url.endsWith('/42') ? { kb: { id: 42, name: '原值', chunk_count: 30 } } : { list: [{ id: 1, name: '第一页', chunk_count: 20 }], total: 31 };
        return { config, status: 200, headers: {}, data: { code: 200, result } };
      };
      try {
        assert.deepEqual(await fetchAiKbChoices({}), { list: [{ id: 1, name: '第一页' }], total: 31 });
        assert.deepEqual(await fetchAiKbChoices({ id: 42 }), { list: [{ id: 42, name: '原值' }], total: 1 });
        assert.deepEqual(paths, ['/sys/ai/kbs', '/sys/ai/kbs', '/sys/ai/kbs/42']);
      } finally { request.defaults.adapter = previous; }
    });
    await t.test('旧服务精确 ID 与关键词组合从筛选分页核对，不因详情可读而绕过关键词', async () => {
      const { fetchAiKbChoices } = await server.ssrLoadModule('/src/api/ai.ts');
      const previous = request.defaults.adapter, reads = [];
      const matches = Array.from({ length: 503 }, (_, i) => ({ id: (503 - i) * 2, name: '描述匹配的选项' }));
      request.defaults.adapter = async config => {
        assert.equal(config.url, '/sys/ai/kbs');
        const { query, page, page_size: size } = config.params;
        assert.equal(query, '%关键字'); assert.equal(size, 100);
        reads.push(page);
        return { config, status: 200, headers: {}, data: { code: 200, result: { list: matches.slice((page - 1) * size, page * size), total: matches.length } } };
      };
      try {
        assert.deepEqual(await fetchAiKbChoices({ id: 4, page: 9, page_size: 1, query: '%关键字' }), { list: [{ id: 4, name: '描述匹配的选项' }], total: 1 });
        assert.deepEqual(reads, [1, 4, 5, 6]);
        reads.length = 0;
        assert.deepEqual(await fetchAiKbChoices({ id: 3, query: '%关键字' }), { list: [], total: 0 });
        assert.deepEqual(reads, [1, 4, 5, 6]);
        reads.length = 0;
        assert.deepEqual(await fetchAiKbChoices({ id: 2000, query: '%关键字' }), { list: [], total: 0 });
        assert.deepEqual(reads, [1]);
      } finally { request.defaults.adapter = previous; }
    });
    await t.test('隐藏资料标签不发请求也不轮询，恢复后保留同一筛选缓存', async () => {
      const params = { page: 2, page_size: 10, query: '故障', status: 'ready' };
      const hidden = queries.kbDocListOptions(scope, 7, params, false);
      const observer = new QueryObserver(client, hidden);
      const unsubscribe = observer.subscribe(() => {});
      try {
        await new Promise(resolve => setTimeout(resolve, 10));
        assert.equal(calls.length, 0, '隐藏面板不读取列表');
        assert.equal(hidden.refetchInterval({ state: { data: { list: [{ status: 'pending' }], pending_count: 1 } } }), false);
        const visible = queries.kbDocListOptions(scope, 7, params, true);
        assert.deepEqual(hidden.queryKey, visible.queryKey, '可见性不改变筛选缓存的身份');
        observer.setOptions(visible);
        await observer.refetch();
        assert.equal(calls.length, 1);
        assert.deepEqual(calls[0].params, params);
      } finally { unsubscribe(); client.clear(); calls.length = 0; }
    });

    await t.test('快速输入只读取最后查询，关闭取消计时器与已经发出的查询', async () => {
      const search = await server.ssrLoadModule('/src/pages/ai/knowledgeSearch.ts');
      assert.equal(typeof search.createKnowledgeSearchDebouncer, 'function');
      t.mock.timers.enable({ apis: ['setTimeout'] });
      let currentKey;
      const requests = [];
      const debouncer = search.createKnowledgeSearchDebouncer(value => {
        const options = search.kbChoiceOptions(scope, { page: 1, page_size: 20, query: value });
        currentKey = options.queryKey;
        requests.push(client.fetchQuery(options));
      }, () => { if (currentKey) void client.cancelQueries({ queryKey: currentKey, exact: true }); });
      try {
        debouncer.search('发'); t.mock.timers.tick(100);
        debouncer.search('发布'); t.mock.timers.tick(100);
        debouncer.search('发布规范'); t.mock.timers.tick(299);
        assert.equal(calls.length, 0);
        t.mock.timers.tick(1); await Promise.all(requests);
        assert.equal(calls.length, 1);
        assert.equal(calls[0].params.query, '发布规范');
        assert.equal(calls[0].params.choices_only, true);
        debouncer.search('关闭前的输入'); debouncer.cancel(); t.mock.timers.tick(1000);
        assert.equal(calls.length, 1, '已关闭的下拉框不再启动搜索');

        let release; let started;
        const sent = new Promise(resolve => { started = resolve; });
        request.defaults.adapter = config => new Promise(resolve => { release = () => resolve({ config, status: 200, statusText: 'OK', headers: {}, data: { code: 200, result: { list: [], total: 0 } } }); started(); });
        debouncer.search('迟到结果'); t.mock.timers.tick(300);
        const pending = requests.at(-1);
        const rejected = assert.rejects(pending);
        await sent;
        debouncer.cancel(); release(); await rejected;
        assert.equal(client.getQueryData(currentKey), undefined, '关闭后的迟到查询不回填有效结果');
      } finally { debouncer.cancel(); t.mock.timers.reset(); client.clear(); calls.length = 0; }
    });

    await t.test('检索结果保留原问题且不重复提供分析入口', async () => {
      const { KbSearchResults } = await server.ssrLoadModule('/src/pages/ai/KbSearchModal.tsx');
      const html = renderToStaticMarkup(createElement(KbSearchResults, { result: { list: [], elapsed_ms: 1 }, query: '发布失败如何回滚？' }));
      assert.doesNotMatch(html, /用此问题分析/);
      assert.match(html, /发布失败如何回滚/);
    });

    await t.test('统一问题输入提供两项显式操作，恢复未知请求时保留原标识与问题', async () => {
      const previousStorage = globalThis.sessionStorage;
      globalThis.sessionStorage = window.sessionStorage;
      const { WorkflowDraftRun } = await server.ssrLoadModule('/src/pages/ai/WorkflowDraftRun.tsx');
      const { workflowTemplate } = await server.ssrLoadModule('/src/pages/ai/workflowDefinition.ts');
      const { draftRunAttemptKey, writeDraftRunAttempt } = await server.ssrLoadModule('/src/pages/ai/draftRunAttempts.ts');
      const props = { scope, attemptId: 'knowledge:7', guard: { writing: false, can: () => true, isCurrent: () => true },
        draft: workflowTemplate('kb_analyze', []), fixedKb: { id: 7, ready_count: 1 }, capabilities: { retrieval_configured: true, analysis_configured: true },
        onFindSources() {} };
      const render = () => renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(WorkflowDraftRun, props)));
      const key = draftRunAttemptKey([scope[0], null], props.attemptId);
      const before = calls.length;
      try {
        const html = render();
        assert.match(html, /查找原文/);
        assert.match(html, /生成带引用的回答/);
        assert.equal(calls.length, before, '显示操作不会触发检索或模型');
        const request = { request_id: '11111111-1111-4111-8111-111111111111', name: '原运行', definition: '{"version":2,"steps":[]}', input: '{"question":"原未知问题","kb_id":7}' };
        writeDraftRunAttempt(sessionStorage, key, request);
        const restored = render();
        assert.match(restored, /原未知问题/);
        assert.match(restored, /11111111-1111-4111-8111-111111111111/);
        assert.equal(calls.length, before);
      } finally { sessionStorage.removeItem(key); globalThis.sessionStorage = previousStorage; }
    });

    await t.test('分析和反馈原文入口使用就地预览按钮，保留当前结果', async () => {
      const { WorkflowAnalysisResult } = await server.ssrLoadModule('/src/pages/ai/WorkflowAnalysisResult.tsx');
      const { FeedbackSourceSnapshot } = await server.ssrLoadModule('/src/pages/ai/KnowledgeFeedbackPanel.tsx');
      const source = { index: 1, chunk_id: 11, kb_id: 7, doc_id: 9, doc_title: '发布规范', snippet: '先备份。', score: 0.9 };
      const analysis = { version: 1, type: 'kb_analysis', status: 'completed', analysis: '先备份。[1]', sources: [source], usage: { scope: 'chat', prompt_tokens: 1, completion_tokens: 1, total_tokens: 2, complete: true } };
      for (const html of [
        renderToStaticMarkup(createElement(WorkflowAnalysisResult, { analysis: { step: 1, name: '分析', result: analysis } })),
        renderToStaticMarkup(createElement(FeedbackSourceSnapshot, { snapshot: { sources: [source], documents: [] }, onOpenDoc() {} })),
      ]) {
        assert.match(html, /查看当前原文/);
        assert.doesNotMatch(html, /href="\/ai\/knowledge\//, '查看原文不整页离开当前工作区');
      }
    });
  } finally { client.clear(); await server.close(); globalThis.window = originalWindow; }
});
