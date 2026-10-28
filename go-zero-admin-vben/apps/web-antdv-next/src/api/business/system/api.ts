import type { ApiResource, PageQuery, PageResult } from './types';

import { requestClient } from '#/api/request';
export const getApis = (
  params: PageQuery &
    Partial<ApiResource> & { orderKey?: string; desc?: boolean },
) =>
  requestClient.get<PageResult<ApiResource>>('/v1/sys/api/getApiList', {
    params,
  });
export const getAllApis = () =>
  requestClient.get<{ apiList: ApiResource[] | null }>(
    '/v1/sys/api/getAllApiList',
  );
export const createApi = (data: ApiResource) =>
  requestClient.post('/v1/sys/api/createApi', data);
export const updateApi = (data: ApiResource) =>
  requestClient.put('/v1/sys/api/updateApi', data);
export const deleteApi = (data: ApiResource) =>
  requestClient.delete('/v1/sys/api/deleteApi', { data });
export const deleteApis = (ids: number[]) =>
  requestClient.delete('/v1/sys/api/deleteApisByIds', { data: { ids } });

export interface ApiSyncItem {
  key: string;
  id: number;
  path: string;
  method: string;
  apiGroup: string;
  description: string;
  currentApiGroup: string;
  currentDescription: string;
}
export interface ApiSyncPreview {
  version: string;
  added: ApiSyncItem[];
  changed: ApiSyncItem[];
  obsolete: ApiSyncItem[];
}
export const previewApiSync = () =>
  requestClient.get<ApiSyncPreview>('/v1/sys/api/previewSync');
export const applyApiSync = (version: string, keys: string[]) =>
  requestClient.post<{ added: number; updated: number }>('/v1/sys/api/applySync', {
    version,
    keys,
  });
