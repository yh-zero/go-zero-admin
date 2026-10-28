import type { LoginResult } from '../business/session';

import { isAuthenticationError } from '#/adapter/business/session';
import { baseRequestClient, requestClient } from '#/api/request';
export namespace AuthApi {
  export interface LoginParams {
    username?: string;
    password?: string;
    captcha?: string;
    captchaId?: string;
  }
}
export function loginApi(data: AuthApi.LoginParams) {
  return requestClient.post<LoginResult>('/v1/sys/login', data);
}
export function getCaptchaApi() {
  return requestClient.get<{ captchaId: string; captchaImg: string }>(
    '/v1/sys/randomImage',
  );
}

// Use a plain client so a rejected token cannot recursively invoke logout.
export async function logoutApi(token: string) {
  try {
    const response = await baseRequestClient.post<{
      data: { code: number; message?: string };
    }>(
      '/v1/sys/logout',
      {},
      {
        headers: { Authorization: `Bearer ${token}` },
        timeout: 5000,
      },
    );
    if (response.data.code !== 200 && response.data.code !== 100003)
      throw new Error(response.data.message || '服务器退出失败');
  } catch (error) {
    // An already expired/revoked token has no remaining session to revoke.
    if (!isAuthenticationError(error)) throw error;
  }
}
