import { useAccessStore, useUserStore } from '@vben/stores';

import { message, Modal } from 'antdv-next';

import { notifyPermissionChange } from '#/router/permission-refresh-events';
export interface TableQuery {
  page: { currentPage: number; pageSize: number };
}

export const statusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 2 },
];
export const methods = [
  'GET',
  'POST',
  'PUT',
  'DELETE',
  'PATCH',
  'HEAD',
  'OPTIONS',
].map((value) => ({ label: value, value }));
export function usePermission(menu: string) {
  const access = useAccessStore();
  return (button: string) => access.accessCodes.includes(`${menu}:${button}`);
}
export function flattenTree<T extends { children?: T[] }>(rows: T[]): T[] {
  return rows.flatMap((row) => [row, ...flattenTree(row.children ?? [])]);
}
export function confirmAction(
  title: string,
  action: () => Promise<void>,
  content?: string,
) {
  Modal.confirm({
    title,
    content,
    okText: '确认',
    cancelText: '取消',
    onOk: action,
  });
}
// A full reload re-runs authenticated menu generation and retains the original Vben menus.
export function refreshAccess() {
  message.success('已保存，正在更新菜单与按钮权限');
  notifyPermissionChange();
}

export function isCurrentRole(roleId: number) {
  return useUserStore().userInfo?.roles?.includes(String(roleId)) ?? false;
}

export function refreshRoleAccess(roleId: number) {
  if (isCurrentRole(roleId)) refreshAccess();
  else {
    notifyPermissionChange();
    message.success('授权已保存');
  }
}
