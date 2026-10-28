import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getPermissionEdit, getPermissionSnapshot } from './permissions';
const mocks = vi.hoisted(() => ({ get: vi.fn() }));
vi.mock('#/api/request', () => ({ requestClient: mocks }));
describe('permission snapshot wire contract', () => {
  beforeEach(() => vi.resetAllMocks());
  it('normalizes empty repeated protobuf fields so no-button sessions can generate routes', async () => {
    mocks.get.mockResolvedValue({
      revision: '7',
      fingerprint: 'empty',
      authorityId: 22,
      defaultRouter: '',
      menus: null,
      codes: null,
    });
    expect(await getPermissionSnapshot('token')).toEqual({
      revision: '7',
      fingerprint: 'empty',
      authorityId: 22,
      defaultRouter: '',
      menus: [],
      codes: [],
    });
  });
  it('normalizes edit collections without mixing separate resource/grant requests', async () => {
    mocks.get.mockResolvedValue({
      revision: '7',
      authorityId: 22,
      kind: 'menu',
      menus: null,
      apis: null,
      departments: null,
      menuIds: null,
      menuBtnIds: null,
      policies: null,
      dataScope: { authorityId: 22, scope: 'self', departmentIds: null },
    });
    const snapshot = await getPermissionEdit(22, 'menu', 'token');
    expect(snapshot.menuIds).toEqual([]);
    expect(snapshot.dataScope.departmentIds).toEqual([]);
    expect(mocks.get).toHaveBeenCalledExactlyOnceWith(
      '/v1/sys/permissions/edit',
      {
        params: { authorityId: 22, kind: 'menu' },
        headers: { Authorization: 'Bearer token' },
      },
    );
  });
  it('rejects a wrong role or invalid revision rather than enabling the editor', async () => {
    mocks.get.mockResolvedValue({
      authorityId: 11,
      kind: 'menu',
      revision: '7',
    });
    await expect(getPermissionEdit(22, 'menu')).rejects.toThrow('不匹配');
    mocks.get.mockResolvedValue({
      authorityId: 22,
      kind: 'menu',
      revision: 'NaN',
    });
    await expect(getPermissionEdit(22, 'menu')).rejects.toThrow('不匹配');
  });
});
