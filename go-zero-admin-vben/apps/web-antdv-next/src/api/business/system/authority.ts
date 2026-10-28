import type { Authority, PageQuery, PageResult } from './types';

import { sessionRequestOptions } from '#/adapter/business/request-session';
import { requestClient } from '#/api/request';
export const getAuthorities = (params: PageQuery, token?: string) =>
  requestClient.get<PageResult<Authority>>(
    '/v1/sys/authority/getAuthorityList',
    { ...sessionRequestOptions(token), params },
  );
export const createAuthority = (data: Authority, token?: string) =>
  requestClient.post(
    '/v1/sys/authority/createAuthority',
    data,
    sessionRequestOptions(token),
  );
export const updateAuthority = (data: Authority, token?: string) =>
  requestClient.put(
    '/v1/sys/authority/updateAuthority',
    data,
    sessionRequestOptions(token),
  );
export const deleteAuthority = (id: number, token?: string) =>
  requestClient.delete('/v1/sys/authority/deleteAuthority', {
    ...sessionRequestOptions(token),
    data: { id },
  });
export const saveAuthorityMenus = (
  authorityId: number,
  ids: number[],
  expectedRevision: string,
  token?: string,
) =>
  requestClient.post(
    '/v1/sys/authority/addAuthorityMenu',
    { authorityId, expectedRevision, menuIds: [...new Set(ids)].join(',') },
    sessionRequestOptions(token),
  );
// The API pages root roles; each root contains its complete descendants.
export async function getAllAuthorities(token?: string) {
  const result: Authority[] = [];
  for (let pageNo = 1; ; pageNo++) {
    const page = await getAuthorities({ pageNo, pageSize: 100 }, token);
    const rows = page.list ?? [];
    result.push(...rows);
    if (rows.length === 0 || result.length >= page.total) return result;
  }
}
