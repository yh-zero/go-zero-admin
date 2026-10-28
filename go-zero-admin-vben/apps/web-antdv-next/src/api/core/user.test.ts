import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getUserInfoApi } from './user';

const mocks = vi.hoisted(() => ({
  access: { accessToken: 'old-token' },
  getCurrent: vi.fn(),
  refresh: vi.fn(),
}));
vi.mock('@vben/stores', () => ({ useAccessStore: () => mocks.access }));
vi.mock('#/api/business/account', () => ({ getCurrentUser: mocks.getCurrent }));
vi.mock('#/adapter/business/session', () => ({
  refreshSessionUser: mocks.refresh,
  StaleSessionResponseError: class extends Error {},
}));

describe('current-user response token binding', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.access.accessToken = 'old-token';
  });
  it('does not bind old profile data to the replacement token', async () => {
    let finish!: (value: object) => void;
    mocks.getCurrent.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    const pending = getUserInfoApi();
    mocks.access.accessToken = 'new-token';
    finish({ ID: 1, userName: 'old-user' });
    await expect(pending).rejects.toThrow();
    expect(mocks.refresh).not.toHaveBeenCalled();
  });
  it('converts an old failed request into a stale response without expiring the new login', async () => {
    let fail!: (error: object) => void;
    mocks.getCurrent.mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          fail = reject;
        }),
    );
    const pending = getUserInfoApi();
    mocks.access.accessToken = 'new-token';
    fail({ response: { status: 401 } });
    await expect(pending).rejects.toBeInstanceOf(Error);
    expect(mocks.access.accessToken).toBe('new-token');
    expect(mocks.refresh).not.toHaveBeenCalled();
  });
  it('refreshes the profile using the token that sent the request', async () => {
    const user = { ID: 1, userName: 'user' };
    mocks.getCurrent.mockResolvedValueOnce(user);
    await getUserInfoApi();
    expect(mocks.refresh).toHaveBeenCalledWith(user, 'old-token');
  });
});
