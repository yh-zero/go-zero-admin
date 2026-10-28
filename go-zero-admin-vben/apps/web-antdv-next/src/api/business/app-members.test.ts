import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  getAppGrants,
  getAppPlans,
  getAppSessions,
  getAppUsers,
  grantAppMembership,
  revokeAppGrant,
  revokeAppSession,
  updateAppUserAvatar,
  updateAppUserStatus,
} from './app-members';

const client = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}));
vi.mock('#/api/request', () => ({ requestClient: client }));
const pinned = { headers: { Authorization: 'Bearer admin-token' } };
describe('app administration HTTP contract', () => {
  beforeEach(() => vi.clearAllMocks());
  it('pins all list reads to the initiating administrator and preserves filters', async () => {
    await getAppUsers(
      { pageNo: 2, pageSize: 20, keyword: 'mail', status: 'pending' },
      'admin-token',
    );
    expect(client.get).toHaveBeenLastCalledWith('/v1/sys/app-members/users', {
      ...pinned,
      params: { pageNo: 2, pageSize: 20, keyword: 'mail', status: 'pending' },
    });
    await getAppSessions(7, 'admin-token');
    expect(client.get).toHaveBeenLastCalledWith(
      '/v1/sys/app-members/sessions',
      { ...pinned, params: { userId: 7 } },
    );
    await getAppPlans('admin-token');
    expect(client.get).toHaveBeenLastCalledWith(
      '/v1/sys/app-members/plans',
      pinned,
    );
    await getAppGrants({ pageNo: 1, pageSize: 10, userId: 7 }, 'admin-token');
    expect(client.get).toHaveBeenLastCalledWith('/v1/sys/app-members/grants', {
      ...pinned,
      params: { pageNo: 1, pageSize: 10, userId: 7 },
    });
  });
  it('keeps status, device revocation and grant revocation contracts distinct', async () => {
    await updateAppUserStatus(
      { userId: 7, status: 'disabled', reason: '违规' },
      'admin-token',
    );
    expect(client.put).toHaveBeenLastCalledWith(
      '/v1/sys/app-members/status',
      { userId: 7, status: 'disabled', reason: '违规' },
      pinned,
    );
    await revokeAppSession('session-id', 'admin-token');
    expect(client.delete).toHaveBeenLastCalledWith(
      '/v1/sys/app-members/session',
      { ...pinned, data: { id: 'session-id' } },
    );
    await revokeAppGrant({ id: 17, reason: '撤销' }, 'admin-token');
    expect(client.delete).toHaveBeenLastCalledWith(
      '/v1/sys/app-members/grant',
      { ...pinned, data: { id: 17, reason: '撤销' } },
    );
  });
  it('preserves the grant request id supplied by the retryable form', async () => {
    const data = {
      userId: 7,
      planId: 1,
      days: 30,
      requestId: 'request-uuid',
      reason: '人工开通',
    };
    await grantAppMembership(data, 'admin-token');
    expect(client.post).toHaveBeenCalledWith(
      '/v1/sys/app-members/grants',
      data,
      pinned,
    );
  });
  it.each(['https://files.example.test/avatar.png', ''])(
    'pins the avatar-only update and preserves explicit clear: %s',
    async (avatarUrl) => {
      await updateAppUserAvatar({ userId: 7, avatarUrl }, 'admin-token');
      expect(client.put).toHaveBeenCalledExactlyOnceWith(
        '/v1/sys/app-members/avatar',
        { userId: 7, avatarUrl },
        pinned,
      );
    },
  );
});
