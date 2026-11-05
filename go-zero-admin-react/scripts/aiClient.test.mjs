import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
const storage = () => { const values = new Map(); return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: key => values.delete(key) }; };
const frame = (event, data) => `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;

test('AI流式发送复用鉴权、过期前刷新和取消，普通请求保持默认超时', async () => {
  const originalWindow = globalThis.window;
  const originalFetch = globalThis.fetch;
  const originalNavigator = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
  globalThis.window = { localStorage: storage(), sessionStorage: storage(), addEventListener() {} };
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { locks: { request: async (_name, action) => action() } } });
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { request, apiPost } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const { sendAiMessageStream } = await server.ssrLoadModule('/src/api/aiStream.ts');
    useAuthStore.getState().setSession({ access_token: 'test-access', refresh_token: 'test-refresh', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'test-session' });
    useAuthStore.getState().setPermCodes(['sys:ai:message:send']);
    let captured;
    let refreshes = 0;
    request.defaults.adapter = async config => {
      captured = config;
      const result = config.url === '/sys/auth/refresh'
        ? { access_token: 'test-access-new', refresh_token: 'test-refresh-new', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'test-session' } : {};
      if (config.url === '/sys/auth/refresh') refreshes += 1;
      return { data: { code: 200, result }, status: 200, statusText: 'OK', headers: {}, config };
    };
    const input = { request_id: '00000000-0000-4000-8000-000000000001', conversation_id: 0, content: 'local request' };
    const attempts = [];
    globalThis.fetch = async (url, options) => {
      attempts.push({ url, ...options });
      return new Response(frame('error', { code: 401, msg: 'local expiration' }), { headers: { 'Content-Type': 'text/event-stream' } });
    };
    await assert.rejects(sendAiMessageStream(input, {}), error => error.code === 401);
    assert.equal(attempts.length, 1, '开流后业务401不自动重放模型请求');
    assert.equal(refreshes, 0);
    assert.equal(attempts[0].headers.Authorization, 'Bearer test-access');
    assert.equal(attempts[0].headers.Accept, 'text/event-stream');
    assert.ok(attempts[0].signal instanceof AbortSignal);
    assert.deepEqual(JSON.parse(attempts[0].body), input);
    useAuthStore.setState({ expiresAt: Date.now() / 1000 - 1 });
    await assert.rejects(sendAiMessageStream(input, {}), error => error.code === 401);
    assert.equal(refreshes, 1, '过期会话开流前沿用统一刷新');
    assert.equal(attempts[1].headers.Authorization, 'Bearer test-access-new');
    assert.deepEqual(JSON.parse(attempts[1].body), input);
    const abort = new AbortController();
    abort.abort();
    await assert.rejects(sendAiMessageStream(input, {}, abort.signal), error => error.name === 'AbortError');
    assert.equal(attempts.length, 2, '取消后不发起请求');
    await apiPost('/sys/ordinary', { value: 1 });
    assert.equal(captured.timeout, 30_000);
  } finally {
    await server.close();
    globalThis.window = originalWindow;
    globalThis.fetch = originalFetch;
    if (originalNavigator) Object.defineProperty(globalThis, 'navigator', originalNavigator);
    else delete globalThis.navigator;
  }
});
