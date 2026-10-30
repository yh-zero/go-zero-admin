import { Layout, Menu, Dropdown, Avatar, Space } from 'antd';
import { useNavigate, Outlet, useLocation } from 'react-router-dom';
import {
  DashboardOutlined,
  UserOutlined,
  SafetyOutlined,
  MenuOutlined,
  PartitionOutlined,
  BookOutlined,
  FileOutlined,
  FileSearchOutlined,
  ApiOutlined,
  LogoutOutlined,
} from '@ant-design/icons';
import { useEffect, useState } from 'react';
import { useAuthStore } from '../stores/auth';
import { fetchMyMenus } from '../api/auth';
import type { MenuInfo } from '../api/types';

// 菜单图标映射（后端存图标名字符串，前端映射组件）
const icons: Record<string, React.ReactNode> = {
  dashboard: <DashboardOutlined />,
  user: <UserOutlined />,
  role: <SafetyOutlined />,
  menu: <MenuOutlined />,
  dept: <PartitionOutlined />,
  dict: <BookOutlined />,
  file: <FileOutlined />,
  audit: <FileSearchOutlined />,
  api: <ApiOutlined />,
};

// 菜单树 → antd Menu items
function toMenuItems(menus: MenuInfo[]): NonNullable<React.ComponentProps<typeof Menu>['items']> {
  return menus
    .filter((m) => m.visible && m.status === 1 && m.type !== 3) // 隐藏按钮类型
    .sort((a, b) => a.sort - b.sort)
    .map((m) => ({
      key: m.path || String(m.id),
      icon: icons[m.icon] || <DashboardOutlined />,
      label: m.title,
      children: m.children ? toMenuItems(m.children) : undefined,
    }));
}

export default function BasicLayout() {
  const [menus, setMenus] = useState<MenuInfo[]>([]);
  const navigate = useNavigate();
  const location = useLocation();
  const logout = useAuthStore((s) => s.logout);

  useEffect(() => {
    fetchMyMenus().then(setMenus).catch(() => {});
  }, []);

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Layout.Sider theme="dark" width={220}>
        <div style={{ color: '#fff', padding: 16, fontSize: 18, fontWeight: 600 }}>
          go-zero-admin
        </div>
        <Menu
          theme="dark"
          mode="inline"
          items={toMenuItems(menus)}
          selectedKeys={[location.pathname]}
          onClick={({ key }) => key.startsWith('/') && navigate(key)}
        />
      </Layout.Sider>
      <Layout>
        <Layout.Header style={{ background: '#fff', padding: '0 24px', textAlign: 'right' }}>
          <Dropdown
            menu={{
              items: [{ key: 'logout', icon: <LogoutOutlined />, label: '退出登录' }],
              onClick: ({ key }) => {
                if (key === 'logout') {
                  logout();
                  navigate('/login');
                }
              },
            }}
          >
            <Space style={{ cursor: 'pointer' }}>
              <Avatar icon={<UserOutlined />} />
              admin
            </Space>
          </Dropdown>
        </Layout.Header>
        <Layout.Content style={{ margin: 24 }}>
          <Outlet />
        </Layout.Content>
      </Layout>
    </Layout>
  );
}
