import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  token: string; expiresAt: number; permCodes: string[]; permissionsReady: boolean;
  setSession: (token: string, expiresAt: number) => void;
  setPermCodes: (codes: string[]) => void;
  clearPermissions: () => void;
  logout: () => void;
}
export const useAuthStore = create<AuthState>()(persist((set) => ({
  token: '', expiresAt: 0, permCodes: [], permissionsReady: false,
  setSession: (token, expiresAt) => set({ token, expiresAt, permCodes: [], permissionsReady: false }),
  setPermCodes: (permCodes) => set({ permCodes, permissionsReady: true }),
  clearPermissions: () => set({ permCodes: [], permissionsReady: false }),
  logout: () => set({ token: '', expiresAt: 0, permCodes: [], permissionsReady: false }),
}), {
  name: 'go-zero-auth',
  partialize: ({ token, expiresAt }) => ({ token, expiresAt }),
  merge: (persisted, current) => {
    const saved = persisted as Partial<AuthState> | undefined;
    return typeof saved?.token === 'string' && !saved.token.startsWith('mock-token-') && typeof saved.expiresAt === 'number' && saved.expiresAt * 1000 > Date.now()
      ? { ...current, token: saved.token, expiresAt: saved.expiresAt }
      : current;
  },
}));
