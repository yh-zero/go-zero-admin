import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  isCurrentSessionRequest,
  mapBackendUser,
  readSession,
  refreshSessionUser,
  saveSession,
} from './session';

describe('current user response replaces cached display data', () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
  });
  it('uses the bundled avatar when a user has no avatar URL', () => {
    const user = { ID: 1, userName: 'admin', authorityId: 1, headerImg: '' };
    expect(mapBackendUser(user, 'test').avatar).toBe('/default-avatar.svg');
  });
  it('resolves the database default avatar marker under the application base', async () => {
    vi.stubEnv('BASE_URL', '/console/');
    vi.resetModules();
    const session = await import('./session');
    const user = {
      ID: 1,
      userName: 'admin',
      authorityId: 1,
      headerImg: '/default-avatar.svg',
    };
    expect(session.mapBackendUser(user, 'test').avatar).toBe(
      '/console/default-avatar.svg',
    );
    expect(
      session.mapBackendUser(
        { ...user, headerImg: 'https://files.example.test/avatar.png' },
        'test',
      ).avatar,
    ).toBe('https://files.example.test/avatar.png');
  });
  it('refreshes identity and homepage without extending token expiry or storing secrets', () => {
    const expiry = Math.floor(Date.now() / 1000) + 1000;
    saveSession({
      accessToken: 'test',
      accessExpire: expiry,
      userInfo: { ID: 1, userName: 'admin', authorityId: 1 },
    });
    const current = {
      ID: 1,
      userName: 'admin',
      nickName: 'New name',
      authorityId: 1,
      authority: { defaultRouter: 'menu' },
      password: 'never-store',
      sessionVersion: 10,
    };
    const display = refreshSessionUser(current, 'test');
    expect(display.realName).toBe('New name');
    expect(readSession('test').defaultRouter).toBe('menu');
    expect(readSession('test').expiresAt).toBe(expiry);
    expect(localStorage.getItem('go-zero-admin:session:v1')).not.toContain(
      'never-store',
    );
    expect(localStorage.getItem('go-zero-admin:session:v1')).not.toContain(
      'sessionVersion',
    );
    expect(() => refreshSessionUser(current, 'another-token')).toThrow(
      '登录信息已失效，请重新登录',
    );
  });
});

describe('authentication failures are bound to their original request', () => {
  it('ignores delayed failures for a previous token or public request', () => {
    const error = {
      response: { status: 401 },
      config: { headers: { Authorization: 'Bearer old-token' } },
    };
    expect(isCurrentSessionRequest(error, 'old-token')).toBe(true);
    expect(isCurrentSessionRequest(error, 'new-token')).toBe(false);
    expect(isCurrentSessionRequest(error, null)).toBe(false);
    expect(
      isCurrentSessionRequest({ response: { status: 401 } }, 'new-token'),
    ).toBe(false);
  });
});
