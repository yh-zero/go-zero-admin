import { describe, expect, it } from 'vitest';

import {
  compareRevision,
  createEditSession,
  createPermissionRefresh,
} from './permission-runtime';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
describe('permission operation isolation', () => {
  it('does not rebuild or accept contradictory content at an already committed revision', async () => {
    let fingerprint = 'one';
    const applied: string[] = [];
    const refresh = createPermissionRefresh({
      token: () => 'a',
      load: async () => ({ revision: '7', fingerprint }),
      build: async (value) => value.fingerprint,
      commit: (value) => {
        applied.push(value);
      },
    });
    await refresh.refresh();
    fingerprint = 'contradictory';
    await refresh.refresh();
    expect(applied).toEqual(['one']);
  });
  it('invalidates old confirmations after close, object change and account change', () => {
    let token = 'a';
    const session = createEditSession(() => token);
    const old = session.begin(1);
    session.begin(2);
    expect(session.valid(old)).toBe(false);
    const closed = session.capture();
    session.invalidate();
    expect(session.valid(closed)).toBe(false);
    const account = session.begin(2);
    token = 'b';
    expect(session.valid(account)).toBe(false);
  });
  it('compares decimal revisions without floating point truncation', () => {
    expect(compareRevision('9007199254740993', '9007199254740992')).toBe(1);
    expect(() => compareRevision('NaN', '1')).toThrow();
  });
  it('drops reversed responses and advances equal fingerprints without rebuilding', async () => {
    const old = deferred<{ revision: string; fingerprint: string }>();
    const newer = deferred<{ revision: string; fingerprint: string }>();
    const applied: string[] = [];
    let loads = 0;
    let builds = 0;
    const refresh = createPermissionRefresh({
      token: () => 'a',
      load: () => (++loads === 1 ? old.promise : newer.promise),
      build: async (value) => {
        builds++;
        return value.revision;
      },
      commit: (value) => {
        applied.push(value);
      },
    });
    const first = refresh.refresh();
    const second = refresh.refresh();
    newer.resolve({ revision: '10', fingerprint: 'x' });
    await second;
    old.resolve({ revision: '9', fingerprint: 'old' });
    await first;
    expect(applied).toEqual(['10']);
    const same = createPermissionRefresh({
      token: () => 'a',
      load: async () => ({ revision: String(++loads), fingerprint: 'same' }),
      build: async () => {
        builds++;
        return 'same';
      },
      commit: () => {},
    });
    await same.refresh();
    const before = builds;
    await same.refresh();
    expect(builds).toBe(before);
    expect(same.revision).toBe(String(loads));
  });
  it('cannot commit when account changes during pure candidate generation', async () => {
    let token = 'a';
    const candidate = deferred<string>();
    const applied: string[] = [];
    const refresh = createPermissionRefresh({
      token: () => token,
      load: async () => ({ revision: '1', fingerprint: 'x' }),
      build: () => candidate.promise,
      commit: (value) => {
        applied.push(value);
      },
    });
    const pending = refresh.refresh();
    await Promise.resolve();
    token = 'b';
    candidate.resolve('route');
    await pending;
    expect(applied).toEqual([]);
  });
});
