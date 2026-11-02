import axios from 'axios';
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type { LoginResp } from '../api/types';
import { syncAuthFromStorage, useAuthStore } from '../stores/auth';
import { ApiError, decodeResponse } from './response';
import { clearSession, clearSessionLocked, withSessionLock } from './session';
import { requiredWritePermission } from './permissions';
import { trackRequest, releaseRequest, cancelPendingRequests } from './requestLifecycle';
import { queryClient } from './queryClient';

let reportError: ((text: string) => void) | undefined;
type AuthRequestConfig = InternalAxiosRequestConfig & { authVersion?: number; departmentVersion?: number; authRetried?: boolean };
let refreshFlight: { version: number; promise: Promise<void> } | undefined;
const publicUrls = new Set(['/ping', '/sys/auth/captcha', '/sys/auth/login', '/sys/auth/refresh']);
const refreshUrl = '/sys/auth/refresh';
const logoutUrl = '/sys/auth/logout';
const reportedErrors = new WeakSet<ApiError>();
const usesDepartment = (url = '') => !publicUrls.has(url) && !url.startsWith('/sys/auth/') && url !== '/sys/dept/my';
export function setRequestErrorReporter(reporter?: (text: string) => void) { reportError = reporter; }
function assertCurrentSession(config?: AuthRequestConfig) {
  syncAuthFromStorage();
  const auth = useAuthStore.getState();
  if (config?.authVersion !== undefined && config.authVersion !== auth.sessionVersion) {
    throw new axios.CanceledError('会话已切换');
  }
  if (config?.departmentVersion !== undefined && config.departmentVersion !== auth.departmentVersion) {
    throw new axios.CanceledError('部门已切换');
  }
}
async function rejectRequest(error: ApiError, config?: AuthRequestConfig): Promise<never> {
  assertCurrentSession(config);
  if (config?.url === refreshUrl) throw error;
  if (error.code === 401 && config?.authVersion !== undefined && config.url !== logoutUrl) {
    const token = String(config.headers.get('Authorization') ?? '').replace(/^Bearer /, '');
    if (!await clearSession({ sessionVersion: config.authVersion, token: token || useAuthStore.getState().token })) {
      throw new axios.CanceledError('会话已更新');
    }
  }
  reportApiError(error);
  throw error;
}
function reportApiError(error: ApiError) {
  if (reportedErrors.has(error)) return;
  reportedErrors.add(error);
  reportError?.(error.message.slice(0, 300));
}
export const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/v1',
  timeout: 30_000,
});
request.interceptors.request.use(async (input) => {
  const config = input as AuthRequestConfig;
  const isPublic = publicUrls.has(config.url ?? '');
  const departmentScoped = usesDepartment(config.url);
  syncAuthFromStorage();
  let auth = useAuthStore.getState();
  if (auth.isLoggingOut && !isPublic && config.url !== logoutUrl) throw new axios.CanceledError('正在退出登录');
  assertCurrentSession(config);
  if (!isPublic && auth.token) {
    config.authVersion ??= auth.sessionVersion;
    if (departmentScoped) config.departmentVersion ??= auth.departmentVersion;
    if (!config.authRetried && config.url !== logoutUrl && auth.expiresAt * 1000 <= Date.now() + 5000 && auth.refreshToken) {
      try { await refreshAccessToken(); }
      catch (error) { if (error instanceof ApiError) return rejectRequest(error, config); throw error; }
      assertCurrentSession(config);
      auth = useAuthStore.getState();
    }
  }
  const required = requiredWritePermission(config.method ?? 'get', config.url ?? '');
  if (required && (!auth.permissionsReady || !auth.permCodes.includes(required))) {
    throw new ApiError(403, '操作权限未加载或已变化，请重新加载权限后重试');
  }
  if (auth.token && !isPublic) config.headers.Authorization = 'Bearer ' + auth.token;
  else config.headers.delete('Authorization');
  if (departmentScoped && auth.selectedDeptId !== null) config.headers['X-Dept-Id'] = String(auth.selectedDeptId);
  else config.headers.delete('X-Dept-Id');
  config.signal = trackRequest(config.signal instanceof AbortSignal ? config.signal : undefined);
  return config;
});
request.interceptors.response.use(async (response) => {
  releaseRequest(response.config.signal);
  const config = response.config as AuthRequestConfig;
  assertCurrentSession(config);
  try { response.data = decodeResponse(response.data); return response; }
  catch (error) {
    if (error instanceof ApiError) return handleFailure(error, config);
    throw error;
  }
}, async (error: unknown) => {
  if (error instanceof ApiError) return rejectRequest(error);
  if (axios.isCancel(error)) return Promise.reject(error);
  if (!axios.isAxiosError(error)) return rejectRequest(new ApiError(500, '请求未完成，请重试'));
  releaseRequest(error.config?.signal);
  const config = error.config as AuthRequestConfig | undefined;
  assertCurrentSession(config);
  const status = error.response?.status;
  if (status === 401) return handleFailure(new ApiError(401, '登录已过期，请重新登录'), config);
  const body = error.response?.data as unknown;
  try { decodeResponse(body); } catch (businessError) {
    if (businessError instanceof ApiError) return handleFailure(businessError, config);
  }
  const text = status === 403 ? '没有访问权限'
    : status ? '请求失败（HTTP ' + status + '）'
    : error.code === 'ECONNABORTED' ? '请求超时，请稍后重试' : '无法连接服务，请确认服务已启动';
  return rejectRequest(new ApiError(status ?? 500, text), config);
});

