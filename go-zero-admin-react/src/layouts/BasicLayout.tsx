import { useEffect, useState } from 'react';
import { Alert, Button, Drawer, Grid, Layout } from 'antd';
import { DashboardOutlined } from '@ant-design/icons';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { fetchMyMenus, fetchMyPermCodes } from '../api/auth';
import { fetchProfile } from '../api/profile';
import { useAuthStore } from '../stores/auth';
import { pageLinks } from '../router/registry';
import { fallbackItems, getMenuAncestorKeys, toMenuItems } from './navigation';
import { BrandMark } from '../components/BrandMark';
import { WorkspaceHeader } from './WorkspaceHeader';
import { ResizableSidebar } from './ResizableSidebar';
import { WorkspaceMenu } from './WorkspaceMenu';

export default function BasicLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const screens = Grid.useBreakpoint();
  const isMobile = screens.lg === false;
  const navigate = useNavigate();
  const location = useLocation();
  const setPermCodes = useAuthStore(state => state.setPermCodes);
  const clearPermissions = useAuthStore(state => state.clearPermissions);
  const departmentVersion = useAuthStore(state => state.departmentVersion);
  const menus = useQuery({ queryKey: ['my-menus'], queryFn: fetchMyMenus });
  const permissions = useQuery({ queryKey: ['perm-codes'], queryFn: fetchMyPermCodes });
  const profile = useQuery({ queryKey: ['profile'], queryFn: fetchProfile });
  useEffect(() => {
    if (useAuthStore.getState().departmentVersion !== departmentVersion) return;
    if (permissions.isFetching || permissions.isError) clearPermissions();
    else if (permissions.data) setPermCodes(permissions.data);
  }, [permissions.data, permissions.dataUpdatedAt, permissions.isFetching, permissions.isError, setPermCodes, clearPermissions, departmentVersion]);
  const title = pageLinks.find(page => page.path === location.pathname)?.title ?? '管理工作台';
  const remoteItems = toMenuItems(menus.data ?? []);
  const menuItems = remoteItems.length ? [{ key: '/dashboard', label: '工作台', icon: <DashboardOutlined /> }, ...remoteItems.filter(item => item?.key !== '/dashboard')] : fallbackItems;
  // 异步菜单或路由祖先变化时重建初始展开状态，调宽和同目录跳转保留用户选择。
  const routeParentKeys = getMenuAncestorKeys(menuItems, location.pathname) ?? [];
  const navigation = <>
    <div className="app-brand"><BrandMark /><span className="brand-copy"><strong>go-zero-admin</strong><span>管理控制台</span></span></div>
    <div className="navigation-caption">工作空间</div>
    <WorkspaceMenu key={JSON.stringify(routeParentKeys)} parentKeys={routeParentKeys}
      collapsed={!isMobile && collapsed} items={menuItems} pathname={location.pathname}
      onClick={({ key }) => { if (key.startsWith('/')) { navigate(key); setMobileOpen(false); } }} />
  </>;
  return <Layout className="app-layout">
    {isMobile ? <Drawer open={mobileOpen} onClose={() => setMobileOpen(false)} placement="left" size={240}
      title="导航菜单" rootClassName="navigation-drawer">{navigation}</Drawer>
      : <ResizableSidebar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)}>{navigation}</ResizableSidebar>}
    <Layout className="app-workspace"><WorkspaceHeader title={title} name={profile.data?.nickname || profile.data?.username || '当前用户'}
      navigationOpen={isMobile ? mobileOpen : !collapsed} onToggle={() => isMobile ? setMobileOpen(!mobileOpen) : setCollapsed(!collapsed)} />
    <Layout.Content className="app-content">
      <div className="content-container">
      {menus.isError && <Alert className="integration-alert" type="warning" showIcon title="导航暂不可用，已显示接入页面入口；具体访问仍需后端授权。" action={<Button size="small" onClick={() => void menus.refetch()}>重试</Button>} />}
      {permissions.isError && <Alert className="integration-alert" type="warning" showIcon title="操作权限加载失败，管理写操作已隐藏。" action={<Button size="small" onClick={() => void permissions.refetch()}>重试</Button>} />}
      <Outlet />
      </div>
    </Layout.Content></Layout>
  </Layout>;
}
