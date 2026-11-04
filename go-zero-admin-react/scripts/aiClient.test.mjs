import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

function memoryStorage() {
  const values = new Map();
  return { getItem: (key) => values.get(key) ?? null, setItem: (key, value) => values.set(key, value), removeItem: (key) => values.delete(key) };
}

test('AI长调用沿用鉴权和取消拦截，超时覆盖不改变普通请求', async () => {
  const originalWindow = globalThis.window;
  const originalNavigator = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
  globalThis.window = { localStorage: memoryStorage(), sessionStorage: memoryStorage(), addEventListener() {} };
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { locks: { request: async (_name, action) => action() } } });
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { request, apiPost } = await server.ssrLoadModule('/src/utils/request.ts');
    const { useAuthStore } = await server.ssrLoadModule('/src/stores/auth.ts');
    const { sendAiMessage } = await server.ssrLoadModule('/src/api/ai.ts');
    useAuthStore.getState().setSession({ access_token: 'test-access', refresh_token: 'test-refresh', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'test-session' });
    useAuthStore.getState().setPermCodes(['sys:ai:message:send']);
    let captured;
    request.defaults.adapter = async (config) => {
      captured = config;
      return { data: { code: 200, result: { conversation_id: 1 } }, status: 200, statusText: 'OK', headers: {}, config };
    };
    const input = { request_id: '00000000-0000-4000-8000-000000000001', conversation_id: 0, content: 'local request' };
    assert.equal((await sendAiMessage(input)).conversation_id, 1);
    assert.ok(captured.timeout > 120_000 && captured.timeout <= 130_000, 'AI请求必须覆盖120秒HTTP/RPC上限');
    assert.deepEqual(JSON.parse(captured.data), input);
    assert.equal(captured.headers.get('Authorization'), 'Bearer test-access');
    assert.ok(captured.signal instanceof AbortSignal);
    await apiPost('/sys/ordinary', { value: 1 });
    assert.equal(captured.timeout, 30_000, '普通请求仍保持默认超时');
    let refreshes = 0;
    const attempts = [];
    request.defaults.adapter = async (config) => {
      let result;
      if (config.url === '/sys/auth/refresh') {
        refreshes += 1;
        result = { code: 200, result: { access_token: 'test-access-new', refresh_token: 'test-refresh-new', expires_at: Date.now() / 1000 + 3600, refresh_expires_at: Date.now() / 1000 + 7200, session_id: 'test-session' } };
      } else {
        attempts.push({ timeout: config.timeout, data: JSON.parse(config.data), token: config.headers.get('Authorization') });
        result = attempts.length === 1 ? { code: 401, msg: 'local expiration' } : { code: 200, result: { conversation_id: 2 } };
      }
      return { data: result, status: 200, statusText: 'OK', headers: {}, config };
    };
    assert.equal((await sendAiMessage(input)).conversation_id, 2);
    assert.equal(refreshes, 1);
    assert.equal(attempts.length, 2);
    assert.equal(attempts[0].token, 'Bearer test-access');
    assert.equal(attempts[1].token, 'Bearer test-access-new');
    for (const attempt of attempts) {
      assert.ok(attempt.timeout > 120_000 && attempt.timeout <= 130_000);
      assert.deepEqual(attempt.data, input, '鉴权重试复用原始request_id和消息参数');
    }
  } finally {
    await server.close();
    globalThis.window = originalWindow;
    if (originalNavigator) Object.defineProperty(globalThis, 'navigator', originalNavigator);
    else delete globalThis.navigator;
  }
});
