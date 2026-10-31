import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export const SIDEBAR_DEFAULT_WIDTH = 240;
export const SIDEBAR_MIN_WIDTH = 200;
export const SIDEBAR_MAX_WIDTH = 360;

export function clampSidebarWidth(width: number) {
  return Number.isFinite(width)
    ? Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, Math.round(width)))
    : SIDEBAR_DEFAULT_WIDTH;
}

interface LayoutState {
  sidebarWidth: number;
  setSidebarWidth: (width: number) => void;
}

export const useLayoutStore = create<LayoutState>()(persist(set => ({
  sidebarWidth: SIDEBAR_DEFAULT_WIDTH,
  setSidebarWidth: width => set({ sidebarWidth: clampSidebarWidth(width) }),
}), {
  name: 'go-zero-layout',
  storage: createJSONStorage(() => ({
    getItem: key => {
      try { return localStorage.getItem(key); } catch { return null; }
    },
    setItem: (key, value) => {
      // 浏览器禁用存储或配额不足时，仍允许在当前页面调节宽度。
      try { localStorage.setItem(key, value); } catch { /* 保留内存状态 */ }
    },
    removeItem: key => {
      try { localStorage.removeItem(key); } catch { /* 无可清理的持久化状态 */ }
    },
  })),
  partialize: ({ sidebarWidth }) => ({ sidebarWidth }),
  merge: (persisted, current) => {
    const saved = persisted as Partial<LayoutState> | undefined;
    return { ...current, sidebarWidth: clampSidebarWidth(saved?.sidebarWidth ?? SIDEBAR_DEFAULT_WIDTH) };
  },
}));
