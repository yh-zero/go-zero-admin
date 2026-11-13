import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';

const storage = () => {
  const values = new Map();
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
};

test('知识库和资料名称作为默认入口，次要操作按需展开', async t => {
  const previousWindow = globalThis.window;
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  try {
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const scope = ['knowledge-presentation', 1, 1];
    useAuthStore.setState({ token: 'local-fixture', sessionId: scope[0], sessionVersion: 1, departmentVersion: 1, permissionsReady: true, isLoggingOut: false,
      permCodes: ['sys:ai:kb:list', 'sys:ai:kb:edit', 'sys:ai:kb:delete', 'sys:ai:kb:evaluate'] });
    Object.assign(useAuthStore.getInitialState(), useAuthStore.getState());
    const { aiWorkspaceKeys } = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
    const kb = { id: 7, name: '发布手册', description: '', owner_id: 1, doc_count: 1, chunk_count: 0, created_at: 1, updated_at: 1, ready_count: 0, pending_count: 0, failed_count: 1 };
    const capabilities = { analysis_configured: true, retrieval_configured: true };
    client.setQueryData([...aiWorkspaceKeys.kbs(scope), { page: 1, page_size: 20, query: '' }], { list: [kb], total: 1, capabilities });
    client.setQueryData(aiWorkspaceKeys.kb(scope, 7), { kb, capabilities });
    client.setQueryData([...aiWorkspaceKeys.kbDocs(scope, 7), { page: 1, page_size: 10 }], { list: [{ id: 8, kb_id: 7, title: '故障指南', status: 'failed', stage: 'failed', version: 1, error: '处理失败', chunk_count: 0, embedding_model: '', created_at: 1, updated_at: 1 }], total: 1, ready_count: 0, pending_count: 0, failed_count: 1 });
    const render = Component => renderToStaticMarkup(createElement(QueryClientProvider, { client }, createElement(MemoryRouter, null, createElement(Component, { scope, kbId: 7, kb, onChanged() {} }))));
    await t.test('名称是唯一知识库入口，编辑删除默认收起', async () => {
      const { AIKnowledgeSession } = await server.ssrLoadModule('/src/pages/ai/AIKnowledge.tsx');
      const html = render(AIKnowledgeSession);
      assert.match(html, /发布手册/);
      assert.doesNotMatch(html, /进入工作区/);
      assert.match(html, /更多/);
      assert.doesNotMatch(html, />编辑<|>删除</);
    });
    await t.test('资料名称打开原文，失败重试明确且添加方式收起', async () => {
      const { KnowledgeDocumentsPanel } = await server.ssrLoadModule('/src/pages/ai/KnowledgeDocumentsDrawer.tsx');
      const html = render(KnowledgeDocumentsPanel);
      assert.match(html, /故障指南/);
      assert.match(html, />重试处理</);
      assert.match(html, /添加资料/);
      assert.match(html, /查看导入批次/);
      assert.doesNotMatch(html, />原文<|>导入 TXT \/ Markdown<|>粘贴资料<|>删除</);
    });
  } finally { client.clear(); await server.close(); globalThis.window = previousWindow; }
});

