import { nextTick } from 'vue';
import {
  createMemoryHistory,
  createRouter,
  type RouteRecordRaw,
} from 'vue-router';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { installPermissionRefresh } from './permission-refresh';
import {
  notifyPermissionChange,
  refreshCurrentPermissions,
  registerPermissionRefresh,
  resetPermissionRefresh,
} from './permission-refresh-events';

const mocks = vi.hoisted(() => ({
  access: {
    accessToken: 'token',
    isAccessChecked: true,
    accessRoutes: [] as RouteRecordRaw[],
    setAccessCodes: vi.fn(),
    setAccessMenus: vi.fn(),
    setAccessRoutes: vi.fn(),
    setIsAccessChecked: vi.fn(),
  },
  user: { userInfo: { homePath: '/business' }, setUserInfo: vi.fn() },
  tabs: {
    tabs: [],
    visitHistory: { retain: vi.fn() },
    cachedRoutes: new Map(),
    cachedTabs: new Set(),
  },
  snapshot: vi.fn(),
  build: vi.fn(),
}));
vi.mock('@vben/stores', () => ({
  useAccessStore: () => mocks.access,
  useUserStore: () => mocks.user,
  useTabbarStore: () => mocks.tabs,
}));
vi.mock('#/adapter/business/session', () => ({
  SESSION_HOME: '/_session/home',
  readSession: () => ({ user: { userId: '1' } }),
}));
vi.mock('#/api/business/system/permissions', () => ({
  getPermissionSnapshot: mocks.snapshot,
}));
vi.mock('./access', () => ({ generateAccess: mocks.build }));
vi.mock('./routes', () => ({ accessRoutes: [] }));

const component = { render: () => null };
function snapshot(revision: string, code = `rev:${revision}`, authorityId = 1) {
  return {
    revision,
    fingerprint: code,
    menus: [],
    codes: [code],
    authorityId,
    defaultRouter: '',
  };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
async function flush() {
  for (let index = 0; index < 16; index++) await Promise.resolve();
  await nextTick();
}
async function setup() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { name: 'Root', path: '/', component },
      { name: 'FallbackNotFound', path: '/:path(.*)*', component },
    ],
  });
  const refresh = installPermissionRefresh(router);
  await refresh();
  await router.replace('/business');
  return { router, refresh };
}

describe('permission refresh requests with a real memory router', () => {
  let cleanupListeners = () => {};
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.access.accessToken = 'token';
    mocks.access.accessRoutes = [];
    mocks.snapshot.mockResolvedValue(snapshot('1'));
    mocks.build.mockImplementation(async () => ({
      accessibleRoutes: [{ name: 'Business', path: '/business', component }],
      accessibleMenus: [],
    }));
    mocks.access.setAccessRoutes.mockImplementation((routes) => {
      mocks.access.accessRoutes = routes;
    });
    const listeners = vi.spyOn(window, 'addEventListener');
    cleanupListeners = () => {
      for (const [type, listener] of listeners.mock.calls)
        if (type === 'focus' || type === 'storage')
          window.removeEventListener(type, listener);
    };
  });
  afterEach(() => {
    cleanupListeners();
    registerPermissionRefresh(undefined);
    vi.restoreAllMocks();
  });

  it('reads the post-save snapshot instead of reusing a pre-save navigation refresh', async () => {
    const old = deferred<ReturnType<typeof snapshot>>();
    mocks.snapshot
      .mockResolvedValueOnce(snapshot('1'))
      .mockReturnValueOnce(old.promise)
      .mockResolvedValue(snapshot('2'));
    const { refresh } = await setup();
    const oldNavigation = refresh();
    notifyPermissionChange();
    old.resolve(snapshot('1'));
    await oldNavigation;
    await flush();
    expect(mocks.snapshot).toHaveBeenCalledTimes(3);
    expect(mocks.access.setAccessCodes).toHaveBeenLastCalledWith(['rev:2']);
  });

  it('keeps navigation waiting when its old refresh is replaced by a pending permission change', async () => {
    const old = deferred<ReturnType<typeof snapshot>>();
    const changed = deferred<ReturnType<typeof snapshot>>();
    mocks.snapshot
      .mockResolvedValueOnce(snapshot('1'))
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(changed.promise);
    const { refresh } = await setup();
    let navigationFinished = false;
    const navigation = refresh().then(() => {
      navigationFinished = true;
    });
    const change = refreshCurrentPermissions(true);
    old.resolve(snapshot('1'));
    await flush();
    expect(navigationFinished).toBe(false);
    changed.resolve(snapshot('2'));
    await change;
    await navigation;
    expect(navigationFinished).toBe(true);
    expect(mocks.access.setAccessCodes).toHaveBeenLastCalledWith(['rev:2']);
  });

  it('never rolls back a newer forced response when an older response arrives later', async () => {
    const old = deferred<ReturnType<typeof snapshot>>();
    const newer = deferred<ReturnType<typeof snapshot>>();
    mocks.snapshot
      .mockResolvedValueOnce(snapshot('1'))
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(newer.promise);
    await setup();
    const first = refreshCurrentPermissions(true);
    const second = refreshCurrentPermissions(true);
    expect(mocks.snapshot).toHaveBeenCalledTimes(3);
    newer.resolve(snapshot('3'));
    await second;
    old.resolve(snapshot('2'));
    await first;
    expect(mocks.access.setAccessCodes.mock.calls).toEqual([
      [['rev:1']],
      [['rev:3']],
    ]);
  });

  it('drops an old session response after reset and accepts the new session lower revision', async () => {
    const old = deferred<ReturnType<typeof snapshot>>();
    const current = deferred<ReturnType<typeof snapshot>>();
    mocks.snapshot
      .mockResolvedValueOnce(snapshot('1'))
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(current.promise);
    const { refresh } = await setup();
    const previous = refresh();
    resetPermissionRefresh();
    mocks.access.accessToken = 'new-token';
    const next = refresh();
    old.resolve(snapshot('999', 'old-session'));
    await flush();
    expect(mocks.access.setAccessCodes).toHaveBeenCalledTimes(1);
    current.resolve(snapshot('1', 'new-session', 2));
    await next;
    await previous;
    expect(mocks.snapshot).toHaveBeenLastCalledWith('new-token');
    expect(mocks.access.setAccessCodes).toHaveBeenLastCalledWith([
      'new-session',
    ]);
  });

  it('shares ordinary focus reads but forces a fresh read for a cross-tab permission change', async () => {
    const old = deferred<ReturnType<typeof snapshot>>();
    mocks.snapshot
      .mockResolvedValueOnce(snapshot('1'))
      .mockReturnValueOnce(old.promise)
      .mockResolvedValue(snapshot('2'));
    const { refresh } = await setup();
    const pending = refresh();
    window.dispatchEvent(new Event('focus'));
    expect(mocks.snapshot).toHaveBeenCalledTimes(2);
    window.dispatchEvent(
      new StorageEvent('storage', { key: 'business-permission-change' }),
    );
    expect(mocks.snapshot).toHaveBeenCalledTimes(3);
    old.resolve(snapshot('1'));
    await pending;
    await flush();
    expect(mocks.access.setAccessCodes).toHaveBeenLastCalledWith(['rev:2']);
  });
});
