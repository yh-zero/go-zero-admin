import { beforeEach, describe, expect, it, vi } from 'vitest';

import { logoutApi } from './auth';

const mocks = vi.hoisted(() => ({ post: vi.fn() }));
vi.mock('#/api/request', () => ({
  baseRequestClient: { post: mocks.post },
  requestClient: {},
}));
vi.mock(
  '#/adapter/business/session',
  async () => await import('../../adapter/business/session'),
);

describe('server logout result handling', () => {
  beforeEach(() => vi.clearAllMocks());
  it('treats an already revoked token as a completed logout', async () => {
    mocks.post.mockRejectedValueOnce({
      response: { status: 401, data: { code: 100003 } },
    });
    await expect(logoutApi('expired-token')).resolves.toBeUndefined();
  });
  it('rejects a business failure even when HTTP status was 200', async () => {
    mocks.post.mockResolvedValueOnce({
      data: { code: 100001, message: 'Temporary failure' },
    });
    await expect(logoutApi('token')).rejects.toThrow('Temporary failure');
  });
  it('does not swallow network failures', async () => {
    mocks.post.mockRejectedValueOnce(new Error('network'));
    await expect(logoutApi('token')).rejects.toThrow('network');
  });
});
