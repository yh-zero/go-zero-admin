import type { ReactNode } from 'react';
import { useAuthStore } from '../stores/auth';
export function Auth({ code, children }: { code: string; children: ReactNode }) {
  const allowed = useAuthStore((state) => state.permissionsReady && state.permCodes.includes(code));
  return allowed ? <>{children}</> : null;
}
