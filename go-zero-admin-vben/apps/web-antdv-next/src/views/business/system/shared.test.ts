import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { registerPermissionRefresh } from '#/router/permission-refresh-events';

import { refreshRoleAccess } from './shared';

const mocks = vi.hoisted(() => ({
  user: { userInfo: { roles: ['888'] } },
  reload: vi.fn(),
  success: vi.fn(),
  refresh: vi.fn().mockResolvedValue(undefined),
}));
vi.mock('@vben/stores', () => ({
  useUserStore: () => mocks.user,
  useAccessStore: vi.fn(),
}));
vi.mock('antdv-next', () => ({
  message: { success: mocks.success },
  Modal: {},
}));

describe('role permission refresh', () => {
  beforeEach(() => {
    registerPermissionRefresh({
      refresh: mocks.refresh,
      reset: vi.fn(),
      repair: vi.fn(),
    });
  });
  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
    registerPermissionRefresh(undefined);
  });
  it('preserves the working page when updating another role', () => {
    vi.stubGlobal('window', { location: { reload: mocks.reload } });
    refreshRoleAccess(889);
    expect(mocks.reload).not.toHaveBeenCalled();
    expect(mocks.refresh).toHaveBeenCalledOnce();
    expect(localStorage.getItem('business-permission-change')).toBeTruthy();
    expect(mocks.success).toHaveBeenCalledWith('授权已保存');
  });
  it('regenerates menus and permission codes when the current role changes', () => {
    vi.stubGlobal('window', { location: { reload: mocks.reload } });
    refreshRoleAccess(888);
    expect(mocks.reload).not.toHaveBeenCalled();
    expect(mocks.refresh).toHaveBeenCalledOnce();
  });
});
