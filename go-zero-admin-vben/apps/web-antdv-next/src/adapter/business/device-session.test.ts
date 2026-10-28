import { describe, expect, it, vi } from 'vitest';

import { revokeListedDevice } from './device-session';
describe('single device revocation effects', () => {
  it('clears this browser only after its device was revoked', async () => {
    const exit = vi.fn();
      const refresh = vi.fn();
    let complete: (() => void) | undefined;
    const revoke = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          complete = resolve;
        }),
    );
    const pending = revokeListedDevice(
      { id: 'current', current: true },
      revoke,
      exit,
      refresh,
    );
    expect(exit).not.toHaveBeenCalled();
    complete?.();
    await pending;
    expect(exit).toHaveBeenCalledOnce();
    expect(refresh).not.toHaveBeenCalled();
  });
  it('refreshes other-device changes and preserves the current login', async () => {
    const exit = vi.fn();
      const refresh = vi.fn();
      const revoke = vi.fn().mockResolvedValue(undefined);
    await revokeListedDevice(
      { id: 'other', current: false },
      revoke,
      exit,
      refresh,
    );
    expect(exit).not.toHaveBeenCalled();
    expect(refresh).toHaveBeenCalledOnce();
  });
  it('does not clear local login or display a refreshed success when revocation fails', async () => {
    const exit = vi.fn();
      const refresh = vi.fn();
      const revoke = vi.fn().mockRejectedValue(new Error('offline'));
    await expect(
      revokeListedDevice(
        { id: 'current', current: true },
        revoke,
        exit,
        refresh,
      ),
    ).rejects.toThrow('offline');
    expect(exit).not.toHaveBeenCalled();
    expect(refresh).not.toHaveBeenCalled();
  });
});
