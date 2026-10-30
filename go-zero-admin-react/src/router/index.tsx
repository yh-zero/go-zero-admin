import { createBrowserRouter, Navigate } from 'react-router-dom';
import type { RouteObject } from 'react-router-dom';
import BasicLayout from '../layouts/BasicLayout';
import LoginPage from '../pages/login';
import type { MenuInfo } from '../api/types';
import { pages, pageLinks } from './registry';
import { flattenMenus, pageNameForMenu, safeMenuPath } from './menuRoutes';
import RequireAuth from './RequireAuth';
import DynamicPage from './DynamicPage';
import PageElement from './PageElement';

export function buildRoutes(menus: MenuInfo[]): RouteObject[] {
  return flattenMenus(menus).filter(menu => menu.type === 2 && menu.status === 1 && safeMenuPath(menu.path) && Object.hasOwn(pages, pageNameForMenu(menu)))
    .map(menu => ({ path: safeMenuPath(menu.path)!.slice(1), element: <PageElement name={pageNameForMenu(menu) as keyof typeof pages} /> }));
}
export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/', element: <RequireAuth><BasicLayout /></RequireAuth>, children: [
    { index: true, element: <Navigate to="/dashboard" replace /> },
    ...pageLinks.map(page => ({ path: page.path.slice(1), element: <PageElement name={page.name} /> })),
    { path: '*', element: <DynamicPage /> },
  ] },
]);
