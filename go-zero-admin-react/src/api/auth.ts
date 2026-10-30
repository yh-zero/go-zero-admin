import { request } from '../utils/request';
import type { CaptchaResp, LoginReq, LoginResp, MenuInfo } from './types';

// 注：以下为接口骨架，联调时启用（request 拦截器 + 后端地址）

export async function fetchCaptcha(): Promise<CaptchaResp> {
  const { data } = await request.get<{ result: CaptchaResp }>('/sys/auth/captcha');
  return data.result;
}

export async function login(req: LoginReq): Promise<LoginResp> {
  const { data } = await request.post<{ result: LoginResp }>('/sys/auth/login', req);
  return data.result;
}

export async function fetchMyMenus(): Promise<MenuInfo[]> {
  const { data } = await request.get<{ result: MenuInfo[] }>('/sys/menu/my');
  return data.result;
}

export async function fetchMyPermCodes(): Promise<string[]> {
  const { data } = await request.get<{ result: string[] }>('/sys/menu/my/perm-codes');
  return data.result;
}
