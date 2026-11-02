import { apiDelete, apiGet, apiPost } from '../utils/request';
import type { CreateTenantReq, CreateTenantResp, GetTenantListReq, GetTenantResp, PageResult, TenantInfo } from './types';

export const fetchTenants = (params: GetTenantListReq) =>
  apiGet<PageResult<TenantInfo>>('/sys/tenant/list', params);
export const fetchTenant = (id: number) => apiGet<GetTenantResp>(`/sys/tenant/${id}`);
export const createTenant = (data: CreateTenantReq) => apiPost<CreateTenantResp>('/sys/tenant/create', data);
export const startTenant = (id: number) => apiPost(`/sys/tenant/${id}/start`);
export const stopTenant = (id: number) => apiPost(`/sys/tenant/${id}/stop`);
export const deleteTenant = (id: number, confirmSlug: string) =>
  apiDelete(`/sys/tenant/${id}`, { confirm_slug: confirmSlug });
export const resetTenantAdmin = (id: number) =>
  apiPost<{ new_password: string }>(`/sys/tenant/${id}/reset-admin`);
