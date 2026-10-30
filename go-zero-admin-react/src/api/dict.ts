import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type { CreateDictReq, DictInfo, GetDictListReq, PageResult, UpdateDictReq } from './types';

export const fetchDicts = (params: GetDictListReq) => apiGet<PageResult<DictInfo>>('/sys/dict/list', params);
export async function fetchDictByCode(code: string): Promise<DictInfo> {
  const result = await apiGet<{ dict: DictInfo }>(`/sys/dict/code/${encodeURIComponent(code)}`);
  return result.dict;
}
export const createDict = (data: CreateDictReq) => apiPost('/sys/dict/create', data);
export const updateDict = (id: number, data: UpdateDictReq) => apiPut(`/sys/dict/${id}`, data);
export const deleteDict = (id: number) => apiDelete(`/sys/dict/${id}`);
