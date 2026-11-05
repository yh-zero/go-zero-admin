// 多标签页会话同步：storage 事件监听 + 跨标签状态合并。
import { useAuthStore } from './auth';
import { beginSharedSessionWrite, endSharedSessionWrite, readPersistedSession,
  saveDepartmentPreference } from './authStorage';

function applySharedSession(apply: () => void) {
  // 同步只读快照；读取后其他页面可能已轮换，不能把这个快照再次持久化。
  beginSharedSessionWrite();
  try { apply(); } finally { endSharedSessionWrite(); }
}

// 高频 API 调用时避免重复读 localStorage + JSON.parse（1 秒内不重复同步）。
let lastSyncAt = 0;
const syncThrottleMs = 1000;

export function syncAuthFromStorage(force = false): void {
  const now = Date.now();
  if (!force && now - lastSyncAt < syncThrottleMs) return;
  lastSyncAt = now;
  const saved = readPersistedSession();
  const current = useAuthStore.getState();
  if (!saved) {
    if (current.token) applySharedSession(() => current.logout());
    return;
  }
  if (saved.token === current.token && saved.refreshToken === current.refreshToken) return;
  const sameSession = saved.sessionId === current.sessionId;
  if (!sameSession) saveDepartmentPreference('', null);
  applySharedSession(() => useAuthStore.setState({
    token: saved.token, expiresAt: saved.expiresAt, refreshToken: saved.refreshToken,
    refreshExpiresAt: saved.refreshExpiresAt, sessionId: saved.sessionId,
    ...(!sameSession ? { sessionVersion: current.sessionVersion + 1, isLoggingOut: false,
      selectedDeptId: null, departmentVersion: current.departmentVersion + 1,
      permCodes: [], permissionsReady: false } : {}),
  }));
}

export function initAuthStorageSync(): void {
  if (typeof window === 'undefined') return;
  window.addEventListener('storage', (event) => {
    if (event.storageArea === window.localStorage && (event.key === 'go-zero-auth' || event.key === null)) {
      syncAuthFromStorage();
    }
  });
}