test('知识库后台摄取返回接收凭证，提交与普通管理请求保留默认超时', async () => {
  const previousWindow = globalThis.window;
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const api = await server.ssrLoadModule('/src/api/ai.ts');
    const { request } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    useAuthStore.getState().setSession({ access_token: 'local-kb-test', refresh_token: 'local-kb-refresh',
      expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'local-kb-session' });
    useAuthStore.getState().setPermCodes(['sys:ai:kb:edit']);
    const requests = [];
    request.defaults.adapter = async config => {
      requests.push(config);
      return { data: { code: 200, result: { doc: { id: 9, kb_id: 7, title: '本地摄取测试', status: 'pending', stage: 'queued', version: 1, error: '', chunk_count: 1, embedding_model: 'fixture-embedding', created_at: 1, updated_at: 1 }, job_id: 4, batch_id: '11111111-1111-4111-8111-111111111111' } }, status: 200, statusText: 'OK', headers: {}, config };
    };
    const content = { title: '本地摄取测试', content: '只通过请求适配器检查，不调用模型。', request_id: '11111111-1111-4111-8111-111111111111' };
    const accepted = await api.createAiKbDoc(7, content);
    assert.equal(accepted.doc.status, 'pending');
    assert.equal(accepted.job_id, 4);
    assert.equal(accepted.batch_id, content.request_id);
    assert.equal(requests[0].timeout, 30_000);
    assert.equal(requests[0].url, '/sys/ai/kbs/7/docs');
    assert.deepEqual(JSON.parse(requests[0].data), content);
    await api.retryAiKbDoc(7, 9, { request_id: '22222222-2222-4222-8222-222222222222' });
    assert.equal(requests[1].timeout, 30_000);
    assert.equal(requests[1].url, '/sys/ai/kbs/7/docs/9/retry');
    await api.fetchAiKbDocs(7);
    assert.equal(requests[2].timeout, 30_000);
    await api.fetchAiTools();
    assert.equal(requests[3].timeout, 30_000);
    await api.updateAiKbDoc(7, 9, { ...content, expected_version: 2 });
    assert.equal(requests[4].timeout, 30_000);
    assert.equal(requests[4].url, '/sys/ai/kbs/7/docs/9');
    assert.equal(JSON.parse(requests[4].data).expected_version, 2);
    await api.updateAiKb(7, { name: '更新标题' });
    assert.equal(requests[5].timeout, 30_000);
    useAuthStore.getState().setPermCodes([]);
    await assert.rejects(api.updateAiKbDoc(7, 9, { ...content, expected_version: 2 }), /权限/);
    assert.equal(requests.length, 6, '无权限的更新不得发出请求');
  } finally {
    await server.close();
    globalThis.window = previousWindow;
  }
});

