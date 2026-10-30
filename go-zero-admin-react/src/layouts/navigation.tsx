import type { ComponentProps, ReactNode } from 'react';
import { Menu } from 'antd';
import { DashboardOutlined, UserOutlined, SafetyOutlined, MenuOutlined, PartitionOutlined, BookOutlined, FileOutlined, FileSearchOutlined, ApiOutlined } from '@ant-design/icons';
import type { MenuInfo } from '../api/types';
import { pageLinks } from '../router/registry';
import { flattenMenus, safeMenuPath } from '../router/menuRoutes';
import { buildTree } from '../utils/tree';
const icons: Record<string, ReactNode> = {
  dashboard: <DashboardOutlined />, user: <UserOutlined />, role: <SafetyOutlined />, menu: <MenuOutlined />,
  dept: <PartitionOutlined />, dict: <BookOutlined />, file: <FileOutlined />, audit: <FileSearchOutlined />, api: <ApiOutlined />,
};
type Items = NonNullable<ComponentProps<typeof Menu>['items']>;
export function toMenuItems(menus: MenuInfo[]): Items {
  const convert = (nodes: MenuInfo[]): Items => nodes.filter(node => node.visible && node.status === 1 && node.type !== 3).map(node => {
    const children = node.children?.length ? convert(node.children) : undefined;
    const path = safeMenuPath(node.path);
    return { key: children?.length ? 'menu:' + node.id : path || 'menu:' + node.id, icon: icons[node.icon] || <MenuOutlined />, label: node.title, children: children?.length ? children : undefined, disabled: !children?.length && !path };
  });
  return convert(buildTree(flattenMenus(menus)));
}
export const fallbackItems: Items = pageLinks.map(page => ({ key: page.path, label: page.title, icon: icons[page.icon] }));
