import type { ComponentProps } from 'react';
import { Menu } from 'antd';
import { MenuIcon } from '../components/MenuIcon';
import type { MenuInfo } from '../api/types';
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

export function toMenuItems(menus: MenuInfo[], compactTools = false): Items {
  const tree = buildTree(flattenMenus(menus));
  const hasWorkflow = (nodes: MenuInfo[]): boolean => nodes.some(node => node.visible && node.status === 1 && node.type !== 3
    && (node.type === 2 && safeMenuPath(node.path) === '/ai/workflows' || hasWorkflow(node.children ?? [])));
  const hideTools = compactTools && hasWorkflow(tree);
  const convert = (nodes: MenuInfo[]): Items => nodes.filter(node => node.visible && node.status === 1 && node.type !== 3 && !(hideTools && safeMenuPath(node.path) === '/ai/tools')).map(node => {
    const children = node.children?.length ? convert(node.children) : undefined;
    const path = safeMenuPath(node.path);
    return { key: children?.length ? 'menu:' + node.id : path || 'menu:' + node.id, icon: <MenuIcon menu={node} />, label: node.title, children: children?.length ? children : undefined, disabled: !children?.length && !path };
  });
  return convert(tree);
}
