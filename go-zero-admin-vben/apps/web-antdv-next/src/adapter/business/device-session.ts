import type { DeviceSession } from '../../api/business/device-sessions';

export function formatSessionTime(value: string) {
  const time = new Date(value);
  return Number.isNaN(time.getTime())
    ? '—'
    : time.toLocaleString('zh-CN', { hour12: false });
}

// Clear local state only after the target was revoked; never revoke all devices.
export async function revokeListedDevice(
  session: Pick<DeviceSession, 'current' | 'id'>,
  revoke: (id: string) => Promise<unknown>,
  exitCurrentSession: () => Promise<unknown>,
  refresh: () => Promise<unknown>,
) {
  await revoke(session.id);
  if (session.current) await exitCurrentSession();
  else await refresh();
}
