import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';
import type { LoginResp } from '../api/types';

const storageKey = 'go-zero-auth';
const departmentStorageKey = 'go-zero-department';
let lastPersistedSession: string | null = null;
let applyingSharedSession = false;
const sessionStorage = createJSONStorage(() => ({
  getItem: (name: string) => {
    lastPersistedSession = window.localStorage.getItem(name);
    return lastPersistedSession;
  },
  setItem: (name: string, value: string) => {
    if (applyingSharedSession) { lastPersistedSession = value; return; }
    // 权限、部门等本页状态变动不能把尚未同步的旧令牌写回其他标签页。
    if (value === lastPersistedSession) return;
    window.localStorage.setItem(name, value);
    lastPersistedSession = value;
  },
  removeItem: (name: string) => { window.localStorage.removeItem(name); lastPersistedSession = null; },
}));

interface AuthState {
  token: string; expiresAt: number; refreshToken: string; refreshExpiresAt: number; sessionId: string;
  sessionVersion: number; isLoggingOut: boolean; permCodes: string[]; permissionsReady: boolean;
  selectedDeptId: number | null; departmentVersion: number;
  setSession: (session: LoginResp) => void;
  refreshSession: (session: LoginResp, version: number) => void;
  beginLogout: () => void;
  setDepartment: (id: number | null) => void;
  setPermCodes: (codes: string[]) => void;
  clearPermissions: () => void;
  logout: () => void;
}
export const useAuthStore = create<AuthState>()(persist((set) => ({
  token: '', expiresAt: 0, refreshToken: '', refreshExpiresAt: 0, sessionId: '', sessionVersion: 0,
  isLoggingOut: false, permCodes: [], permissionsReady: false, selectedDeptId: null, departmentVersion: 0,
  setSession: (session) => {
    saveDepartmentPreference('', null);
    set(state => ({ ...sessionFields(session), sessionVersion: state.sessionVersion + 1,
      isLoggingOut: false, permCodes: [], permissionsReady: false, selectedDeptId: null, departmentVersion: state.departmentVersion + 1 }));
  },
  refreshSession: (session, version) => set(state => state.sessionVersion === version ? sessionFields(session) : state),
  beginLogout: () => set({ isLoggingOut: true }),
  setDepartment: (id) => set(state => {
    saveDepartmentPreference(state.sessionId, id);
    return { selectedDeptId: id, departmentVersion: state.departmentVersion + 1, permCodes: [], permissionsReady: false };
  }),
  setPermCodes: (permCodes) => set({ permCodes, permissionsReady: true }),
  clearPermissions: () => set({ permCodes: [], permissionsReady: false }),
  logout: () => {
    saveDepartmentPreference('', null);
    set(state => ({ token: '', expiresAt: 0, refreshToken: '', refreshExpiresAt: 0, sessionId: '',
      sessionVersion: state.sessionVersion + 1, isLoggingOut: false, permCodes: [], permissionsReady: false,
      selectedDeptId: null, departmentVersion: state.departmentVersion + 1 }));
  },
}), {
  name: storageKey,
  storage: sessionStorage,
  partialize: ({ token, expiresAt, refreshToken, refreshExpiresAt, sessionId }) => ({ token, expiresAt, refreshToken, refreshExpiresAt, sessionId }),
  merge: (persisted, current) => {
    const saved = persisted as Partial<AuthState> | undefined;
    const sessionId = typeof saved?.sessionId === 'string' ? saved.sessionId : '';
    const hasValidRefresh = typeof saved?.refreshToken === 'string' && saved.refreshToken.length > 0
      && typeof saved.refreshExpiresAt === 'number' && saved.refreshExpiresAt * 1000 > Date.now();
    return typeof saved?.token === 'string' && !saved.token.startsWith('mock-token-') && typeof saved.expiresAt === 'number'
      && (saved.expiresAt * 1000 > Date.now() || hasValidRefresh)
      ? { ...current, token: saved.token, expiresAt: saved.expiresAt, refreshToken: saved.refreshToken ?? '',
        refreshExpiresAt: saved.refreshExpiresAt ?? 0, sessionId, selectedDeptId: readDepartmentPreference(sessionId) }
      : current;
  },
}));

function sessionFields(session: LoginResp) {
  return { token: session.access_token, expiresAt: session.expires_at, refreshToken: session.refresh_token,
    refreshExpiresAt: session.refresh_expires_at, sessionId: session.session_id };
}

function readDepartmentPreference(sessionId: string): number | null {
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

function saveDepartmentPreference(sessionId: string, selectedDeptId: number | null) {
  if (typeof window === 'undefined') return;
  // 部门只留在当前标签页，独立于共享令牌，避免切部门写回其他页面的旧会话。
  try {
    if (sessionId && selectedDeptId !== null) {
      window.sessionStorage.setItem(departmentStorageKey, JSON.stringify({ sessionId, selectedDeptId }));
    } else window.sessionStorage.removeItem(departmentStorageKey);
  } catch { /* 存储被禁用时，部门选择仍可在本页内存中使用。 */ }
}

export function syncAuthFromStorage() {
  let saved: Partial<AuthState> | undefined;
  try {
    const raw = window.localStorage.getItem(storageKey);
    saved = raw ? (JSON.parse(raw) as { state?: Partial<AuthState> }).state : undefined;
  } catch { return; }
  const current = useAuthStore.getState();
  const valid = typeof saved?.token === 'string' && saved.token.length > 0 && !saved.token.startsWith('mock-token-')
    && typeof saved.expiresAt === 'number' && typeof saved.refreshToken === 'string'
    && typeof saved.refreshExpiresAt === 'number' && typeof saved.sessionId === 'string'
    && (saved.expiresAt * 1000 > Date.now() || saved.refreshExpiresAt * 1000 > Date.now());
  if (!saved || !valid) {
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

function applySharedSession(apply: () => void) {
  // 同步只读快照；读取后其他页面可能已轮换，不能把这个快照再次持久化。
  applyingSharedSession = true;
  try { apply(); } finally { applyingSharedSession = false; }
}

if (typeof window !== 'undefined') {
  window.addEventListener('storage', (event) => {
    if (event.storageArea === window.localStorage && (event.key === storageKey || event.key === null)) syncAuthFromStorage();
  });
}
