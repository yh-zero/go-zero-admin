import test, { after } from 'node:test';
import assert from 'node:assert/strict';
import axios from 'axios';
import { createServer } from 'vite';

const now = 1_800_000_000;
const session = (suffix = 'old', expiresAt = now + 3600, sessionId = 'fixture-session') => ({
  access_token: `fixture-${suffix}-access`, refresh_token: `fixture-${suffix}-refresh`,
  expires_at: expiresAt, refresh_expires_at: now + 7200, session_id: sessionId,
});
const response = (config, result = {}, code = 200) => ({
  data: { code, msg: code === 200 ? '成功' : 'fixture request rejected', result },
  status: 200, statusText: 'OK', headers: {}, config,
});
function deferred() {
  let resolve;
  const promise = new Promise(done => { resolve = done; });
  return { promise, resolve };
}

// 每个模块图保留独立store、刷新单航班和同步节流，模拟共享存储的同源标签页。
async function createTabs() {
  const originalWindow = globalThis.window;
  const originalNavigator = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
  const originalNow = Date.now;
  const values = new Map(), listeners = [], servers = [], tabs = [];
  const storage = {
    getItem: key => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    removeItem: key => values.delete(key),
  };
  globalThis.window = { localStorage: storage, sessionStorage: storage,
    addEventListener: (name, listener) => { if (name === 'storage') listeners.push(listener); } };
  let queue = Promise.resolve(), beforeLock = () => {};
  Object.defineProperty(globalThis, 'navigator', { configurable: true, value: { locks: {
    request: (_name, locked) => {
      const result = queue.then(async () => { await beforeLock(); return locked(); });
      queue = result.catch(() => undefined);
      return result;
    },
  } } });
  function persist(value) {
    storage.setItem('go-zero-auth', JSON.stringify({ state: {
      token: value.access_token, refreshToken: value.refresh_token, expiresAt: value.expires_at,
      refreshExpiresAt: value.refresh_expires_at, sessionId: value.session_id,
    }, version: 0 }));
  }
  async function close() {
    Date.now = originalNow;
    for (const tab of tabs) { tab.cancelPendingRequests(); tab.setRequestErrorReporter(); }
    await Promise.all(servers.map(server => server.close()));
    globalThis.window = originalWindow;
    if (originalNavigator) Object.defineProperty(globalThis, 'navigator', originalNavigator);
    else delete globalThis.navigator;
  }
  try {
    for (let index = 0; index < 2; index++) {
      const server = await createServer({ configFile: false, envFile: false, logLevel: 'error',
        server: { middlewareMode: true, watch: null, ws: false } });
      servers.push(server);
      const auth = await server.ssrLoadModule('/src/stores/auth.ts');
      const transport = await server.ssrLoadModule('/src/utils/request.ts');
      const lifecycle = await server.ssrLoadModule('/src/utils/requestLifecycle.ts');
      const sessions = await server.ssrLoadModule('/src/utils/session.ts');
      tabs.push({ server, ...auth, ...transport, ...lifecycle, ...sessions });
    }
    return { tabs, storage, persist, close, resumeClock: () => { Date.now = originalNow; },
      reset: async () => {
        await queue;
        beforeLock = () => {};
        Date.now = () => now * 1000;
        for (const tab of tabs) { tab.cancelPendingRequests(); tab.setRequestErrorReporter(); tab.useAuthStore.getState().logout(); }
        values.clear();
        for (const tab of tabs) tab.syncAuthFromStorage(true);
      },
      onLock: callback => { beforeLock = callback; },
      emitStorage: (key = 'go-zero-auth') => listeners.forEach(listener => listener({ storageArea: storage, key })),
    };
  } catch (error) { await close(); throw error; }
}

let environment;
after(async () => { if (environment) await environment.close(); });
async function withTabs(count, action) {
  environment ??= await createTabs();
  await environment.reset();
  try { await action({ ...environment, tabs: environment.tabs.slice(0, count) }); }
  finally { environment.resumeClock(); }
}

