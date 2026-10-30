import type { ReactNode } from 'react';
import { useAuthStore } from '../stores/auth';

/** 按钮权限：code 在 permCodes 中才渲染子元素 */
export function Auth({ code, children }: { code: string; children: ReactNode }) {
  const permCodes = useAuthStore((s) => s.permCodes);
  if (!permCodes.includes(code)) return null;
  return <>{children}</>;
}
