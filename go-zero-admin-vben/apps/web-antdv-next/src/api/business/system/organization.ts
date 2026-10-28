import { sessionRequestOptions } from '#/adapter/business/request-session';
import { requestClient } from '#/api/request';

export interface Department {
  id: number;
  parentId: number;
  name: string;
  code: string;
  sort: number;
  status: number;
  leader?: string;
  children?: Department[];
}
export interface Position {
  id: number;
  name: string;
  code: string;
  sort: number;
  status: number;
}
export interface OrganizationQuery {
  keyword?: string;
  status?: number;
}
export interface Membership {
  userId: number;
  departmentId: number;
  positionIds: number[];
}
export type DataScope =
  | 'all'
  | 'custom'
  | 'department'
  | 'department_and_children'
  | 'self';
export interface RoleDataScope {
  authorityId: number;
  scope: DataScope;
  departmentIds: number[];
  revision?: string;
  expectedRevision?: string;
}
const base = '/v1/sys/organization';

export async function getDepartments(params: OrganizationQuery = {}) {
  const result = await requestClient.get<{ list?: Department[] | null }>(
    `${base}/departments`,
    { params },
  );
  return result.list ?? [];
}
export const createDepartment = (data: Omit<Department, 'children' | 'id'>) =>
  requestClient.post<Department>(`${base}/departments`, data);
export const updateDepartment = (data: Omit<Department, 'children'>) =>
  requestClient.put(`${base}/departments`, data);
export const deleteDepartment = (id: number) =>
  requestClient.delete(`${base}/departments`, { data: { id } });

export async function getPositions(params: OrganizationQuery = {}) {
  const result = await requestClient.get<{ list?: null | Position[] }>(
    `${base}/positions`,
    { params },
  );
  return result.list ?? [];
}
export const createPosition = (data: Omit<Position, 'id'>) =>
  requestClient.post<Position>(`${base}/positions`, data);
export const updatePosition = (data: Position) =>
  requestClient.put(`${base}/positions`, data);
export const deletePosition = (id: number) =>
  requestClient.delete(`${base}/positions`, { data: { id } });

export async function getMembership(userId: number): Promise<Membership> {
  const result = await requestClient.get<Membership>(`${base}/membership`, {
    params: { userId },
  });
  return { ...result, positionIds: result.positionIds ?? [] };
}
export const updateMembership = (data: Membership) =>
  requestClient.put(`${base}/membership`, data);
export async function getRoleDataScope(
  authorityId: number,
): Promise<RoleDataScope> {
  const result = await requestClient.get<RoleDataScope>(`${base}/dataScope`, {
    params: { authorityId },
  });
  return { ...result, departmentIds: result.departmentIds ?? [] };
}
export const updateRoleDataScope = (data: RoleDataScope, token?: string) =>
  requestClient.put(`${base}/dataScope`, data, sessionRequestOptions(token));
