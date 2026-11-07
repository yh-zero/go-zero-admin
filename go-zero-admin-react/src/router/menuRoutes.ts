import type { MenuInfo } from '../api/types';
import { hiddenPageForPath } from './hiddenPages';
export function pageMenuPath(path: string): string | null {
  const safe = safeMenuPath(path);
  return safe ? hiddenPageForPath(safe)?.menuPath ?? safe : null;
}
export const isPageAccessible = (path: string, accessible: ReadonlySet<string>) => accessible.has(pageMenuPath(path) ?? '');
export function safeMenuPath(path: string): string | null {
  if (!path.startsWith('/') || path.startsWith('//') || /[?#:*\\]/.test(path) || path.split('/').some(part => part === '..' || part === '.')) return null;
  return path.replace(/\/+$/, '') || '/';
}
export function flattenMenus(items: MenuInfo[]): MenuInfo[] {
  return items.flatMap(item => [item, ...flattenMenus(item.children ?? [])]);
}
export function getAccessiblePagePaths(menus: MenuInfo[]): Set<string> {
  const paths = new Set(['/dashboard']);
  for (const menu of flattenMenus(menus)) {
    if (menu.type !== 2 || menu.status !== 1) continue;
    const path = safeMenuPath(menu.path);
    if (path) paths.add(path);
  }
  return paths;
}
export function pageNameForMenu(menu: MenuInfo): string {
  return menu.name || menu.component.replace(/^\/?(src\/)?pages\//, '').replace(/\/(index|[A-Z][^/]*)?(\.tsx)?$/, '');
}
