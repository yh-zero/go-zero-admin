import { apiGet, apiPut, apiPost } from '../utils/request';
import type { ProfileInfo, UpdateProfileReq, ChangePasswordReq } from './types';
export const fetchProfile = async () => (await apiGet<{ user: ProfileInfo }>('/sys/profile/')).user;
export const updateProfile = (values: UpdateProfileReq) => apiPut('/sys/profile/', values);
export const changePassword = (values: ChangePasswordReq) => apiPost('/sys/user/password/change', values);
