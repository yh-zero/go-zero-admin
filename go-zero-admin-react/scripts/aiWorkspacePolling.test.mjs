import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { QueryClient, QueryObserver } from '@tanstack/react-query';

test('资料原文与运行结果按可见元信息协调实际请求次数', async t => {
  const previousWindow = globalThis.window;
  const memory = () => { const values = new Map(); return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }; };
  globalThis.window = { localStorage: memory(), sessionStorage: memory(), addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  try {
    const documents = await server.ssrLoadModule('/src/pages/ai/knowledgeDocumentQueries.ts');
    const tasks = await server.ssrLoadModule('/src/pages/ai/workspaceQueries.ts');
    const { request } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    useAuthStore.setState({ token: 'local-poll-test', sessionId: 'poll-session', sessionVersion: 1, departmentVersion: 1, expiresAt: Date.now() / 1000 + 3600, isLoggingOut: false, permissionsReady: true, permCodes: ['sys:ai:kb:list', 'sys:ai:task:list'] });
    const scope = ['poll-session', 1, 1];
    const calls = [];
    let status = 'ready';
    request.defaults.adapter = async config => {
      calls.push({ url: config.url, meta: config.params?.meta_only === true });
      return { data: { code: 200, result: config.url.includes('/docs/') ? { doc: { id: 9, kb_id: 7, status, version: 1, updated_at: 2 }, content: config.params.meta_only ? '' : '原文' } : { task: { id: 11, status, next_step: 1, updated_at: 2 } } }, status: 200, statusText: 'OK', headers: {}, config };
    };
    async function observe(options) {
      const observer = new QueryObserver(client, options);
      let unsubscribe;
      if (options.enabled === false) { unsubscribe = observer.subscribe(() => {}); await new Promise(resolve => setTimeout(resolve, 10)); }
      else await new Promise((resolve, reject) => { unsubscribe = observer.subscribe(result => { if (result.isSuccess) resolve(); if (result.isError) reject(result.error); }); });
      unsubscribe();
    }
    await t.test('ready文档首次只读全文一次，pending消费父列表元信息或单ID轮询', async () => {
      assert.equal(typeof documents.knowledgeDocumentMetadataOptions, 'function');
      for (const [state, parent, expected] of [['ready', undefined, 1], ['pending', { id: 9, status: 'pending' }, 1], ['pending', undefined, 2]]) {
        client.clear(); calls.length = 0; status = state;
        const full = await client.fetchQuery(documents.knowledgeDocumentOptions(scope, 7, 9));
        await observe(documents.knowledgeDocumentMetadataOptions(scope, 7, 9, full.doc, parent));
        assert.equal(calls.length, expected, `${state}/${parent ? '父列表' : '列表外'}`);
        assert.equal(calls.filter(call => !call.meta).length, 1);
        assert.ok(calls.every(call => call.url === '/sys/ai/kbs/7/docs/9'));
      }
    });
    await t.test('终态任务首次只读详情一次，列表外活动任务才读meta', async () => {
      assert.equal(typeof tasks.taskMetadataOptions, 'function');
      for (const [state, parent, expected] of [['succeeded', undefined, 1], ['running', { id: 11, status: 'running' }, 1], ['running', undefined, 2]]) {
        client.clear(); calls.length = 0; status = state;
        const full = await client.fetchQuery(tasks.taskDetailOptions(scope, 11));
        await observe(tasks.taskMetadataOptions(scope, 11, full.task, parent));
        assert.equal(calls.length, expected, `${state}/${parent ? '父列表' : '列表外'}`);
        assert.equal(calls.filter(call => !call.meta).length, 1);
      }
    });
  } finally { client.clear(); await server.close(); globalThis.window = previousWindow; }
});
