import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

const storage = () => {
  const values = new Map();
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
};

test('知识库摄取等待覆盖业务截止时间，普通管理请求保留默认超时', async () => {
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
      return { data: { code: 200, result: { doc: { id: 9, status: 'ready' } } }, status: 200, statusText: 'OK', headers: {}, config };
    };
    const content = { title: '本地摄取测试', content: '只通过请求适配器检查，不调用模型。' };
    assert.equal((await api.createAiKbDoc(7, content)).doc.status, 'ready');
    assert.equal(requests[0].timeout, 125_000);
    assert.equal(requests[0].url, '/sys/ai/kbs/7/docs');
    assert.deepEqual(JSON.parse(requests[0].data), content);
    await api.retryAiKbDoc(7, 9);
    assert.equal(requests[1].timeout, 125_000);
    assert.equal(requests[1].url, '/sys/ai/kbs/7/docs/9/retry');
    await api.fetchAiKbDocs(7);
    assert.equal(requests[2].timeout, 30_000);
    await api.fetchAiTools();
    assert.equal(requests[3].timeout, 30_000);
    await api.updateAiKbDoc(7, 9, { ...content, expected_version: 2 });
    assert.equal(requests[4].timeout, 125_000);
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
    await t.test('pending按最近更新时间保护120秒，failed可重试、ready不可重试', () => {
      assert.equal(typeof documents.canRetryDocument, 'function', '缺少重试保护');
      const pending = { id: 9, status: 'pending', created_at: 1, updated_at: 100 };
      assert.equal(documents.canRetryDocument(pending, 219999), false);
      assert.equal(documents.canRetryDocument(pending, 220000), true);
      assert.equal(documents.canRetryDocument({ ...pending, status: 'failed' }, 100001), true);
      assert.equal(documents.canRetryDocument({ ...pending, status: 'ready' }, 999999), false);
    });
    await t.test('创建失败结果保留doc供下次retry；pending保护期不发送重试请求', async () => {
      assert.equal(typeof documents.documentWriteOptions, 'function', '缺少可恢复的文档提交');
      const { QueryClient, MutationObserver } = await import('@tanstack/react-query');
      const { request } = await server.ssrLoadModule('/src/utils/request.ts');
      const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
      useAuthStore.setState({ token: 'local-doc-test', sessionId: 'local-doc-session', sessionVersion: 1, departmentVersion: 1,
        expiresAt: Date.now() / 1000 + 3600, permissionsReady: true, permCodes: ['sys:ai:kb:edit'], isLoggingOut: false });
      const client = new QueryClient();
      const responses = ['failed', 'pending', 'ready'];
      const calls = [];
      request.defaults.adapter = async config => {
        calls.push(config.url);
        return { data: { code: 200, result: { doc: { id: 9, status: responses.shift(), error: '索引服务暂不可用', updated_at: Math.floor(Date.now() / 1000) } } },
          status: 200, statusText: 'OK', headers: {}, config };
      };
      try {
        const mutation = new MutationObserver(client, documents.documentWriteOptions(7));
        const values = { title: '文档', content: '正文' };
        const failed = (await mutation.mutate({ values })).doc;
        assert.equal(failed.status, 'failed');
        assert.equal(failed.id, 9);
        const pending = (await mutation.mutate({ values, doc: failed })).doc;
        assert.equal(pending.status, 'pending');
        await assert.rejects(mutation.mutate({ values, doc: pending }), /等待/);
        assert.equal(calls.length, 2);
        const recovered = (await mutation.mutate({ values, doc: { ...pending, updated_at: Math.floor(Date.now() / 1000) - 121 } })).doc;
        assert.equal(recovered.status, 'ready');
        assert.deepEqual(calls, ['/sys/ai/kbs/7/docs', '/sys/ai/kbs/7/docs/9/retry', '/sys/ai/kbs/7/docs/9/retry']);
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
