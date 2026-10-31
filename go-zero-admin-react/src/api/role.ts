import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type { CreateRoleReq, GetRoleListReq, ListResult, MenuInfo, PageResult, RoleInfo, UpdateRoleReq } from './types';

export const fetchRoles = (params: GetRoleListReq) => apiGet<PageResult<RoleInfo>>('/sys/role/list', params);
export async function fetchRoleOptions(): Promise<RoleInfo[]> {
  const roles: RoleInfo[] = [];
  for (let page = 1; ; page++) {
    const result = await fetchRoles({ page, page_size: 100 });
    roles.push(...result.list);
    if (roles.length >= result.total || result.list.length === 0) return roles;
  }
}
export async function fetchRole(id: number): Promise<RoleInfo> {
  return (await apiGet<{ role: RoleInfo }>(`/sys/role/${id}`)).role;
}
export const createRole = (values: CreateRoleReq) => apiPost('/sys/role/create', values);
export const updateRole = (id: number, values: UpdateRoleReq) => apiPut(`/sys/role/${id}`, values);
export const deleteRole = (id: number) => apiDelete(`/sys/role/${id}`);
export const fetchRoleMenus = () => apiGet<ListResult<MenuInfo>>('/sys/menu/tree');
export const updateRoleMenus = (id: number, menuIds: number[]) =>
  apiPost(`/sys/role/${id}/menus`, { menu_ids: menuIds });
