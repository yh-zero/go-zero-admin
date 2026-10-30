import { apiDelete, apiGet } from '../utils/request';
import type { ApiInfo, GetApiListReq, PageResult } from './types';

export const fetchApiResources = (params: GetApiListReq) => apiGet<PageResult<ApiInfo>>('/sys/api/list', params);
export const deleteApiResource = (id: number) => apiDelete(`/sys/api/${id}`);
