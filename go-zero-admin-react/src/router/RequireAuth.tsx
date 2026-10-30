import { useEffect } from 'react';
import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../stores/auth';
import { clearSession } from '../utils/session';
export default function RequireAuth({ children }: { children: ReactNode }) {
  const { token, expiresAt } = useAuthStore();
  const location = useLocation();
  useEffect(() => {
    if (!token) return;
    const remaining = expiresAt * 1000 - Date.now();
    if (remaining <= 0) { clearSession(); return; }
    const timer = window.setTimeout(clearSession, Math.min(remaining, 2_147_483_647));
    return () => window.clearTimeout(timer);
  }, [token, expiresAt]);
  return token ? <>{children}</> : <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
}
