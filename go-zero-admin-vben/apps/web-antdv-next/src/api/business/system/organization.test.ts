import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  deleteDepartment,
  deletePosition,
  getDepartments,
  getMembership,
  getPositions,
  getRoleDataScope,
  updateMembership,
  updateRoleDataScope,
} from './organization';

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}));
vi.mock('#/api/request', () => ({ requestClient: mocks }));

describe('organization HTTP contract', () => {
  beforeEach(() => vi.resetAllMocks());
  it('uses list query parameters and normalizes empty backend arrays', async () => {
    mocks.get.mockResolvedValue({ list: null });
    expect(await getDepartments({ keyword: '研发', status: 1 })).toEqual([]);
    expect(mocks.get).toHaveBeenLastCalledWith(
      '/v1/sys/organization/departments',
      { params: { keyword: '研发', status: 1 } },
    );
    expect(await getPositions()).toEqual([]);
    expect(mocks.get).toHaveBeenLastCalledWith(
      '/v1/sys/organization/positions',
      { params: {} },
    );
  });
  it('sends DELETE ids in the body and preserves explicit clearing of assignments', async () => {
    await deleteDepartment(7);
    await deletePosition(8);
    expect(mocks.delete.mock.calls).toEqual([
      ['/v1/sys/organization/departments', { data: { id: 7 } }],
      ['/v1/sys/organization/positions', { data: { id: 8 } }],
    ]);
    await updateMembership({ userId: 3, departmentId: 0, positionIds: [] });
    expect(mocks.put).toHaveBeenLastCalledWith(
      '/v1/sys/organization/membership',
      { userId: 3, departmentId: 0, positionIds: [] },
    );
    await updateRoleDataScope(
      {
        authorityId: 9,
        scope: 'self',
        departmentIds: [],
        expectedRevision: '7',
      },
      'token',
    );
    expect(mocks.put).toHaveBeenLastCalledWith(
      '/v1/sys/organization/dataScope',
      {
        authorityId: 9,
        scope: 'self',
        departmentIds: [],
        expectedRevision: '7',
      },
      { headers: { Authorization: 'Bearer token' } },
    );
  });
  it('reads targeted configurations with query ids and normalizes nullable repeated fields', async () => {
    mocks.get.mockResolvedValueOnce({
      userId: 3,
      departmentId: 0,
      positionIds: null,
    });
    expect(await getMembership(3)).toEqual({
      userId: 3,
      departmentId: 0,
      positionIds: [],
    });
    expect(mocks.get).toHaveBeenLastCalledWith(
      '/v1/sys/organization/membership',
      { params: { userId: 3 } },
    );
    mocks.get.mockResolvedValueOnce({
      authorityId: 9,
      scope: 'self',
      departmentIds: null,
    });
    expect(await getRoleDataScope(9)).toEqual({
      authorityId: 9,
      scope: 'self',
      departmentIds: [],
    });
    expect(mocks.get).toHaveBeenLastCalledWith(
      '/v1/sys/organization/dataScope',
      { params: { authorityId: 9 } },
    );
  });
});
