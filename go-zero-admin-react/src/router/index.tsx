import { createElement, lazy, Suspense } from 'react';
import { createBrowserRouter, Navigate } from 'react-router-dom';
import type { RouteObject } from 'react-router-dom';
import { Spin } from 'antd';
import { useAuthStore } from '../stores/auth';
import BasicLayout from '../layouts/BasicLayout';
import LoginPage from '../pages/login';
import type { MenuInfo } from '../api/types';

/**
 * 页面组件注册表
 * key = 后端菜单的 name 字段；新增业务页面时在此登记
 */
const pages = {
  dashboard: lazy(() => import('../pages/dashboard')),
};

/** 后端菜单树 → 路由表（仅 type=2 菜单参与路由） */
export function buildRoutes(menus: MenuInfo[]): RouteObject[] {
  return menus
    .filter((m) => m.type === 2 && m.path && pages[m.name as keyof typeof pages])
    .map((m) => ({
      path: m.path,
      element: (
        <Suspense fallback={<Spin style={{ margin: '40px auto', display: 'block' }} />}>
          {createElement(pages[m.name as keyof typeof pages])}
        </Suspense>
      ),
    }));
}

/** 鉴权守卫：无 token 跳登录 */
function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token);
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: (
      <RequireAuth>
        <BasicLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      {
        path: 'dashboard',
        element: (
          <Suspense fallback={<Spin style={{ margin: '40px auto', display: 'block' }} />}>
            {createElement(pages.dashboard)}
          </Suspense>
        ),
      },
    ],
  },
  { path: '*', element: <Navigate to="/" replace /> },
]);

/** 动态路由注入（联调时：登录后拉 /sys/menu/my 调用） */
export function injectDynamicRoutes(menus: MenuInfo[]) {
  const layout = router.routes.find((r) => r.path === '/');
  if (!layout || !('children' in layout)) return;
  const dynamic = buildRoutes(menus);
  (layout as { children: RouteObject[] }).children = [
    ...((layout as { children?: RouteObject[] }).children ?? []),
    ...dynamic,
  ];
}