test('两个标签页同时过期只轮换一次，等待锁的请求采用已轮换会话', async () => {
  await withTabs(2, async ({ tabs, storage }) => {
    const expired = session('old', now - 1), rotated = session('new');
    const calls = [];
    for (const tab of tabs) {
      tab.useAuthStore.getState().setSession(expired);
      tab.syncAuthFromStorage(true);
      tab.request.defaults.adapter = async config => {
        calls.push({ url: config.url, authorization: config.headers.get('Authorization') });
        if (config.url === '/sys/auth/refresh') {
          assert.equal(JSON.parse(config.data).refresh_token, 'fixture-old-refresh');
          if (calls.filter(call => call.url === '/sys/auth/refresh').length > 1) return response(config, undefined, 401);
          return response(config, rotated);
        }
        return response(config, { readable: config.headers.get('Authorization') === 'Bearer fixture-new-access' });
      };
    }
    assert.deepEqual(await Promise.all(tabs.map(tab => tab.apiGet('/sys/profile'))), [{ readable: true }, { readable: true }]);
    assert.equal(calls.filter(call => call.url === '/sys/auth/refresh').length, 1);
    assert.equal(JSON.parse(storage.getItem('go-zero-auth')).state.token, 'fixture-new-access');
  });
});

test('取得清会话锁后重新核对令牌，迟到的旧401不能清除已轮换会话', async () => {
  await withTabs(1, async ({ tabs: [tab], persist, storage, onLock }) => {
    tab.useAuthStore.getState().setSession(session());
    tab.syncAuthFromStorage(true);
    onLock(() => persist(session('new')));
    tab.request.defaults.adapter = async config => response(config, undefined, 401);
    await assert.rejects(tab.request.get('/sys/profile', { authRetried: true }), error => axios.isCancel(error));
    assert.equal(tab.useAuthStore.getState().token, 'fixture-new-access');
    assert.equal(JSON.parse(storage.getItem('go-zero-auth')).state.token, 'fixture-new-access');
  });
});

test('storage轮换事件不受请求读取节流影响，立即更新当前会话', async () => {
  await withTabs(1, async ({ tabs: [tab], persist, emitStorage }) => {
    tab.useAuthStore.getState().setSession(session());
    tab.syncAuthFromStorage(true);
    persist(session('new'));
    emitStorage();
    assert.equal(tab.useAuthStore.getState().token, 'fixture-new-access');
  });
});

for (const eventKey of ['go-zero-auth', null]) {
  test(`storage退出事件立即取消在途请求并清除权限（key=${eventKey}）`, async () => {
    await withTabs(1, async ({ tabs: [tab], storage, emitStorage }) => {
      tab.useAuthStore.getState().setSession(session());
      tab.useAuthStore.getState().setPermCodes(['sys:user:edit']);
      tab.syncAuthFromStorage(true);
      const pending = tab.trackRequest();
      storage.removeItem('go-zero-auth');
      emitStorage(eventKey);
      assert.equal(tab.useAuthStore.getState().token, '');
      assert.equal(tab.useAuthStore.getState().permissionsReady, false);
      assert.equal(pending.aborted, true);
    });
  });
}

test('退出在会话锁内采用另一标签页的新令牌，撤销成功后清除共享会话', async () => {
  await withTabs(1, async ({ tabs: [tab], persist, storage, onLock }) => {
    tab.useAuthStore.getState().setSession(session());
    tab.syncAuthFromStorage(true);
    let firstLock = true;
    onLock(() => { if (firstLock) { firstLock = false; persist(session('new')); } });
    const calls = [];
    tab.request.defaults.adapter = async config => {
      calls.push({ url: config.url, authorization: config.headers.get('Authorization') });
      return response(config);
    };
    await tab.logoutSession();
    assert.deepEqual(calls, [{ url: '/sys/auth/logout', authorization: 'Bearer fixture-new-access' }]);
    assert.equal(tab.useAuthStore.getState().token, '');
    assert.equal(JSON.parse(storage.getItem('go-zero-auth')).state.token, '');
  });
});

