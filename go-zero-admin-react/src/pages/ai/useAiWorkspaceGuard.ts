import { useCallback, useEffect, useRef, useState } from 'react';
import { useAuthStore } from '../../stores/auth';
import type { AiWorkspaceScope } from './workspaceQueries';

export function isAiWorkspaceCurrent(scope: AiWorkspaceScope, readCode: string): boolean {
  const current = useAuthStore.getState();
  return !!current.token && !current.isLoggingOut && current.sessionId === scope[0]
    && current.sessionVersion === scope[1] && current.departmentVersion === scope[2]
    && current.permissionsReady && current.permCodes.includes(readCode);
}

export function useAiWorkspaceGuard(scope: AiWorkspaceScope, readCode: string) {
  const auth = useAuthStore();
  const mounted = useRef(true);
  const busy = useRef(false);
  const [writing, setWriting] = useState(false);
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  const [sessionId, sessionVersion, departmentVersion] = scope;
  const isCurrent = useCallback(() => mounted.current
    && isAiWorkspaceCurrent([sessionId, sessionVersion, departmentVersion], readCode),
  [sessionId, sessionVersion, departmentVersion, readCode]);
  const beginWrite = useCallback((code: string) => {
    if (busy.current || !isCurrent() || !useAuthStore.getState().permCodes.includes(code)) return false;
    // React 尚未提交 disabled 状态时，ref 也能挡住双击或 Enter 重入。
    busy.current = true;
    setWriting(true);
    return true;
  }, [isCurrent]);
  const finishWrite = useCallback(() => {
    busy.current = false;
    if (isCurrent()) setWriting(false);
  }, [isCurrent]);
  return { writing, isCurrent, beginWrite, finishWrite,
    can: (code: string) => auth.permissionsReady && isCurrent() && useAuthStore.getState().permCodes.includes(code) };
}
