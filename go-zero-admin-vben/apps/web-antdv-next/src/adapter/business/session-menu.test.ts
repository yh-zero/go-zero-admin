import type { BackendMenu, LoginResult } from '../../api/business/session';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import { adaptMenus } from './menu';
import {
  clearSessionCache,
  isAuthenticationError,
  MissingSessionError,
  readSession,
  safeRedirect,
  saveSession,
  SESSION_HOME,
  updateSessionHome,
} from './session';

const menu = (name: string, extra: Partial<BackendMenu> = {}): BackendMenu => ({
  name,
  path: name,
  ...extra,
});
describe('server menu contract', () => {
  it.each([
    ['AntDesignOutlined', 'lucide:house'],
    ['CheckSquareOutlined', 'lucide:shield-check'],
    ['ant-design:home-outlined', 'ant-design:home-outlined'],
    ['lucide:users-round', 'lucide:users-round'],
    ['', 'lucide:folder'],
    ['unknown-legacy-icon', 'lucide:folder'],
    ['toString', 'lucide:folder'],
    ['https://example.test/icon.svg', 'lucide:folder'],
    ['lucide:', 'lucide:folder'],
    ['lucide:<svg>', 'lucide:folder'],
  ])('renders stored menu icon %j as %j', (icon, expected) => {
    const result = adaptMenus([menu('index', { meta: { icon } })], '', '888');
    expect(result.routes[0]?.meta?.icon).toBe(expected);
  });

  it('preserves hierarchy, sorting, hidden nodes and chooses authorized default leaf', () => {
    const result = adaptMenus(
      [
        menu('hidden', { hidden: true, sort: 0 }),
        menu('admin', {
          sort: 1,
          children: [menu('users', { sort: 2 }), menu('roles', { sort: 1 })],
        }),
        menu('index', { sort: 2, component: 'views/index.vue' }),
      ],
      'users',
      '888',
    );
    expect(result.homePath).toBe('/admin/users');
    expect(result.routes[1]?.children?.map((r) => r.path)).toEqual([
      '/admin/roles',
      '/admin/users',
    ]);
    expect(result.routes[0]?.meta?.hideInMenu).toBe(true);
    expect(result.routes[2]?.component).toBe('/dashboard/workspace/index.vue');
    expect(result.routes[1]?.children?.[0]?.component).toBe(
      '/business/access/pending.vue',
    );
    expect(result.routes[1]?.redirect).toBe('/admin/roles');
  });
  it('falls back from an unauthorized default and never loads arbitrary components', () => {
    const result = adaptMenus(
      [menu('users', { component: 'https://example.test/evil.vue' })],
      'missing',
      '888',
    );
    expect(result.homePath).toBe('/users');
    expect(result.routes[0]?.component).toBe('/business/access/pending.vue');
  });
  it('supports empty menus and hidden-only menus without redirect loops', () => {
    for (const menus of [null, [], [menu('hidden', { hidden: true })]]) {
      const result = adaptMenus(menus, 'missing', '888');
      expect(result.homePath).toBe(SESSION_HOME);
      expect(result.routes.at(-1)?.redirect).toBeUndefined();
    }
  });
  it('uses current-role button grants, not other roles', () => {
    expect(
      adaptMenus(
        [menu('users', { btns: { edit: 888, delete: 999 } })],
        '',
        '888',
      ).codes,
    ).toEqual(['users:edit']);
  });
  it('rejects duplicate names, paths and core routes', () => {
    for (const menus of [
      [menu('same'), menu('same')],
      [menu('one'), menu('two', { path: 'one' })],
      [menu('one'), menu('two', { path: '/ONE/' })],
      [menu('Root')],
      [menu('bad', { path: '/auth/login' })],
      [menu('bad', { path: '/AUTH/Login' })],
      [menu('bad', { path: '/Account/password' })],
      [menu('bad', { path: '/_SESSION' })],
      [menu('bad', { path: '../index' })],
      [menu('bad', { path: '//external' })],
    ])
      expect(() => adaptMenus(menus, '', '888')).toThrow();
  });
  it('resolves directory home to visible descendant and preserves absolute child paths', () => {
    const result = adaptMenus(
      [
        menu('admin', {
          children: [
            menu('hidden', { hidden: true }),
            menu('users', { path: '/users' }),
          ],
        }),
      ],
      'admin',
      '888',
    );
    expect(result.homePath).toBe('/users');
    expect(result.routes[0]?.redirect).toBe('/users');
  });
});
describe('local session and authentication errors', () => {
  const response = (): LoginResult => ({
    accessToken: 'test-token',
    accessExpire: Date.now() / 1000 + 3600,
    userInfo: {
      ID: 1,
      userName: 'tester',
      nickName: '测试',
      authorityId: 888,
      authority: { defaultRouter: 'index' },
    },
  });
  beforeEach(() => {
    localStorage.clear();
    vi.useRealTimers();
  });
  it('maps only current authority and validates token/cache binding', () => {
    const user = saveSession(response());
    expect(user.roles).toEqual(['888']);
    expect(readSession('test-token').defaultRouter).toBe('index');
    expect(() => readSession('another-token')).toThrow(MissingSessionError);
    clearSessionCache();
    expect(() => readSession('test-token')).toThrow(MissingSessionError);
  });
  it('rejects expired sessions and malformed responses', () => {
    const result = response();
    saveSession(result);
    vi.spyOn(Date, 'now').mockReturnValue((result.accessExpire + 1) * 1000);
    expect(() => readSession(result.accessToken)).toThrow(MissingSessionError);
    expect(() => saveSession(result)).toThrow();
    vi.restoreAllMocks();
  });
  it('updates only the current role homepage without changing credentials', () => {
    saveSession(response());
    updateSessionHome('test-token', 999, 'unrelated');
    expect(readSession('test-token').defaultRouter).toBe('index');
    updateSessionHome('test-token', 888, 'users');
    const session = readSession('test-token');
    expect(session.defaultRouter).toBe('users');
    expect(session.user.roles).toEqual(['888']);
    expect(session.user.token).toBe('test-token');
  });
  it('does not persist extra password or captcha fields returned by a server', () => {
    saveSession({
      ...response(),
      userInfo: { ...response().userInfo, password: 'do-not-store' },
      captcha: 'do-not-store',
    } as LoginResult);
    expect(localStorage.getItem('go-zero-admin:session:v1')).not.toContain(
      'do-not-store',
    );
  });
  it('distinguishes expired credentials from permission/network/server failures', () => {
    expect(isAuthenticationError({ response: { status: 401 } })).toBe(true);
    expect(
      isAuthenticationError({
        response: { status: 200, data: { code: 100003 } },
      }),
    ).toBe(true);
    for (const error of [
      { response: { status: 403 } },
      { response: { status: 500 } },
      new Error('network'),
    ])
      expect(isAuthenticationError(error)).toBe(false);
  });
  it('rejects external, auth, malformed and error-page redirects', () => {
    for (const path of [
      'https://example.test',
      '//example.test',
      '/%5Cexample.test',
      '%oops',
      '/auth/login',
      '/_session/error',
    ])
      expect(safeRedirect(path, SESSION_HOME)).toBe(SESSION_HOME);
    expect(safeRedirect('%2Fadmin%2Fusers', SESSION_HOME)).toBe('/admin/users');
  });
});
