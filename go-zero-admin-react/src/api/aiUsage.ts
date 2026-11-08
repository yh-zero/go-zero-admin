import { apiGet, apiPut } from '../utils/request';
import type { AiSetUsagePolicyReq, AiUsageAdminQuery, AiUsageCallInfo, AiUsagePolicyInfo, AiUsagePolicyResp, AiUsageQuery, AiUsageSummaryResp, PageResult } from './types';

const base = '/sys/ai/usage';
export const fetchMyUsageSummary = (query: AiUsageQuery, signal?: AbortSignal) => apiGet<AiUsageSummaryResp>(`${base}/summary`, query, { signal });
export const fetchMyUsageCalls = (query: AiUsageQuery, signal?: AbortSignal) => apiGet<PageResult<AiUsageCallInfo>>(`${base}/calls`, query, { signal });
export const fetchMyUsagePolicy = (signal?: AbortSignal) => apiGet<AiUsagePolicyResp>(`${base}/policy`, undefined, { signal });
export const fetchAdminUsageSummary = (query: AiUsageAdminQuery, signal?: AbortSignal) => apiGet<AiUsageSummaryResp>(`${base}/admin/summary`, query, { signal });
export const fetchAdminUsageCalls = (query: AiUsageAdminQuery, signal?: AbortSignal) => apiGet<PageResult<AiUsageCallInfo>>(`${base}/admin/calls`, query, { signal });
export const fetchUsagePolicies = (query: AiUsageAdminQuery, signal?: AbortSignal) => apiGet<PageResult<AiUsagePolicyInfo>>(`${base}/admin/policies`, query, { signal });
export const setUsagePolicy = (ownerId: number, data: AiSetUsagePolicyReq) => apiPut<AiUsagePolicyResp>(`${base}/admin/policies/${ownerId}`, data);
