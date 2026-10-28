import { beforeEach, describe, expect, it, vi } from 'vitest';

import { StaleSessionResponseError } from '../adapter/business/session';
import { saveAuthorityMenus } from './business/system/authority';
import { savePolicies, savePolicyApiIds } from './business/system/casbin';
import { saveAuthorityButtons } from './business/system/menu';
import { updateRoleDataScope } from './business/system/organization';
import { applyPermissionRollback } from './business/system/permissions';
const mocks = vi.hoisted(() => ({
  access: { accessToken: 'a' },
  sent: vi.fn(),
}));
vi.mock('@vben/stores', () => ({ useAccessStore: () => mocks.access }));
vi.mock('@vben/hooks', () => ({ useAppConfig: () => ({ apiURL: '/api' }) }));
vi.mock('@vben/preferences', () => ({
  preferences: { app: { locale: 'zh-CN' } },
}));
vi.mock('#/store', () => ({ useAuthStore: () => ({ logout: vi.fn() }) }));
vi.mock('antdv-next', () => ({ message: { error: vi.fn() } }));
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
    async post(
      path: string,
      data: unknown,
      options?: { headers: Record<string, unknown> },
    ) {
      // Axios request interceptors execute asynchronously, after the UI's guard.
      await Promise.resolve();
      const config = { headers: { ...options?.headers } };
      for (const handler of this.handlers) await handler.fulfilled(config);
      mocks.sent(path, data, config.headers);
    }
    async put(
      path: string,
      data: unknown,
      options?: { headers: Record<string, unknown> },
    ) {
      return this.post(path, data, options);
    }
  },
}));
describe('fixed permission token at request dispatch', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.access.accessToken = 'a';
  });
  it('rejects an old operation if account changes between API invocation and async interceptor', async () => {
    const pending = saveAuthorityMenus(22, [1], '7', 'a');
    mocks.access.accessToken = 'b';
    await expect(pending).rejects.toBeInstanceOf(StaleSessionResponseError);
    expect(mocks.sent).not.toHaveBeenCalled();
  });
  it.each<[string, () => Promise<unknown>]>([
    ['button', () => saveAuthorityButtons(22, [2], '7', 'a')],
    ['apiIds', () => savePolicyApiIds(22, [3], '7', 'a')],
    [
      'apiPaths',
      () => savePolicies(22, [{ method: 'GET', path: '/test' }], '7', 'a'),
    ],
    [
      'dataScope',
      () =>
        updateRoleDataScope(
          {
            authorityId: 22,
            scope: 'self',
            departmentIds: [],
            expectedRevision: '7',
          },
          'a',
        ),
    ],
    ['rollback', () => applyPermissionRollback(3, '7', 'preview', 'a')],
  ])(
    'rejects stale fixed tokens for %s dispatch too',
    async (_kind, action) => {
      const pending = action();
      mocks.access.accessToken = 'b';
      await expect(pending).rejects.toBeInstanceOf(StaleSessionResponseError);
      expect(mocks.sent).not.toHaveBeenCalled();
    },
  );
  it('sends the captured token for a still-current operation', async () => {
    await saveAuthorityMenus(22, [1], '7', 'a');
    expect(mocks.sent).toHaveBeenCalledWith(
      '/v1/sys/authority/addAuthorityMenu',
      { authorityId: 22, menuIds: '1', expectedRevision: '7' },
      expect.objectContaining({ Authorization: 'Bearer a' }),
    );
  });
  it('retains ordinary request token injection when no fixed token is supplied', async () => {
    const pending = saveAuthorityMenus(22, [1], '7');
    mocks.access.accessToken = 'b';
    await pending;
    expect(mocks.sent.mock.calls[0]?.[2].Authorization).toBe('Bearer b');
  });
});
