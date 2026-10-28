import { describe, expect, it } from 'vitest';

import { assertIndependentMenus } from './menu-conflicts';

describe('original pages and backend menus', () => {
  const local = [
    {
      name: 'Dashboard',
      path: '/dashboard',
      children: [{ name: 'Workspace', path: 'workspace' }],
    },
  ];

  it('allows original pages alongside independent backend menus without changing either tree', () => {
    const backend = [{ name: 'users', path: '/admin/users' }];
    const before = JSON.stringify({ local, backend });
    expect(() => assertIndependentMenus(local, backend)).not.toThrow();
    expect(JSON.stringify({ local, backend })).toBe(before);
  });

  it('rejects a backend name that mixed mode would silently merge with an original page', () => {
    expect(() =>
      assertIndependentMenus(local, [
        { name: 'Workspace', path: '/admin/users' },
      ]),
    ).toThrow('路由冲突');
  });

  it('rejects a full path collision even when a local child path is relative', () => {
    expect(() =>
      assertIndependentMenus(local, [
        { name: 'other', path: '/dashboard/workspace' },
      ]),
    ).toThrow('路由冲突');
  });
  it('rejects differently cased paths that Vue Router would match to the same page', () => {
    expect(() =>
      assertIndependentMenus(local, [
        { name: 'other', path: '/Dashboard/WORKSPACE/' },
      ]),
    ).toThrow('路由冲突');
  });
});
