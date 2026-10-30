import { apiGet, apiPost } from '../utils/request';
import type { CaptchaResp, LoginReq, LoginResp, MenuInfo, ListResult } from './types';
export const fetchCaptcha = () => apiGet<CaptchaResp>('/sys/auth/captcha');
export const login = (values: LoginReq) => apiPost<LoginResp>('/sys/auth/login', values);
export const fetchMyMenus = async (): Promise<MenuInfo[]> => (await apiGet<ListResult<MenuInfo>>('/sys/menu/my')).list ?? [];
export const fetchMyPermCodes = async (): Promise<string[]> => (await apiGet<{ codes: string[] }>('/sys/menu/my/perm-codes')).codes ?? [];
