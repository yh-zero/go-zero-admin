import type { MenuInfo } from '../api/types';
export function safeMenuPath(path: string): string | null {
  if (!path.startsWith('/') || path.startsWith('//') || /[?#:*\\]/.test(path) || path.split('/').some(part => part === '..' || part === '.')) return null;
  return path.replace(/\/+$/, '') || '/';
}
export function flattenMenus(items: MenuInfo[]): MenuInfo[] {
  return items.flatMap(item => [item, ...flattenMenus(item.children ?? [])]);
}
export function pageNameForMenu(menu: MenuInfo): string {
  return menu.name || menu.component.replace(/^\/?(src\/)?pages\//, '').replace(/\/(index|[A-Z][^/]*)?(\.tsx)?$/, '');
}
