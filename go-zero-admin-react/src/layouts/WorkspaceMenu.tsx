import { useState } from 'react';
import { Menu } from 'antd';
import type { MenuProps } from 'antd';

interface Props {
  items: NonNullable<MenuProps['items']>;
  pathname: string;
  collapsed: boolean;
  parentKeys: string[];
  onClick: NonNullable<MenuProps['onClick']>;
}

export function WorkspaceMenu({ items, pathname, collapsed, parentKeys, onClick }: Props) {
  const [openKeys, setOpenKeys] = useState(parentKeys);

  // 悬浮菜单独立挂载，避免折叠时的弹出状态覆盖展开目录，或在路由变化后残留。
  return <Menu key={collapsed ? 'popup' : 'inline'} theme="dark" mode="inline"
    inlineCollapsed={collapsed} items={items} selectedKeys={[pathname]}
    openKeys={collapsed ? undefined : openKeys} onOpenChange={collapsed ? undefined : setOpenKeys}
    onClick={onClick} />;
}