test('旧会话响应迟到时不清除另一账号的新会话', async () => {
  await withTabs(1, async ({ tabs: [tab], persist, storage }) => {
    tab.useAuthStore.getState().setSession(session());
    tab.syncAuthFromStorage(true);
    const started = deferred(), finish = deferred();
    tab.request.defaults.adapter = async config => { started.resolve(); await finish.promise; return response(config, undefined, 401); };
    const attempt = tab.request.get('/sys/profile', { authRetried: true });
    const rejected = assert.rejects(attempt, error => axios.isCancel(error));
    await started.promise;
    persist(session('other', now + 3600, 'fixture-other-session'));
    finish.resolve();
    await rejected;
    assert.equal(tab.useAuthStore.getState().sessionId, 'fixture-other-session');
    assert.equal(JSON.parse(storage.getItem('go-zero-auth')).state.token, 'fixture-other-access');
  });
});

test('连续取消20次请求后，会话清理不再触及任何已结束请求', async () => {
  await withTabs(1, async ({ tabs: [tab] }) => {
    tab.useAuthStore.getState().setSession(session());
    const NativeAbortController = globalThis.AbortController;
    const tracked = [];
    globalThis.AbortController = class extends NativeAbortController {
      constructor() { super(); tracked.push(this); }
    };
    try {
      for (let index = 0; index < 20; index++) {
        const started = deferred(), controller = new NativeAbortController();
        tab.request.defaults.adapter = config => {
          started.resolve();
          return new Promise((_, reject) => config.signal.addEventListener('abort', () => reject(new axios.CanceledError('fixture navigation', config)), { once: true }));
        };
        const rejected = assert.rejects(tab.apiGet('/sys/profile', undefined, { signal: controller.signal }), error => axios.isCancel(error));
        await started.promise;
        controller.abort();
        await rejected;
      }
      assert.equal(tracked.length, 20);
      tab.cancelPendingRequests();
      assert.equal(tracked.filter(controller => controller.signal.aborted).length, 0, '已取消的20项请求必须全部解除跟踪');
    } finally { globalThis.AbortController = NativeAbortController; }
  });
});

for (const { status, code, message } of [
  { status: 403, code: undefined, message: /没有访问权限/ },
  { status: 500, code: undefined, message: /HTTP 500/ },
  { status: undefined, code: undefined, message: /无法连接服务/ },
  { status: undefined, code: 'ECONNABORTED', message: /请求超时/ },
]) {
  test(`真实请求层分类HTTP/网络错误（status=${status}, code=${code}）`, async () => {
    await withTabs(1, async ({ tabs: [tab] }) => {
      tab.request.defaults.adapter = async config => {
        throw new axios.AxiosError('fixture transport failure', code, config, undefined,
          status === undefined ? undefined : { data: '', status, statusText: 'fixture error', headers: {}, config });
      };
      await assert.rejects(tab.apiGet('/ping'), error => error.code === (status ?? 500) && message.test(error.message));
    });
  });
}

test('上传采用50秒专项预算，普通读取仍保持30秒默认预算', async () => {
  await withTabs(1, async ({ tabs: [tab] }) => {
    const { uploadFile } = await tab.server.ssrLoadModule('/src/api/file.ts');
    tab.useAuthStore.getState().setSession(session());
    tab.useAuthStore.getState().setPermCodes(['sys:file:upload']);
    const file = { id: 7, bucket: 'fixture', object_key: 'fixture/7.txt', original_name: 'sample.txt', ext: '.txt',
      mime: 'text/plain', size: 1, visibility: 'private', ref_count: 0, uploader_id: 1, created_at: now };
    const calls = [];
    tab.request.defaults.adapter = async config => {
      calls.push({ url: config.url, timeout: config.timeout });
      return response(config, config.url === '/sys/file/upload' ? { file } : {});
    };
    assert.deepEqual(await uploadFile({ name: 'sample.txt', content_base64: 'eA==', scene: 'common', visibility: 'private' }), file);
    await tab.apiGet('/sys/profile');
    assert.deepEqual(calls, [{ url: '/sys/file/upload', timeout: 50_000 }, { url: '/sys/profile', timeout: 30_000 }]);
  });
});
