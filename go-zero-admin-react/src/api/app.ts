import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type {
  AppAdminUserInfo, AppFeedbackInfo, AppMembershipInfo, AppMembershipPlan,
  DeviceSession, GetAppFeedbackListReq, GetAppMembershipListReq, GetAppUserListReq,
  GrantAppMembershipReq, ListResult, PageResult, ReviewAppUserReq, UpdateAppFeedbackReq, UpdateAppUserReq,
} from './types';

export const fetchAppUsers = (params: GetAppUserListReq) =>
  apiGet<PageResult<AppAdminUserInfo>>('/sys/app/users', params);
export const fetchAppUser = async (id: number): Promise<AppAdminUserInfo> =>
  (await apiGet<{ user: AppAdminUserInfo }>(`/sys/app/users/${id}`)).user;
export const updateAppUser = (id: number, values: UpdateAppUserReq) => apiPut(`/sys/app/users/${id}`, values);
export const fetchAppMembershipPlans = async (): Promise<AppMembershipPlan[]> =>
  (await apiGet<ListResult<AppMembershipPlan>>('/sys/app/membership-plans')).list ?? [];
export const reviewAppUser = (id: number, values: ReviewAppUserReq) =>
  apiPost(`/sys/app/users/${id}/review`, values);
export const setAppUserAvatar = (id: number, fileId: number) =>
  apiPut(`/sys/app/users/${id}/avatar`, { file_id: fileId });
export const fetchAppUserSessions = async (id: number): Promise<DeviceSession[]> =>
  (await apiGet<ListResult<DeviceSession>>(`/sys/app/users/${id}/sessions`)).list ?? [];
export const revokeAppUserSession = (userId: number, sessionId: string) =>
  apiDelete(`/sys/app/users/${userId}/sessions/${encodeURIComponent(sessionId)}`);
export const fetchAppMemberships = (params: GetAppMembershipListReq) =>
  apiGet<PageResult<AppMembershipInfo>>('/sys/app/memberships', params);
export const grantAppMembership = (values: GrantAppMembershipReq) =>
  apiPost('/sys/app/memberships', values);
export const revokeAppMembership = (id: number) => apiDelete(`/sys/app/memberships/${id}`);
export const fetchAppFeedback = (params: GetAppFeedbackListReq) => apiGet<PageResult<AppFeedbackInfo>>('/sys/app/feedback', params);
export const updateAppFeedback = (id: number, values: UpdateAppFeedbackReq) => apiPut(`/sys/app/feedback/${id}`, values);
