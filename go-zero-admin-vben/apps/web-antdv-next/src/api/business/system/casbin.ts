import type { Policy } from './types';

import { sessionRequestOptions } from '#/adapter/business/request-session';
import { requestClient } from '#/api/request';
export const getPolicies = (authorityId: number) =>
  requestClient.get<{ list: null | Policy[] }>(
    '/v1/sys/casbin/getPathByAuthorityId',
    { params: { authorityId } },
  );
export const savePolicies = (
  authorityId: number,
  casbinInfoList: Policy[],
  expectedRevision: string,
  token?: string,
) =>
  requestClient.put(
    '/v1/sys/casbin/updateCasbinData',
    { authorityId, casbinInfoList, expectedRevision },
    sessionRequestOptions(token),
  );
export const savePolicyApiIds = (
  authorityId: number,
  apiIds: number[],
  expectedRevision: string,
  token?: string,
) =>
  requestClient.put(
    '/v1/sys/casbin/updateCasbinDataByApiIds',
    { authorityId, apiIds, expectedRevision },
    sessionRequestOptions(token),
  );
