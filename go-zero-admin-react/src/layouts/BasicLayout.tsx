import { useEffect, useState } from 'react';
import { Alert, Button, Drawer, Grid, Layout, Spin } from 'antd';
import { DashboardOutlined } from '@ant-design/icons';
import { Navigate, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { fetchMyMenus, fetchMyPermCodes } from '../api/auth';
import { fetchProfile } from '../api/profile';
import { fetchMyDepts } from '../api/dept';
import { useAuthStore } from '../stores/auth';
import { pageLinks } from '../router/registry';
import { getMenuAncestorKeys, toMenuItems } from './navigation';
import { getAccessiblePagePaths, safeMenuPath } from '../router/menuRoutes';
import { switchDepartment } from '../utils/request';
import { QueryState } from '../components/QueryState';
import { BrandMark } from '../components/BrandMark';
import { WorkspaceHeader } from './WorkspaceHeader';
import { ResizableSidebar } from './ResizableSidebar';
import { WorkspaceMenu } from './WorkspaceMenu';
import { queryKeys } from '../utils/queryKeys';

export default function BasicLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const screens = Grid.useBreakpoint();
  const isMobile = screens.lg === false;
  const navigate = useNavigate();
  const location = useLocation();
  const setPermCodes = useAuthStore(state => state.setPermCodes);
  const clearPermissions = useAuthStore(state => state.clearPermissions);
  const selectedDeptId = useAuthStore(state => state.selectedDeptId);
  const departmentVersion = useAuthStore(state => state.departmentVersion);
  const departments = useQuery({ queryKey: [queryKeys.myDepts], queryFn: fetchMyDepts });
  const departmentReady = departments.isSuccess && (selectedDeptId === null || departments.data.some(dept => dept.id === selectedDeptId));
  const menus = useQuery({ queryKey: [queryKeys.myMenus], queryFn: fetchMyMenus, enabled: departmentReady });
  const permissions = useQuery({ queryKey: [queryKeys.permCodes], queryFn: fetchMyPermCodes, enabled: departmentReady });
  const profile = useQuery({ queryKey: [queryKeys.profile], queryFn: fetchProfile, enabled: departmentReady });
  useEffect(() => {
    if (departments.isSuccess && selectedDeptId !== null && !departmentReady) {
      void switchDepartment(null).catch(() => undefined);
    }
  }, [departments.isSuccess, selectedDeptId, departmentReady]);
  useEffect(() => {
    if (useAuthStore.getState().departmentVersion !== departmentVersion) return;
    if (!departmentReady || permissions.isFetching || permissions.isError) clearPermissions();
    else if (permissions.data) setPermCodes(permissions.data);
  }, [permissions.data, permissions.dataUpdatedAt, permissions.isFetching, permissions.isError, setPermCodes, clearPermissions, departmentVersion, departmentReady]);
  const title = pageLinks.find(page => page.path === location.pathname)?.title ?? '管理工作台';
  const accessiblePaths = getAccessiblePagePaths(menus.isSuccess && departmentReady ? menus.data ?? [] : []);
  const remoteItems = toMenuItems(menus.isSuccess && departmentReady ? menus.data ?? [] : []);
  const menuItems = [{ key: '/dashboard', label: '工作台', icon: <DashboardOutlined /> }, ...remoteItems.filter(item => item?.key !== '/dashboard')];
  const pageContent = departments.isError
    ? <QueryState error={departments.error} onRetry={() => departments.refetch()} />
    : !departmentReady || menus.isPending ? <Spin className="route-spinner" />
      : menus.isError ? <QueryState error={menus.error} onRetry={() => menus.refetch()} />
        : accessiblePaths.has(safeMenuPath(location.pathname) ?? '') ? <Outlet /> : <Navigate to="/dashboard" replace />;
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
    <Layout className="app-workspace"><WorkspaceHeader title={title} name={profile.data?.nickname || profile.data?.username || '当前用户'} canViewProfile={accessiblePaths.has('/sys/profile')}
      navigationOpen={isMobile ? mobileOpen : !collapsed} onToggle={() => isMobile ? setMobileOpen(!mobileOpen) : setCollapsed(!collapsed)} />
    <Layout.Content className="app-content">
      <div className="content-container">
      {permissions.isError && <Alert className="integration-alert" type="warning" showIcon title="操作权限加载失败，管理写操作已隐藏。" action={<Button size="small" onClick={() => void permissions.refetch()}>重试</Button>} />}
      {pageContent}
      </div>
    </Layout.Content></Layout>
  </Layout>;
}
