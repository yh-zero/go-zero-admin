import { syncAuthFromStorage, useAuthStore } from '../stores/auth';
import { queryClient } from './queryClient';
import { cancelPendingRequests } from './requestLifecycle';
import type { LoginResp } from '../api/types';
import { ApiError } from './response';
useAuthStore.subscribe((state, previous) => {
  if (state.sessionVersion === previous.sessionVersion) return;
  cancelPendingRequests();
  void queryClient.cancelQueries();
  queryClient.clear();
});
type SessionSnapshot = { sessionVersion: number; token?: string; sessionId?: string };
export async function withSessionLock<T>(action: () => Promise<T>, requireLock = false): Promise<T> {
  if (navigator.locks) return navigator.locks.request('go-zero-auth-session', action);
  if (requireLock) throw new ApiError(503, '当前浏览器无法安全刷新会话，请使用 HTTPS 或本机地址');
  return action();
}

export async function saveSession(session: LoginResp): Promise<void> {
  await withSessionLock(async () => { syncAuthFromStorage(true); useAuthStore.getState().setSession(session); });
}

export async function clearSession(expected: SessionSnapshot = useAuthStore.getState()): Promise<boolean> {
  return withSessionLock(async () => clearSessionLocked(expected));
}

// 仅供已持有同源会话锁的退出流程调用，避免重复获取不可重入的锁。
export function clearSessionLocked(expected: SessionSnapshot): boolean {
  syncAuthFromStorage(true);
  const current = useAuthStore.getState();
  if (current.sessionVersion !== expected.sessionVersion
    || (expected.token !== undefined && current.token !== expected.token)
    || (expected.sessionId !== undefined && current.sessionId !== expected.sessionId)) return false;
  cancelPendingRequests();
  void queryClient.cancelQueries();
  queryClient.clear();
  useAuthStore.getState().logout();
  return true;
}
