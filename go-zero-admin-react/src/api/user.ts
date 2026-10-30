import { apiDelete, apiGet, apiPost, apiPut } from '../utils/request';
import type { CreateUserReq, GetUserListReq, PageResult, UpdateUserReq, UserInfo } from './types';

export const fetchUsers = (params: GetUserListReq) => apiGet<PageResult<UserInfo>>('/sys/user/list', params);
export async function fetchUser(id: number): Promise<UserInfo> {
  return (await apiGet<{ user: UserInfo }>(`/sys/user/${id}`)).user;
}
export const createUser = (values: CreateUserReq) => apiPost('/sys/user/create', values);
export const updateUser = (id: number, values: UpdateUserReq) => apiPut(`/sys/user/${id}`, values);
export const deleteUser = (id: number) => apiDelete(`/sys/user/${id}`);
export const resetUserPassword = (id: number, newPassword: string) =>
  apiPost(`/sys/user/${id}/password`, { new_password: newPassword });
