import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { createPermissionRefresh } from '#/adapter/business/permission-runtime';
import { registerPermissionRefresh } from '#/router/permission-refresh-events';

import { useAuthStore } from './auth';

const mocks = vi.hoisted(() => ({
  access: { accessToken: 'old-token', setAccessToken: vi.fn() },
  user: { setUserInfo: vi.fn() },
  logout: vi.fn(),
  getUser: vi.fn(),
  replace: vi.fn(),
  warning: vi.fn(),
  clear: vi.fn(),
  login: vi.fn(),
  save: vi.fn(),
}));
vi.mock('vue-router', () => ({
  useRouter: () => ({
    currentRoute: { value: { fullPath: '/account', query: {} } },
    replace: mocks.replace,
  }),
}));
vi.mock('@vben/constants', () => ({ LOGIN_PATH: '/auth/login' }));
vi.mock('@vben/stores', () => ({
  useAccessStore: () => mocks.access,
  useUserStore: () => mocks.user,
  resetAllStores: () => {
    mocks.access.accessToken = '';
  },
}));
vi.mock('antdv-next', () => ({
  notification: { warning: mocks.warning, success: vi.fn() },
}));
vi.mock('#/api', () => ({
  logoutApi: mocks.logout,
  getUserInfoApi: mocks.getUser,
  loginApi: mocks.login,
}));
vi.mock('#/router', () => ({ resetRoutes: vi.fn() }));
vi.mock('#/adapter/business/session', () => ({
  clearSessionCache: mocks.clear,
  safeRedirect: () => '/account',
  saveSession: mocks.save,
  SESSION_HOME: '/_session/home',
  StaleSessionResponseError: class extends Error {},
}));

describe('logout and current-user request races', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
    mocks.access.accessToken = 'old-token';
    mocks.access.setAccessToken.mockImplementation((token: string) => {
      mocks.access.accessToken = token;
    });
  });
  afterEach(() => registerPermissionRefresh(undefined));
  it('login clears the old refresh epoch before installing the new user session', async () => {
    let finish!: (value: { revision: string; fingerprint: string }) => void;
    const commit = vi.fn();
    const coordinator = createPermissionRefresh({
      token: () => mocks.access.accessToken,
      load: () =>
        new Promise<{ revision: string; fingerprint: string }>((resolve) => {
          finish = resolve;
        }),
      build: async (snapshot) => snapshot,
      commit,
    });
    registerPermissionRefresh({
      refresh: () => coordinator.refresh(),
      reset: () => coordinator.reset(),
      repair: vi.fn(),
    });
    const pending = coordinator.refresh();
    const newUser = {
      roles: ['22'],
      realName: 'New role user',
      homePath: '/account',
    };
    mocks.login.mockResolvedValue({ accessToken: 'new-token' });
    mocks.save.mockReturnValue(newUser);
    await useAuthStore().authLogin({ username: 'new-user', password: 'pass' });
    finish({ revision: '99999999999999999', fingerprint: 'old' });
    await pending;
    expect(commit).not.toHaveBeenCalled();
    expect(mocks.access.accessToken).toBe('new-token');
    expect(mocks.user.setUserInfo).toHaveBeenCalledWith(newUser);
    expect(mocks.replace).toHaveBeenCalledWith('/account');
  });
  it('clearSession invalidates pending refresh even if the identical token is restored', async () => {
    let finish!: (value: { revision: string; fingerprint: string }) => void;
    const commit = vi.fn();
    const coordinator = createPermissionRefresh({
      token: () => mocks.access.accessToken,
      load: () =>
        new Promise<{ revision: string; fingerprint: string }>((resolve) => {
          finish = resolve;
        }),
      build: async (snapshot) => snapshot,
      commit,
    });
    registerPermissionRefresh({
      refresh: () => coordinator.refresh(),
      reset: () => coordinator.reset(),
      repair: vi.fn(),
    });
    const pending = coordinator.refresh();
    useAuthStore().clearSession();
    mocks.access.accessToken = 'old-token';
    finish({ revision: '7', fingerprint: 'old' });
    await pending;
    expect(commit).not.toHaveBeenCalled();
  });
  it('revokes the current token and clears the browser session on logout', async () => {
    await useAuthStore().logout();
    expect(mocks.logout).toHaveBeenCalledExactlyOnceWith('old-token');
    expect(mocks.access.accessToken).toBe('');
    expect(mocks.clear).toHaveBeenCalledOnce();
    expect(mocks.replace).toHaveBeenCalledOnce();
    expect(mocks.warning).not.toHaveBeenCalled();
  });
  it('clears local state even when server logout fails', async () => {
    mocks.logout.mockRejectedValueOnce(new Error('network'));
    await useAuthStore().logout(false);
    expect(mocks.access.accessToken).toBe('');
    expect(mocks.clear).toHaveBeenCalledOnce();
    expect(mocks.replace).toHaveBeenCalledOnce();
    expect(mocks.warning).toHaveBeenCalledOnce();
  });
  it('does not erase a newer login when an old logout finishes', async () => {
    let finish!: () => void;
    mocks.logout.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        }),
    );
    const pending = useAuthStore().logout();
    mocks.access.accessToken = 'new-token';
    finish();
    await pending;
    expect(mocks.access.accessToken).toBe('new-token');
    expect(mocks.clear).not.toHaveBeenCalled();
    expect(mocks.replace).not.toHaveBeenCalled();
  });
  it('does not request logout again for an already rejected token', async () => {
    await useAuthStore().logout(true, false);
    expect(mocks.logout).not.toHaveBeenCalled();
    expect(mocks.access.accessToken).toBe('');
  });
  it('clears the browser session without revoking when no token exists', async () => {
    mocks.access.accessToken = '';
    await useAuthStore().logout(false);
    expect(mocks.logout).not.toHaveBeenCalled();
    expect(mocks.clear).toHaveBeenCalledOnce();
    expect(mocks.replace).toHaveBeenCalledExactlyOnceWith({
      path: '/auth/login',
      query: {},
    });
  });
  it('does not update the store with a previous login user response', async () => {
    let finish!: (value: object) => void;
    mocks.getUser.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    const pending = useAuthStore().fetchUserInfo();
    mocks.access.accessToken = 'new-token';
    finish({ username: 'previous-user' });
    await expect(pending).rejects.toThrow(Error);
    expect(mocks.user.setUserInfo).not.toHaveBeenCalled();
  });
});
