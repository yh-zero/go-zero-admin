import { lazy } from 'react';
export const pages = {
  dashboard: lazy(() => import('../pages/dashboard')), user: lazy(() => import('../pages/user')),
  role: lazy(() => import('../pages/role')), dept: lazy(() => import('../pages/dept')),
  menu: lazy(() => import('../pages/menu')), dict: lazy(() => import('../pages/dict')),
  api: lazy(() => import('../pages/api')), file: lazy(() => import('../pages/file')),
  audit: lazy(() => import('../pages/audit')), profile: lazy(() => import('../pages/profile')),
};
export const pageLinks = [
  { name: 'dashboard', path: '/dashboard', title: '工作台', icon: 'dashboard' },
  { name: 'user', path: '/sys/user', title: '用户管理', icon: 'user' },
  { name: 'role', path: '/sys/role', title: '角色管理', icon: 'role' },
  { name: 'dept', path: '/sys/dept', title: '部门管理', icon: 'dept' },
  { name: 'menu', path: '/sys/menu', title: '菜单管理', icon: 'menu' },
  { name: 'dict', path: '/sys/dict', title: '字典管理', icon: 'dict' },
  { name: 'api', path: '/sys/api', title: 'API 资源', icon: 'api' },
  { name: 'file', path: '/sys/file', title: '文件管理', icon: 'file' },
  { name: 'audit', path: '/sys/audit', title: '审计日志', icon: 'audit' },
  { name: 'profile', path: '/sys/profile', title: '个人中心', icon: 'user' },
] as const;
