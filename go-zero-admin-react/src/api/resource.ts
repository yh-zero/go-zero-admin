import { apiDelete, apiGet, apiPost } from '../utils/request';
import type { ApiInfo, GetApiListReq, PageResult, SyncApiResp } from './types';

export const fetchApiResources = (params: GetApiListReq) => apiGet<PageResult<ApiInfo>>('/sys/api/list', params);
export const deleteApiResource = (id: number) => apiDelete(`/sys/api/${id}`);
export const syncApiResources = () => apiPost<SyncApiResp>('/sys/api/sync');
