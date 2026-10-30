import axios from 'axios';
import { useAuthStore } from '../stores/auth';
import { ApiError, decodeResponse } from './response';
import { clearSession } from './session';
import { requiredWritePermission } from './permissions';
import { trackRequest, releaseRequest } from './requestLifecycle';

let reportError: ((text: string) => void) | undefined;
export function setRequestErrorReporter(reporter?: (text: string) => void) { reportError = reporter; }
function rejectRequest(error: ApiError, requestToken?: string): never {
  const currentToken = useAuthStore.getState().token;
  if (requestToken && requestToken !== currentToken) throw new axios.CanceledError('会话已切换');
  if (error.code === 401 && requestToken && requestToken === currentToken) clearSession();
  reportError?.(error.message.slice(0, 300));
  throw error;
}
export const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/v1',
  timeout: 30_000,
});
request.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;
  const isPublic = ['/ping', '/sys/auth/captcha', '/sys/auth/login'].includes(config.url ?? '');
  const required = requiredWritePermission(config.method ?? 'get', config.url ?? '');
  const auth = useAuthStore.getState();
  if (required && (!auth.permissionsReady || !auth.permCodes.includes(required))) {
    throw new ApiError(403, '操作权限未加载或已变化，请重新加载权限后重试');
  }
  if (token && !isPublic) config.headers.Authorization = 'Bearer ' + token;
  else config.headers.delete('Authorization');
  config.signal = trackRequest(config.signal instanceof AbortSignal ? config.signal : undefined);
  return config;
});
request.interceptors.response.use((response) => {
  releaseRequest(response.config.signal);
  const requestToken = String(response.config.headers.get('Authorization') ?? '').replace(/^Bearer /, '') || undefined;
  if (requestToken && requestToken !== useAuthStore.getState().token) throw new axios.CanceledError('会话已切换');
  try { response.data = decodeResponse(response.data); return response; }
  catch (error) {
    if (error instanceof ApiError) return rejectRequest(error, requestToken);
    throw error;
  }
}, (error: unknown) => {
  if (error instanceof ApiError) return rejectRequest(error);
  if (axios.isCancel(error)) return Promise.reject(error);
  if (!axios.isAxiosError(error)) return rejectRequest(new ApiError(500, '请求未完成，请重试'));
  releaseRequest(error.config?.signal);
  const requestToken = String(error.config?.headers.get('Authorization') ?? '').replace(/^Bearer /, '') || undefined;
  const status = error.response?.status;
  if (status === 401) return rejectRequest(new ApiError(401, '登录已过期，请重新登录'), requestToken);
  const body = error.response?.data as unknown;
  try { decodeResponse(body); } catch (businessError) {
    if (businessError instanceof ApiError) return rejectRequest(businessError, requestToken);
  }
  const text = status === 401 ? '登录已过期，请重新登录'
    : status === 403 ? '没有访问权限'
    : status ? '请求失败（HTTP ' + status + '）'
    : error.code === 'ECONNABORTED' ? '请求超时，请稍后重试' : '无法连接服务，请确认服务已启动';
  return rejectRequest(new ApiError(status ?? 500, text), requestToken);
});
export async function apiGet<T>(url: string, params?: object): Promise<T> { return (await request.get<T>(url, { params })).data; }
export async function apiPost<T = void>(url: string, data?: unknown): Promise<T> { return (await request.post<T>(url, data)).data; }
export async function apiPut<T = void>(url: string, data?: unknown): Promise<T> { return (await request.put<T>(url, data)).data; }
export async function apiDelete<T = void>(url: string): Promise<T> { return (await request.delete<T>(url)).data; }
