import { apiGet } from '../utils/request';
import type { AuditLogInfo, GetAuditLogListReq, PageResult } from './types';
export const fetchAuditLogs = (params: GetAuditLogListReq) => apiGet<PageResult<AuditLogInfo>>('/sys/audit/list', params);
