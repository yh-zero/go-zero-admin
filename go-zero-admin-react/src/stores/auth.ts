import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';
import type { LoginResp } from '../api/types';
import { authPersistStorage, readDepartmentPreference, saveDepartmentPreference } from './authStorage';

export interface AuthState {
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
  name: 'go-zero-auth',
  storage: createJSONStorage(() => authPersistStorage),
  partialize: ({ token, expiresAt, refreshToken, refreshExpiresAt, sessionId }) => ({ token, expiresAt, refreshToken, refreshExpiresAt, sessionId }),
  merge: (persisted, current) => {
    const saved = persisted as Partial<AuthState> | undefined;
    const sessionId = typeof saved?.sessionId === 'string' ? saved.sessionId : '';
    const hasValidRefresh = typeof saved?.refreshToken === 'string' && saved.refreshToken.length > 0
      && typeof saved?.refreshExpiresAt === 'number' && saved.refreshExpiresAt * 1000 > Date.now();
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

// 重新导出同步函数，保持向后兼容（request.ts 等模块的 import 路径不变）。
export { syncAuthFromStorage } from './authSync';
import { initAuthStorageSync } from './authSync';
initAuthStorageSync();
