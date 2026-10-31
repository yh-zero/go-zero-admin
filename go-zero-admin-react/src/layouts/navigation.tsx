import type { ComponentProps } from 'react';
import { Menu } from 'antd';
import { MenuIcon } from '../components/MenuIcon';
import type { MenuInfo } from '../api/types';
import { pageLinks } from '../router/registry';
import { flattenMenus, safeMenuPath } from '../router/menuRoutes';
import { buildTree } from '../utils/tree';
type Items = NonNullable<ComponentProps<typeof Menu>['items']>;

export function getMenuAncestorKeys(items: Items, pathname: string): string[] | undefined {
  for (const item of items) {
    if (!item) continue;
    if (item.key === pathname) return [];
    if ('children' in item && item.children) {
      const ancestors = getMenuAncestorKeys(item.children, pathname);
      if (ancestors) {
        return item.key == null || item.type === 'group' ? ancestors : [String(item.key), ...ancestors];
      }
    }
  }
  return undefined;
}

export function toMenuItems(menus: MenuInfo[]): Items {
  const convert = (nodes: MenuInfo[]): Items => nodes.filter(node => node.visible && node.status === 1 && node.type !== 3).map(node => {
    const children = node.children?.length ? convert(node.children) : undefined;
    const path = safeMenuPath(node.path);
    return { key: children?.length ? 'menu:' + node.id : path || 'menu:' + node.id, icon: <MenuIcon menu={node} />, label: node.title, children: children?.length ? children : undefined, disabled: !children?.length && !path };
  });
  return convert(buildTree(flattenMenus(menus)));
}
export const fallbackItems: Items = pageLinks.map(page => ({ key: page.path, label: page.title, icon: <MenuIcon menu={{ ...page, type: 2 }} /> }));
