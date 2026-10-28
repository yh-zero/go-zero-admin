import type { BackendMenu } from '../session';
import type { Department, RoleDataScope } from './organization';
import type { ApiResource, Menu, Policy } from './types';

import { sessionRequestOptions } from '#/adapter/business/request-session';
import { requestClient } from '#/api/request';
export type PermissionKind = 'api' | 'button' | 'dataScope' | 'menu';
export interface PermissionEdit {
  revision: string;
  authorityId: number;
  kind: PermissionKind;
  menus: Menu[];
  apis: ApiResource[];
  departments: Department[];
  menuIds: number[];
  menuBtnIds: number[];
  policies: Policy[];
  dataScope: RoleDataScope;
}
export interface PermissionSnapshot {
  revision: string;
  fingerprint: string;
  authorityId: number;
  defaultRouter: string;
  menus: BackendMenu[];
  codes: string[];
}
export interface PermissionState {
  menuIds?: number[];
  menuBtnIds?: number[];
  policies?: Policy[];
  dataScope?: RoleDataScope;
}
export interface PermissionChange {
  id: number;
  authorityId: number;
  kind: PermissionKind;
  beforeRevision: string;
  afterRevision: string;
  before: PermissionState;
  after: PermissionState;
  actorId: number;
  actorName: string;
  traceId: string;
  createdAt: string;
}
export async function getPermissionEdit(
  authorityId: number,
  kind: PermissionKind,
  token?: string,
) {
  const snapshot = await requestClient.get<PermissionEdit>(
    '/v1/sys/permissions/edit',
    { ...sessionRequestOptions(token), params: { authorityId, kind } },
  );
  if (
    snapshot.authorityId !== authorityId ||
    snapshot.kind !== kind ||
    !/^\d+$/.test(snapshot.revision)
  )
    throw new Error('权限编辑快照不匹配');
  return {
    ...snapshot,
    menus: snapshot.menus ?? [],
    apis: snapshot.apis ?? [],
    departments: snapshot.departments ?? [],
    menuIds: snapshot.menuIds ?? [],
    menuBtnIds: snapshot.menuBtnIds ?? [],
    policies: snapshot.policies ?? [],
    dataScope: {
      ...snapshot.dataScope,
      departmentIds: snapshot.dataScope.departmentIds ?? [],
    },
  };
}
export async function getPermissionSnapshot(token?: string) {
  const snapshot = await requestClient.get<PermissionSnapshot>(
    '/v1/sys/permissions/snapshot',
    sessionRequestOptions(token),
  );
  return {
    ...snapshot,
    menus: snapshot.menus ?? [],
    codes: snapshot.codes ?? [],
  };
}
export const getPermissionHistory = (
  authorityId: number,
  pageNo = 1,
  pageSize = 20,
  token?: string,
) =>
  requestClient.get<{ list: PermissionChange[]; total: number }>(
    '/v1/sys/permissions/history',
    {
      ...sessionRequestOptions(token),
      params: { authorityId, pageNo, pageSize },
    },
  );
export const previewPermissionRollback = (id: number, token?: string) =>
  requestClient.post<{
    version: string;
    change: PermissionChange;
    allowed: boolean;
    reason: string;
  }>(
    '/v1/sys/permissions/rollback/preview',
    { id },
    sessionRequestOptions(token),
  );
export const applyPermissionRollback = (
  id: number,
  expectedRevision: string,
  version: string,
  token?: string,
) =>
  requestClient.post(
    '/v1/sys/permissions/rollback/apply',
    { id, expectedRevision, version },
    sessionRequestOptions(token),
  );
