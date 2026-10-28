import { beforeEach, describe, expect, it, vi } from 'vitest';

import { StaleSessionResponseError } from '#/adapter/business/session';

import {
  createUser,
  deleteUser,
  getUserResourcePreview,
  getUsers,
  resetUserPassword,
  transferUserResources,
  updateUser,
} from './user';

const mocks = vi.hoisted(() => ({
  access: { accessToken: 'old-token' },
  sent: vi.fn(),
}));
vi.mock('@vben/stores', () => ({ useAccessStore: () => mocks.access }));
vi.mock('@vben/hooks', () => ({ useAppConfig: () => ({ apiURL: '/api' }) }));
vi.mock('@vben/preferences', () => ({
  preferences: { app: { locale: 'zh-CN' } },
}));
vi.mock('#/store', () => ({ useAuthStore: () => ({ logout: vi.fn() }) }));
vi.mock('antdv-next', () => ({ message: { error: vi.fn() } }));
type Options = {
  headers?: Record<string, unknown>;
  data?: unknown;
  params?: unknown;
};
vi.mock('@vben/request', () => ({
  defaultResponseInterceptor: () => ({}),
  errorMessageResponseInterceptor: () => ({}),
  RequestClient: class {
    handlers: Array<{
      fulfilled: (config: {
        headers: Record<string, unknown>;
      }) => Promise<unknown>;
    }> = [];
    addRequestInterceptor(handler: (typeof this.handlers)[number]) {
      this.handlers.push(handler);
    }
    addResponseInterceptor() {}
    delete(path: string, options?: Options) {
      return this.dispatch('DELETE', path, options?.data, options);
    }
    async dispatch(
      method: string,
      path: string,
      data: unknown,
      options?: Options,
    ) {
      // Exercise the real request.ts interceptor after the UI's synchronous guard.
      await Promise.resolve();
      const config = { headers: { ...options?.headers } };
      for (const handler of this.handlers) await handler.fulfilled(config);
      mocks.sent(method, path, data, config.headers);
    }
    get(path: string, options?: Options) {
      return this.dispatch('GET', path, options?.params, options);
    }
    post(path: string, data: unknown, options?: Options) {
      return this.dispatch('POST', path, data, options);
    }
    put(path: string, data: unknown, options?: Options) {
      return this.dispatch('PUT', path, data, options);
    }
  },
}));
const user = {
  userName: 'new',
  passWord: 'before123',
  nickName: 'New',
  enable: 1,
  authorityId: 801,
  authorityIds: [801],
};
const transfer = {
  userId: 2,
  targetUserId: 3,
  includeFiles: true,
  includeAI: false,
};
const operations = [
  ['register', () => createUser(user, 'old-token')],
  [
    'profile-and-roles',
    () => updateUser({ ID: 2, authorityIds: [802] }, 'old-token'),
  ],
  ['delete', () => deleteUser(2, 'old-token')],
  ['reset-password', () => resetUserPassword(2, 'old-token')],
  ['transfer-resources', () => transferUserResources(transfer, 'old-token')],
  ['read-resources', () => getUserResourcePreview(2, 'old-token')],
  ['read-users', () => getUsers({ pageNo: 1, pageSize: 10 }, 'old-token')],
] as const;
describe('user API fixed session at asynchronous network dispatch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.access.accessToken = 'old-token';
  });
  it.each(operations)(
    'rejects stale %s payload before sending under a new actor',
    async (_name, invoke) => {
      const pending = invoke();
      mocks.access.accessToken = 'new-token';
      await expect(pending).rejects.toBeInstanceOf(StaleSessionResponseError);
      expect(mocks.sent).not.toHaveBeenCalled();
    },
  );
  it('preserves selected transfer payload and fixed header while the actor remains current', async () => {
    await transferUserResources(transfer, 'old-token');
    expect(mocks.sent).toHaveBeenCalledWith(
      'POST',
      '/v1/sys/user/resources/transfer',
      transfer,
      expect.objectContaining({ Authorization: 'Bearer old-token' }),
    );
  });
  it('preserves existing calls without an optional token', async () => {
    const pending = deleteUser(2);
    mocks.access.accessToken = 'new-token';
    await pending;
    expect(mocks.sent).toHaveBeenCalledWith(
      'DELETE',
      '/v1/sys/deleteUser',
      { userId: 2 },
      expect.objectContaining({ Authorization: 'Bearer new-token' }),
    );
  });
});
