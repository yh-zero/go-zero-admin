import type { Menu } from './types';

import { sessionRequestOptions } from '#/adapter/business/request-session';
import { requestClient } from '#/api/request';
export const getMenus = () =>
  requestClient.get<{ list: Menu[] | null }>('/v1/sys/menu/getMenuList');
export const getMenuTree = () =>
  requestClient.get<{ list: Menu[] | null }>('/v1/sys/menu/getBaseMenuTree');
export const getAuthorityMenus = (authorityId: number, token?: string) =>
  requestClient.get<{ list: Menu[] | null }>('/v1/sys/menu/getMenuAuthority', {
    ...sessionRequestOptions(token),
    params: { authorityId },
  });
export const getMenu = (id: number, token?: string) =>
  requestClient.get<{ sysBaseMenu: Menu }>('/v1/sys/menu/getBaseMenuById', {
    ...sessionRequestOptions(token),
    params: { id },
  });
export const createMenu = (data: Menu, token?: string) =>
  requestClient.post(
    '/v1/sys/menu/addBaseMenu',
    data,
    sessionRequestOptions(token),
  );
export const updateMenu = (data: Menu, token?: string) =>
  requestClient.put(
    '/v1/sys/menu/updateBaseMenu',
    data,
    sessionRequestOptions(token),
  );
export const deleteMenu = (id: number, token?: string) =>
  requestClient.delete('/v1/sys/menu/deleteBaseMenu', {
    ...sessionRequestOptions(token),
    data: { id },
  });
export const getAuthorityButtons = (authorityId: number) =>
  requestClient.get<{ menuBtnIds: null | number[] }>(
    '/v1/sys/menu/getAuthorityButtons',
    { params: { authorityId } },
  );
export const saveAuthorityButtons = (
  authorityId: number,
  menuBtnIds: number[],
  expectedRevision: string,
  token?: string,
) =>
  requestClient.put(
    '/v1/sys/menu/updateAuthorityButtons',
    { authorityId, menuBtnIds, expectedRevision },
    sessionRequestOptions(token),
  );
export async function previewMenuMove(
  id: number,
  parentId: number,
  token?: string,
) {
  const preview = await requestClient.post<{
    version: string;
    authorityIds: number[];
    ancestorIds: number[];
    addedLinks: Array<{ authorityId: number; menuId: number }>;
  }>(
    '/v1/sys/menu/previewMove',
    { id, parentId },
    sessionRequestOptions(token),
  );
  return {
    ...preview,
    authorityIds: preview.authorityIds ?? [],
    ancestorIds: preview.ancestorIds ?? [],
    addedLinks: preview.addedLinks ?? [],
  };
}
