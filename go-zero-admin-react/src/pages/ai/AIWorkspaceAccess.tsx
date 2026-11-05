import { Fragment, useMemo } from 'react';
import type { ReactNode } from 'react';
import { Result, Skeleton } from 'antd';
import { useAuthStore } from '../../stores/auth';
import type { AiWorkspaceScope } from './workspaceQueries';

interface AccessProps { code: string; title: string; children: (scope: AiWorkspaceScope) => ReactNode }
interface AccessViewProps extends AccessProps { scope: AiWorkspaceScope; permissionsReady: boolean; permCodes: readonly string[] }

export function AIWorkspaceAccess({ code, title, children }: AccessProps) {
  const auth = useAuthStore();
  const scope: AiWorkspaceScope = useMemo(() => [auth.sessionId, auth.sessionVersion, auth.departmentVersion],
    [auth.sessionId, auth.sessionVersion, auth.departmentVersion]);
  return <AIWorkspaceAccessView code={code} title={title} scope={scope}
    permissionsReady={auth.permissionsReady && !auth.isLoggingOut} permCodes={auth.permCodes}>{children}</AIWorkspaceAccessView>;
}

export function AIWorkspaceAccessView({ code, title, children, scope, permissionsReady, permCodes }: AccessViewProps) {
  if (!permissionsReady) return <div role="status" aria-label={`${title}加载中`}><Skeleton active /></div>;
  if (!permCodes.includes(code)) return <Result status="403" title={`无权查看${title}`} subTitle="请联系管理员授予相应页面权限。" />;
  // 同一页面切换身份或部门时销毁选择、表单与正在确认的审批断点。
  return <Fragment key={JSON.stringify(scope)}>{children(scope)}</Fragment>;
}
