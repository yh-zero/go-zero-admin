import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('已取消的工作区查询不发送GET且普通读取保持30秒超时', async () => {
  const previousWindow = globalThis.window;
  const values = new Map();
  const storage = { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) };
  globalThis.window = { localStorage: storage, sessionStorage: storage, addEventListener() {} };
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { request } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const { fetchAiKbs } = await server.ssrLoadModule('/src/api/ai.ts');
    useAuthStore.getState().setSession({ access_token: 'workspace-test-access', refresh_token: 'workspace-test-refresh', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'workspace-test-session' });
    useAuthStore.getState().setPermCodes(['sys:ai:kb:list']);
    const requests = [];
    request.defaults.adapter = async config => {
      requests.push({ timeout: config.timeout, params: config.params });
      return { data: { code: 200, result: { list: [], total: 0, capabilities: { retrieval_configured: false, analysis_configured: false, ingestion_mode: 'background' } } }, status: 200, statusText: 'OK', headers: {}, config };
    };
    const controller = new AbortController();
    controller.abort();
    await assert.rejects(fetchAiKbs({ page: 1 }, controller.signal), error => error.code === 'ERR_CANCELED');
    assert.equal(requests.length, 0, '已取消的查询不能到达传输适配器');
    const result = await fetchAiKbs({ page: 1, query: '发布' });
    assert.equal(result.total, 0);
    assert.deepEqual(requests, [{ timeout: 30_000, params: { page: 1, query: '发布' } }]);
  } finally {
    await server.close();
    globalThis.window = previousWindow;
  }
});
