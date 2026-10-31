import { Fragment, useEffect } from 'react';
import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../stores/auth';
import { clearSession } from '../utils/session';
export default function RequireAuth({ children }: { children: ReactNode }) {
  const { token, expiresAt, refreshToken, refreshExpiresAt, sessionVersion, departmentVersion } = useAuthStore();
  const location = useLocation();
  useEffect(() => {
    if (!token) return;
    let timer: number | undefined;
    const checkExpiry = () => {
      const remaining = Math.max(expiresAt, refreshToken ? refreshExpiresAt : 0) * 1000 - Date.now();
      if (remaining <= 0) { void clearSession({ sessionVersion, token }); return; }
      // 超出浏览器单次定时器上限时只重新检查，不能提前结束长会话。
      timer = window.setTimeout(checkExpiry, Math.min(remaining, 2_147_483_647));
    };
    checkExpiry();
    return () => window.clearTimeout(timer);
  }, [token, expiresAt, refreshToken, refreshExpiresAt, sessionVersion]);
  return token ? <Fragment key={departmentVersion}>{children}</Fragment> : <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
}
