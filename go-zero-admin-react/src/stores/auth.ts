import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  token: string;
  permCodes: string[];
  setToken: (token: string) => void;
  setPermCodes: (codes: string[]) => void;
  logout: () => void;
}

/** 认证状态（骨架：当前为本地 mock，联调时改为调用登录接口） */
export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: '',
      permCodes: [],
      setToken: (token) => set({ token }),
      setPermCodes: (permCodes) => set({ permCodes }),
      logout: () => set({ token: '', permCodes: [] }),
    }),
    { name: 'go-zero-auth' },
  ),
);