test('知识库文档提交与状态恢复行为', async t => {
  const previousWindow = globalThis.window;
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const documents = await server.ssrLoadModule('/src/pages/ai/knowledgeDocuments.ts').catch(() => ({}));
    await t.test('正文按UTF-8字节校验，保留64KiB边界并拒绝空白', () => {
      assert.equal(typeof documents.validateDocumentContent, 'function', '缺少文档正文校验');
      assert.equal(documents.validateDocumentContent('a'.repeat(65536)), undefined);
      assert.equal(documents.validateDocumentContent('中'.repeat(21845) + 'a'), undefined);
      assert.equal(documents.documentContentBytes('🙂'), 4);
      assert.ok(documents.validateDocumentContent('中'.repeat(21846)));
      assert.ok(documents.validateDocumentContent(' \n\t '));
    });
    await t.test('pending保持后台作业身份，failed可重试、ready不可重试', () => {
      assert.equal(typeof documents.canRetryDocument, 'function', '缺少重试保护');
      const pending = { id: 9, status: 'pending', created_at: 1, updated_at: 100 };
      assert.equal(documents.canRetryDocument(pending, 219999), false);
      assert.equal(documents.canRetryDocument(pending, 220000), false);
      assert.equal(documents.canRetryDocument({ ...pending, status: 'failed' }, 100001), true);
      assert.equal(documents.canRetryDocument({ ...pending, status: 'ready' }, 999999), false);
      for (const stage of ['queued', 'embedding', 'indexing']) assert.equal(documents.canRetryDocument({ ...pending, stage, updated_at: 1 }, 999999999), false);
    });
    await t.test('后台批次逐篇使用预生成标识提交，接收后不自动重放', async () => {
      const { prepareKnowledgeBatch, runKnowledgeImport } = await server.ssrLoadModule('/src/pages/ai/knowledgeImport.ts');
      const { createAiKbDoc } = await server.ssrLoadModule('/src/api/ai.ts');
      const { QueryClient, MutationObserver } = await import('@tanstack/react-query');
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      useAuthStore.setState({ token: 'local-doc-test', sessionId: 'local-doc-session', sessionVersion: 1, departmentVersion: 1,
        expiresAt: Date.now() / 1000 + 3600, permissionsReady: true, permCodes: ['sys:ai:kb:edit'], isLoggingOut: false });
      const client = new QueryClient({ defaultOptions: { queries: { gcTime: 0 }, mutations: { gcTime: 0 } } });
      const responses = ['pending', 'ready'];
      const calls = [];
      request.defaults.adapter = async config => {
        calls.push({ url: config.url, data: JSON.parse(config.data) });
        return { data: { code: 200, result: { doc: { id: 9, status: responses.shift(), error: '索引服务暂不可用', updated_at: Math.floor(Date.now() / 1000) } } },
          status: 200, statusText: 'OK', headers: {}, config };
      };
      try {
        const mutation = new MutationObserver(client, { mutationFn: values => createAiKbDoc(7, values), retry: false });
        const prepared = prepareKnowledgeBatch([{ title: '文档甲', content: '正文甲' }, { title: '文档乙', content: '正文乙' }]);
        const changes = [];
        await runKnowledgeImport(prepared.items, values => mutation.mutate(values), change => changes.push(change), () => true);
        assert.equal(calls.length, 2);
        assert.deepEqual(calls.map(call => call.data.request_id).sort(), prepared.items.map(item => item.id).sort());
        assert.ok(calls.every(call => call.url === '/sys/ai/kbs/7/docs' && call.data.batch_id === prepared.batchId));
        assert.deepEqual(changes.filter(change => change.doc).map(change => change.state).sort(), ['pending', 'ready']);
      } finally { client.clear(); }
    });
    await t.test('文档查询仅pending时每3秒刷新，不在隐藏页面持续轮询', async () => {
      const queries = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
      const options = queries.kbDocListOptions(['test', 1, 1], 7, { page: 2, page_size: 20 });
      assert.equal(typeof options.refetchInterval, 'function', '缺少按状态轮询');
      assert.equal(options.refetchInterval({ state: { data: { list: [{ status: 'pending' }] } } }), 3000);
      assert.equal(options.refetchInterval({ state: { data: { list: [{ status: 'failed' }, { status: 'ready' }] } } }), false);
      assert.equal(options.refetchInterval({ state: { data: undefined } }), false);
      assert.equal(options.refetchIntervalInBackground, false);
      assert.notDeepEqual(options.queryKey, queries.kbDocListOptions(['test', 1, 1], 7, { page: 1, page_size: 20 }).queryKey);
    });
    await t.test('单文档状态按ID读取元信息，不扫描列表或拉取正文；原文读取不轮询', async () => {
      const { QueryClient } = await import('@tanstack/react-query');
      const queries = await server.ssrLoadModule('/src/pages/ai/knowledgeDocumentQueries.ts');
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const requests = [];
      request.defaults.adapter = async config => {
        requests.push({ url: config.url, params: config.params });
        return { data: { code: 200, result: { doc: { id: 9, status: 'pending', version: 2 }, content: '' } }, status: 200, statusText: 'OK', headers: {}, config };
      };
      const client = new QueryClient();
      try {
        const scope = ['local-doc-session', 1, 1];
        const options = queries.knowledgeDocumentOptions(scope, 7, 9, true);
        assert.equal((await client.fetchQuery(options)).doc.id, 9);
        assert.deepEqual(requests, [{ url: '/sys/ai/kbs/7/docs/9', params: { meta_only: true } }]);
        assert.equal(options.refetchInterval({ state: { data: { doc: { status: 'pending' } } } }), 3000);
        assert.equal(options.refetchInterval({ state: { data: { doc: { status: 'ready' } } } }), false);
        assert.equal(options.refetchIntervalInBackground, false);
        assert.equal(queries.knowledgeDocumentOptions(scope, 7, 9).refetchInterval({ state: { data: { doc: { status: 'pending' } } } }), false);
        assert.notDeepEqual(options.queryKey, queries.knowledgeDocumentOptions(['other', 1, 1], 7, 9, true).queryKey);
      } finally { client.clear(); }
    });
  } finally { await server.close(); globalThis.window = previousWindow; }
});
