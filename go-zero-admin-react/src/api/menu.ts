import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type { CreateMenuReq, GetMenuListReq, GetMenuTreeReq, ListResult, MenuInfo, PageResult, UpdateMenuReq } from './types';

export const fetchMenus = (params: GetMenuListReq) => apiGet<PageResult<MenuInfo>>('/sys/menu/list', params);
export const fetchMenuTree = (params: GetMenuTreeReq = {}) => apiGet<ListResult<MenuInfo>>('/sys/menu/tree', params);
export async function fetchMenu(id: number): Promise<MenuInfo> {
  const result = await apiGet<{ menu: MenuInfo }>(`/sys/menu/${id}`);
  return result.menu;
}
export const createMenu = (data: CreateMenuReq) => apiPost('/sys/menu/create', data);
export const updateMenu = (id: number, data: UpdateMenuReq) => apiPut(`/sys/menu/${id}`, data);
export const deleteMenu = (id: number) => apiDelete(`/sys/menu/${id}`);
