import { useEffect, useState } from 'react';
import { Alert, Avatar, Button, Dropdown, Layout, Menu, Space, Typography } from 'antd';
import { LogoutOutlined, UserOutlined, MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { fetchMyMenus, fetchMyPermCodes } from '../api/auth';
import { fetchProfile } from '../api/profile';
import { useAuthStore } from '../stores/auth';
import { clearSession } from '../utils/session';
import { pageLinks } from '../router/registry';
import { fallbackItems, toMenuItems } from './navigation';

export default function BasicLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const setPermCodes = useAuthStore(state => state.setPermCodes);
  const clearPermissions = useAuthStore(state => state.clearPermissions);
  const menus = useQuery({ queryKey: ['my-menus'], queryFn: fetchMyMenus });
  const permissions = useQuery({ queryKey: ['perm-codes'], queryFn: fetchMyPermCodes });
  const profile = useQuery({ queryKey: ['profile'], queryFn: fetchProfile });
  useEffect(() => {
    if (permissions.isError) clearPermissions();
    else if (permissions.data) setPermCodes(permissions.data);
  }, [permissions.data, permissions.isError, setPermCodes, clearPermissions]);
  const title = pageLinks.find(page => page.path === location.pathname)?.title ?? '管理工作台';
  const remoteItems = toMenuItems(menus.data ?? []);
  const menuItems = remoteItems.length ? [{ key: '/dashboard', label: '工作台' }, ...remoteItems.filter(item => item?.key !== '/dashboard')] : fallbackItems;
  return <Layout className="app-layout">
    <Layout.Sider width={220} collapsible collapsed={collapsed} onCollapse={setCollapsed} breakpoint="lg" className="app-sider">
      <div className="app-brand">{collapsed ? 'GZ' : 'go-zero-admin'}</div>
      <Menu theme="dark" mode="inline" items={menuItems} selectedKeys={[location.pathname]} onClick={({ key }) => { if (key.startsWith('/')) navigate(key); }} />
    </Layout.Sider>
    <Layout><Layout.Header className="app-header"><Space>
      <Button type="text" aria-label={collapsed ? '展开导航' : '收起导航'} icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={() => setCollapsed(!collapsed)} />
      <Typography.Text strong>{title}</Typography.Text>
    </Space><Dropdown menu={{ items: [{ key: 'profile', label: '个人中心', icon: <UserOutlined /> }, { key: 'logout', label: '退出登录', icon: <LogoutOutlined /> }], onClick: ({ key }) => { if (key === 'logout') { clearSession(); navigate('/login', { replace: true }); } else navigate('/sys/profile'); } }}>
      <Button type="text"><Space><Avatar size="small" icon={<UserOutlined />} />{profile.data?.nickname || profile.data?.username || '当前用户'}</Space></Button>
    </Dropdown></Layout.Header>
    <Layout.Content className="app-content">
      {menus.isError && <Alert className="integration-alert" type="warning" showIcon title="导航暂不可用，已显示接入页面入口；具体访问仍需后端授权。" action={<Button size="small" onClick={() => void menus.refetch()}>重试</Button>} />}
      {permissions.isError && <Alert className="integration-alert" type="warning" showIcon title="操作权限加载失败，管理写操作已隐藏。" action={<Button size="small" onClick={() => void permissions.refetch()}>重试</Button>} />}
      <Outlet />
    </Layout.Content></Layout>
  </Layout>;
}
