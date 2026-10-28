import { sessionRequestOptions } from '#/adapter/business/request-session';
import { requestClient } from '#/api/request';

export type AppUserStatus = 'active' | 'disabled' | 'pending' | 'rejected';
export interface AppUser {
  id: number;
  account: string;
  identityKind: string;
  identityVerified: boolean;
  registrationIP: string;
  registrationRegion: string;
  nickname: string;
  avatarUrl: string;
  status: AppUserStatus;
  reviewReason: string;
  createdAt: string;
  updatedAt: string;
}
export interface AppSession {
  id: string;
  userId: number;
  ip: string;
  userAgent: string;
  createdAt: string;
  expiresAt: string;
  current: boolean;
}
export interface AppPlan {
  id: number;
  code: string;
  name: string;
  version: number;
  entitlements: string[];
}
export interface AppGrant {
  id: number;
  userId: number;
  account: string;
  planId: number;
  planCode: string;
  planVersion: number;
  entitlements: string[];
  startsAt: string;
  endsAt: string;
  revokedAt: string;
  reason: string;
  revokeReason: string;
  createdAt: string;
  status: 'active' | 'expired' | 'revoked' | 'scheduled';
}
export interface AppPage<T> {
  list: null | T[];
  total: number;
  pageNo: number;
  pageSize: number;
}
export interface AppUserQuery {
  pageNo: number;
  pageSize: number;
  keyword?: string;
  status?: AppUserStatus;
}
export interface AppGrantQuery {
  pageNo: number;
  pageSize: number;
  userId?: number;
}
export interface GrantAppMembership {
  userId: number;
  planId: number;
  days: number;
  requestId: string;
  reason: string;
}
export const getAppUsers = (params: AppUserQuery, token: string) =>
  requestClient.get<AppPage<AppUser>>('/v1/sys/app-members/users', {
    ...sessionRequestOptions(token),
    params,
  });
export const updateAppUserStatus = (
  data: { userId: number; status: AppUserStatus; reason: string },
  token: string,
) =>
  requestClient.put<{ message: string }>(
    '/v1/sys/app-members/status',
    data,
    sessionRequestOptions(token),
  );
export const updateAppUserAvatar = (
  data: { userId: number; avatarUrl: string },
  token: string,
) =>
  requestClient.put<{ message: string }>(
    '/v1/sys/app-members/avatar',
    data,
    sessionRequestOptions(token),
  );
export const getAppSessions = (userId: number, token: string) =>
  requestClient.get<{ list: AppSession[] | null }>(
    '/v1/sys/app-members/sessions',
    {
      ...sessionRequestOptions(token),
      params: { userId },
    },
  );
export const revokeAppSession = (id: string, token: string) =>
  requestClient.delete<{ message: string }>('/v1/sys/app-members/session', {
    ...sessionRequestOptions(token),
    data: { id },
  });
export const getAppPlans = (token: string) =>
  requestClient.get<{ list: AppPlan[] | null }>(
    '/v1/sys/app-members/plans',
    sessionRequestOptions(token),
  );
export const getAppGrants = (params: AppGrantQuery, token: string) =>
  requestClient.get<AppPage<AppGrant>>('/v1/sys/app-members/grants', {
    ...sessionRequestOptions(token),
    params,
  });
export const grantAppMembership = (data: GrantAppMembership, token: string) =>
  requestClient.post<AppGrant>(
    '/v1/sys/app-members/grants',
    data,
    sessionRequestOptions(token),
  );
export const revokeAppGrant = (
  data: { id: number; reason: string },
  token: string,
) =>
  requestClient.delete<{ message: string }>('/v1/sys/app-members/grant', {
    ...sessionRequestOptions(token),
    data,
  });