async function refreshAccessToken(): Promise<void> {
  syncAuthFromStorage();
  const auth = useAuthStore.getState();
  if (refreshFlight?.version === auth.sessionVersion) return refreshFlight.promise;
  if (!auth.refreshToken || auth.refreshExpiresAt * 1000 <= Date.now()) throw new ApiError(401, '登录已过期，请重新登录');
  const version = auth.sessionVersion;
  const promise = withSessionLock(() => rotateSession(auth), true).finally(() => {
    if (refreshFlight?.promise === promise) refreshFlight = undefined;
  });
  refreshFlight = { version, promise };
  return promise;
}

async function rotateSession(expected: ReturnType<typeof useAuthStore.getState>): Promise<void> {
  if (!navigator.locks) throw new ApiError(503, '当前浏览器无法安全刷新会话，请使用 HTTPS 或本机地址');
  syncAuthFromStorage();
  const auth = useAuthStore.getState();
  if (auth.sessionVersion !== expected.sessionVersion || auth.sessionId !== expected.sessionId) throw new axios.CanceledError('会话已切换');
  if (auth.refreshToken !== expected.refreshToken && auth.expiresAt * 1000 > Date.now() + 5000) return;
  if (!auth.refreshToken || auth.refreshExpiresAt * 1000 <= Date.now()) throw new ApiError(401, '登录已过期，请重新登录');
  const { data } = await request.post<LoginResp>(refreshUrl, { refresh_token: auth.refreshToken });
  syncAuthFromStorage();
  if (useAuthStore.getState().sessionVersion !== auth.sessionVersion) throw new axios.CanceledError('会话已切换');
  if (!data.access_token || !data.refresh_token || data.session_id !== auth.sessionId) throw new ApiError(401, '登录会话已失效，请重新登录');
  useAuthStore.getState().refreshSession(data, auth.sessionVersion);
}

async function handleFailure(error: ApiError, config?: AuthRequestConfig): Promise<AxiosResponse> {
  assertCurrentSession(config);
  if (error.code !== 401 || !config || config.authRetried || config.authVersion === undefined || config.url === logoutUrl) {
    return rejectRequest(error, config);
  }
  const auth = useAuthStore.getState();
  if (auth.isLoggingOut) throw new axios.CanceledError('正在退出登录');
  const sentToken = String(config.headers.get('Authorization') ?? '').replace(/^Bearer /, '');
  try {
    if (sentToken === auth.token) await refreshAccessToken();
    assertCurrentSession(config);
  } catch (refreshError) {
    if (refreshError instanceof ApiError) return rejectRequest(refreshError, config);
    throw refreshError;
  }
  config.authRetried = true;
  return request(config);
}

export async function logoutSession(allDevices = false): Promise<void> {
  syncAuthFromStorage();
  const auth = useAuthStore.getState();
  if (auth.isLoggingOut) return;
  auth.beginLogout();
  const version = auth.sessionVersion;
  try {
    // 等待已发出的轮换完成，再撤销同一会话，避免退出与刷新互相覆盖。
    if (refreshFlight?.version === version) await refreshFlight.promise;
    await withSessionLock(async () => {
      syncAuthFromStorage();
      const current = useAuthStore.getState();
      if (current.sessionVersion !== version) return;
      try {
        if (current.token && current.expiresAt * 1000 <= Date.now() + 5000) await rotateSession(current);
        if (current.token) await request.post(logoutUrl, { all_devices: allDevices });
      } finally {
        syncAuthFromStorage();
        clearSessionLocked({ sessionVersion: version, sessionId: auth.sessionId });
      }
    });
  } catch (error) {
    if (error instanceof ApiError) reportApiError(error);
    throw error;
  } finally {
    syncAuthFromStorage();
    await clearSession({ sessionVersion: version, sessionId: auth.sessionId });
  }
}

export async function switchDepartment(id: number | null): Promise<void> {
  const auth = useAuthStore.getState();
  if (auth.isLoggingOut || !auth.token || auth.selectedDeptId === id) return;
  const version = auth.sessionVersion;
  // 部门切换不会撤销账号会话；先完成刷新，避免取消已被服务端消费的刷新令牌。
  if (refreshFlight?.version === version) {
    try { await refreshFlight.promise; }
    catch (error) { if (error instanceof ApiError) reportApiError(error); throw error; }
  }
  if (useAuthStore.getState().sessionVersion !== version || useAuthStore.getState().isLoggingOut) return;
  cancelPendingRequests();
  void queryClient.cancelQueries();
  queryClient.clear();
  useAuthStore.getState().setDepartment(id);
}
function requestContext(url: string) {
  syncAuthFromStorage();
  const auth = useAuthStore.getState();
  if (!auth.token || publicUrls.has(url)) return { url };
  return { url, authVersion: auth.sessionVersion, departmentVersion: usesDepartment(url) ? auth.departmentVersion : undefined };
}
export async function apiGet<T>(url: string, params?: object): Promise<T> { return (await request.get<T>(url, { ...requestContext(url), params })).data; }
export async function apiPost<T = void>(url: string, data?: unknown): Promise<T> { return (await request.post<T>(url, data, requestContext(url))).data; }
export async function apiPut<T = void>(url: string, data?: unknown): Promise<T> { return (await request.put<T>(url, data, requestContext(url))).data; }
export async function apiDelete<T = void>(url: string, data?: unknown): Promise<T> {
  return (await request.delete<T>(url, { ...requestContext(url), data })).data;
}
