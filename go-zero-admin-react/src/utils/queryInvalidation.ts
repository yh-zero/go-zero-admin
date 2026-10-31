import type { QueryClient } from '@tanstack/react-query';
import { useAuthStore } from '../stores/auth';

async function invalidateData(client: QueryClient, roots: string[]): Promise<void> {
  // 无缓存的在途请求也必须取消，避免旧响应在写入后重新填入缓存。
  await Promise.all(roots.map((root) => client.cancelQueries({ queryKey: [root] })));
  await Promise.all(roots.map((root) => client.invalidateQueries({ queryKey: [root] })));
}

export function invalidateUserData(client: QueryClient): Promise<void> {
  return invalidateData(client, ['users', 'user', 'profile']);
}

async function invalidateAccessData(client: QueryClient, roots: string[]): Promise<void> {
  const token = useAuthStore.getState().token;
  useAuthStore.getState().clearPermissions();
  await invalidateData(client, roots);
  const permissions = client.getQueryState<string[]>(['perm-codes']);
  // 只有当前会话的完整刷新成功后恢复写权限；未激活或失败的查询继续禁写。
  if (token === useAuthStore.getState().token && permissions?.status === 'success'
    && permissions.fetchStatus === 'idle' && !permissions.isInvalidated && permissions.data) {
    useAuthStore.getState().setPermCodes(permissions.data);
  }
}

export function invalidateRoleData(client: QueryClient): Promise<void> {
  return invalidateAccessData(client, ['roles', 'role', 'my-menus', 'perm-codes']);
}

export function invalidateMenuData(client: QueryClient): Promise<void> {
  return invalidateAccessData(client, ['menus', 'role-menu-options', 'roles', 'role', 'my-menus', 'perm-codes']);
}
