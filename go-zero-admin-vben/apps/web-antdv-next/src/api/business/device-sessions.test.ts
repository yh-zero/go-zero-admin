import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  getDeviceSessions,
  getMyDeviceSessions,
  revokeDeviceSession,
  revokeMyDeviceSession,
} from './device-sessions';
const client = vi.hoisted(() => ({ get: vi.fn(), delete: vi.fn() }));
vi.mock('#/api/request', () => ({ requestClient: client }));
describe('device session HTTP contracts', () => {
  beforeEach(() => vi.clearAllMocks());
  it('keeps personal and administrator queries separate', async () => {
    await getMyDeviceSessions({ pageNo: 2, pageSize: 10 });
    expect(client.get).toHaveBeenLastCalledWith('/v1/sys/session/devices', {
      params: { pageNo: 2, pageSize: 10 },
    });
    await getDeviceSessions({ pageNo: 1, pageSize: 20, userId: 7 });
    expect(client.get).toHaveBeenLastCalledWith(
      '/v1/sys/session/admin/devices',
      { params: { pageNo: 1, pageSize: 20, userId: 7 } },
    );
  });
  it('sends the string device identifier in DELETE body, never the global logout API', async () => {
    await revokeMyDeviceSession('device-uuid');
    expect(client.delete).toHaveBeenLastCalledWith('/v1/sys/session/device', {
      data: { id: 'device-uuid' },
    });
    await revokeDeviceSession('other-device-uuid');
    expect(client.delete).toHaveBeenLastCalledWith(
      '/v1/sys/session/admin/device',
      { data: { id: 'other-device-uuid' } },
    );
  });
});
