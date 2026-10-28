import type { PageQuery, PageResult } from './system/types';

import { requestClient } from '#/api/request';

export interface DeviceSession {
  id: string;
  userId: number;
  username: string;
  ip: string;
  userAgent: string;
  createdAt: string;
  expiresAt: string;
  current: boolean;
}
export interface DeviceSessionQuery extends PageQuery {
  userId?: number;
}
export const getMyDeviceSessions = (params: PageQuery) =>
  requestClient.get<PageResult<DeviceSession>>('/v1/sys/session/devices', {
    params,
  });
export const getDeviceSessions = (params: DeviceSessionQuery) =>
  requestClient.get<PageResult<DeviceSession>>(
    '/v1/sys/session/admin/devices',
    { params },
  );
export const revokeMyDeviceSession = (id: string) =>
  requestClient.delete('/v1/sys/session/device', { data: { id } });
export const revokeDeviceSession = (id: string) =>
  requestClient.delete('/v1/sys/session/admin/device', { data: { id } });
