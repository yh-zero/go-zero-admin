import { apiPost } from '../utils/request';
import type { PermissionCheckReq, PermissionCheckResp } from './types';
export const checkPermission = (data: PermissionCheckReq) => apiPost<PermissionCheckResp>('/sys/permission/check', data);
