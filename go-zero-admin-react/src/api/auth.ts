import { apiDelete, apiGet, apiPost, logoutSession } from '../utils/request';
import type { CaptchaResp, DeviceSession, LoginReq, LoginResp, MenuInfo, ListResult } from './types';
export const fetchCaptcha = () => apiGet<CaptchaResp>('/sys/auth/captcha');
export const login = (values: LoginReq) => apiPost<LoginResp>('/sys/auth/login', values);
export const logout = (allDevices = false) => logoutSession(allDevices);
export const fetchDeviceSessions = async (): Promise<DeviceSession[]> => (await apiGet<ListResult<DeviceSession>>('/sys/auth/sessions')).list ?? [];
export const revokeDeviceSession = (id: string) => apiDelete('/sys/auth/sessions/' + encodeURIComponent(id));
export const fetchMyMenus = async (): Promise<MenuInfo[]> => (await apiGet<ListResult<MenuInfo>>('/sys/menu/my')).list ?? [];
export const fetchMyPermCodes = async (): Promise<string[]> => (await apiGet<{ codes: string[] }>('/sys/menu/my/perm-codes')).codes ?? [];
