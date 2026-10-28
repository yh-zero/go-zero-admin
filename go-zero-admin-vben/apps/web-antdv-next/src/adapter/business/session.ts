import type { UserInfo } from '@vben/types';

import type { BackendUser, LoginResult } from '../../api/business/session';

import { resolveAvatar } from './avatar';

export const SESSION_HOME = '/_session/home';
const SESSION_KEY = 'go-zero-admin:session:v1';

interface Session {
  defaultRouter: string;
  expiresAt: number;
  user: UserInfo;
}

export class MissingSessionError extends Error {}
export class StaleSessionResponseError extends Error {}

export function saveSession(result: LoginResult): UserInfo {
  if (
    !result.accessToken ||
    !Number.isFinite(result.accessExpire) ||
    result.accessExpire * 1000 <= Date.now() ||
    !result.userInfo?.userName ||
    !result.userInfo.authorityId
  ) {
    throw new Error('登录响应缺少有效会话信息');
  }
  const backend = result.userInfo;
  const user = mapBackendUser(backend, result.accessToken);
  const session: Session = {
    defaultRouter: backend.authority?.defaultRouter || '',
    expiresAt: result.accessExpire,
    user,
  };
  // Only display fields and the existing access token; never passwords/captcha.
  localStorage.setItem(SESSION_KEY, JSON.stringify(session));
  return user;
}

export function readSession(token: null | string): Session {
  try {
    const session = JSON.parse(localStorage.getItem(SESSION_KEY) || 'null');
    if (
      token &&
      session?.user?.token === token &&
      typeof session.user.username === 'string' &&
      Array.isArray(session.user.roles) &&
      session.user.roles.every((role: unknown) => typeof role === 'string') &&
      Number.isFinite(session.expiresAt) &&
      session.expiresAt * 1000 > Date.now()
    ) {
      return session as Session;
    }
  } catch {
    // Missing, malformed or stale display caches require a new login.
  }
  throw new MissingSessionError('登录信息已失效，请重新登录');
}

export function clearSessionCache() {
  localStorage.removeItem(SESSION_KEY);
}

// Refresh display-only homepage metadata after editing the current role.
// A role switch still requires login because the JWT carries authorityId.
export function updateSessionHome(
  token: null | string,
  roleId: number,
  home: string,
) {
  const session = readSession(token);
  if (session.user.roles?.includes(String(roleId))) {
    session.defaultRouter = home;
    localStorage.setItem(SESSION_KEY, JSON.stringify(session));
  }
}

export function isAuthenticationError(error: unknown): boolean {
  const response = (
    error as { response?: { data?: { code?: number }; status?: number } }
  )?.response;
  return response?.status === 401 || response?.data?.code === 100_003;
}

export function safeRedirect(value: unknown, fallback: string): string {
  if (typeof value !== 'string') return fallback;
  try {
    const decoded = decodeURIComponent(value);
    return decoded.startsWith('/') &&
      !decoded.startsWith('//') &&
      !decoded.includes('\\') &&
      !decoded.startsWith('/auth') &&
      !decoded.startsWith('/_session/error')
      ? decoded
      : fallback;
  } catch {
    return fallback;
  }
}

export function mapBackendUser(backend: BackendUser, token: string): UserInfo {
  return {
    avatar: resolveAvatar(backend.headerImg),
    desc: '',
    homePath: SESSION_HOME,
    realName: backend.nickName || backend.userName,
    roles: [String(backend.authorityId)],
    token,
    userId: String(backend.ID),
    username: backend.userName,
  };
}

export function refreshSessionUser(
  backend: BackendUser,
  token: string,
): UserInfo {
  const session = readSession(token);
  session.user = mapBackendUser(backend, token);
  session.defaultRouter = backend.authority?.defaultRouter || '';
  localStorage.setItem(SESSION_KEY, JSON.stringify(session));
  return session.user;
}

// An old request may finish after the user has signed in again.
export function isCurrentSessionRequest(
  error: unknown,
  token: null | string,
): boolean {
  if (!token) return false;
  const request = error as {
    config?: { headers?: Record<string, unknown> };
    response?: { config?: { headers?: Record<string, unknown> } };
  };
  const headers =
    request?.config?.headers || request?.response?.config?.headers;
  return (
    (headers?.Authorization || headers?.authorization) === `Bearer ${token}`
  );
}
