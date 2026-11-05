// auth 存储副作用：localStorage / sessionStorage 读写与标签页事件。
// 与 zustand store 分离，便于单测与替换实现（如改为 cookie 或 IndexedDB）。
const storageKey = 'go-zero-auth';
const departmentStorageKey = 'go-zero-department';

let lastPersistedSession: string | null = null;
let applyingSharedSession = false;

// zustand persist 中间件的自定义 storage：同步只读快照，读取后其他页面可能已轮换，
// 不能把这个快照再次持久化到其他标签页。
export const authPersistStorage = {
  getItem: (name: string): string | null => {
    lastPersistedSession = window.localStorage.getItem(name);
    return lastPersistedSession;
  },
  setItem: (name: string, value: string): void => {
    if (applyingSharedSession) { lastPersistedSession = value; return; }
    // 权限、部门等本页状态变动不能把尚未同步的旧令牌写回其他标签页。
    if (value === lastPersistedSession) return;
    window.localStorage.setItem(name, value);
    lastPersistedSession = value;
  },
  removeItem: (name: string): void => {
    window.localStorage.removeItem(name);
    lastPersistedSession = null;
  },
};

export function readRawSession(): Partial<{ token: string; expiresAt: number; refreshToken: string;
  refreshExpiresAt: number; sessionId: string }> | undefined {
  try {
    const raw = window.localStorage.getItem(storageKey);
    return raw ? (JSON.parse(raw) as { state?: Record<string, unknown> }).state : undefined;
  } catch { return undefined; }
}

export function isRawSessionValid(saved: Record<string, unknown> | undefined): boolean {
  return typeof saved?.token === 'string' && (saved.token as string).length > 0
    && !(saved.token as string).startsWith('mock-token-')
    && typeof saved.expiresAt === 'number' && typeof saved.refreshToken === 'string'
    && typeof saved.refreshExpiresAt === 'number' && typeof saved.sessionId === 'string'
    && ((saved.expiresAt as number) * 1000 > Date.now() || (saved.refreshExpiresAt as number) * 1000 > Date.now());
}

// 读取当前持久化的令牌快照；返回 undefined 表示无有效会话。
export function readPersistedSession() {
  const saved = readRawSession();
  if (!saved || !isRawSessionValid(saved as Record<string, unknown>)) return undefined;
  return {
    token: saved.token as string,
    expiresAt: saved.expiresAt as number,
    refreshToken: saved.refreshToken as string,
    refreshExpiresAt: saved.refreshExpiresAt as number,
    sessionId: saved.sessionId as string,
  };
}

export function readDepartmentPreference(sessionId: string): number | null {
  if (!sessionId || typeof window === 'undefined') return null;
  try {
    const raw = window.sessionStorage.getItem(departmentStorageKey);
    const saved = raw ? JSON.parse(raw) as { sessionId?: unknown; selectedDeptId?: unknown } | null : null;
    if (saved?.sessionId === sessionId && typeof saved.selectedDeptId === 'number'
      && Number.isSafeInteger(saved.selectedDeptId) && saved.selectedDeptId > 0) return saved.selectedDeptId;
    if (raw) saveDepartmentPreference('', null);
  } catch { saveDepartmentPreference('', null); }
  return null;
}

export function saveDepartmentPreference(sessionId: string, selectedDeptId: number | null): void {
  if (typeof window === 'undefined') return;
  // 部门只留在当前标签页，独立于共享令牌，避免切部门写回其他页面的旧会话。
  try {
    if (sessionId && selectedDeptId !== null) {
      window.sessionStorage.setItem(departmentStorageKey, JSON.stringify({ sessionId, selectedDeptId }));
    } else window.sessionStorage.removeItem(departmentStorageKey);
  } catch { /* 存储被禁用时，部门选择仍可在本页内存中使用。 */ }
}

export function beginSharedSessionWrite(): void { applyingSharedSession = true; }
export function endSharedSessionWrite(): void { applyingSharedSession = false; }
