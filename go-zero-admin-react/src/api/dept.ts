import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type {
  CreateDeptReq,
  DeptInfo,
  GetDeptTreeReq,
  ListResult,
  UpdateDeptReq,
} from './types';

export function fetchDeptTree(params: GetDeptTreeReq = {}) {
  return apiGet<ListResult<DeptInfo>>('/sys/dept/tree', params);
}

export function createDept(data: CreateDeptReq) {
  return apiPost('/sys/dept/create', data);
}

export function updateDept(id: number, data: Omit<UpdateDeptReq, 'id'>) {
  return apiPut(`/sys/dept/${id}`, data);
}

export function deleteDept(id: number) {
  return apiDelete(`/sys/dept/${id}`);
}
