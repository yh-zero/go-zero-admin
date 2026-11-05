import type { QueryClient } from '@tanstack/react-query';
import { useAuthStore } from '../stores/auth';
import { queryKeys } from './queryKeys';

async function invalidateData(client: QueryClient, roots: string[]): Promise<void> {
  // 无缓存的在途请求也必须取消，避免旧响应在写入后重新填入缓存。
  await Promise.all(roots.map((root) => client.cancelQueries({ queryKey: [root] })));
  await Promise.all(roots.map((root) => client.invalidateQueries({ queryKey: [root] })));
}

export function invalidateTenantData(client: QueryClient): Promise<void> {
  return invalidateData(client, [queryKeys.tenantList]);
}

export function invalidateUserData(client: QueryClient): Promise<void> {
  return invalidateAccessData(client, [queryKeys.users, queryKeys.user, queryKeys.profile, queryKeys.files,
    queryKeys.roles, queryKeys.role, queryKeys.depts, queryKeys.myDepts, queryKeys.myMenus, queryKeys.permCodes]);
}

async function invalidateAccessData(client: QueryClient, roots: string[]): Promise<void> {
  const { sessionVersion, departmentVersion } = useAuthStore.getState();
  useAuthStore.getState().clearPermissions();
  await invalidateData(client, roots);
  const permissions = client.getQueryState<string[]>([queryKeys.permCodes]);
  // 只有当前会话的完整刷新成功后恢复写权限；未激活或失败的查询继续禁写。
  const current = useAuthStore.getState();
  if (sessionVersion === current.sessionVersion && departmentVersion === current.departmentVersion && permissions?.status === 'success'
    && permissions.fetchStatus === 'idle' && !permissions.isInvalidated && permissions.data) {
    useAuthStore.getState().setPermCodes(permissions.data);
  }
}

export function invalidateRoleData(client: QueryClient): Promise<void> {
  return invalidateAccessData(client, [queryKeys.roles, queryKeys.role, queryKeys.roleOptions, queryKeys.profile,
    queryKeys.myDepts, queryKeys.myMenus, queryKeys.permCodes]);
}

export function invalidateMenuData(client: QueryClient): Promise<void> {
  return invalidateAccessData(client, [queryKeys.menus, queryKeys.roleMenuOptions, queryKeys.roles, queryKeys.role,
    queryKeys.myMenus, queryKeys.permCodes]);
}

export function invalidateAppData(client: QueryClient): Promise<void> {
  return invalidateData(client, [queryKeys.appUsers, queryKeys.appUser, queryKeys.appUserOptions, queryKeys.appSessions,
    queryKeys.appMemberships, queryKeys.appFeedback, queryKeys.files]);
}
