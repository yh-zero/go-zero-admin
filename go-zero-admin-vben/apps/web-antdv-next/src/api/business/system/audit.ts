import type { PageQuery, PageResult } from './types';

import { requestClient } from '#/api/request';

export interface AuditLog {
  id: number;
  actorId: number;
  actorName: string;
  authorityId: number;
  eventType: 'login' | 'operation';
  module: string;
  action: string;
  object: string;
  path: string;
  method: string;
  result: 'failure' | 'success';
  statusCode: number;
  ip: string;
  traceId: string;
  durationMs: number;
  params: string;
  createdAt: string;
}
export interface AuditQuery extends PageQuery {
  actorId?: number;
  actorName?: string;
  eventType?: AuditLog['eventType'];
  module?: string;
  result?: AuditLog['result'];
  startTime?: string;
  endTime?: string;
}
export const getAuditLogs = (params: AuditQuery) =>
  requestClient.get<PageResult<AuditLog>>('/v1/sys/audit/getAuditLogList', {
    params,
  });
