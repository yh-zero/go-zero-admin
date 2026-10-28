import type { PermissionChange } from '#/api/business/system/permissions';

import { describe, expect, it } from 'vitest';

import { historyDifference } from './history-difference';
describe('controlled permission history differences', () => {
  it('renders added and revoked menu/button/API/department grants and scope changes', () => {
    const change: PermissionChange = {
      id: 1,
      authorityId: 1,
      kind: 'dataScope',
      beforeRevision: '1',
      afterRevision: '2',
      actorId: 1,
      actorName: 'Admin',
      traceId: 'trace',
      createdAt: '2026-10-09',
      before: {
        menuIds: [1],
        menuBtnIds: [2],
        policies: [{ method: 'get', path: '/old' }],
        dataScope: { authorityId: 1, scope: 'custom', departmentIds: [4] },
      },
      after: {
        menuIds: [3],
        menuBtnIds: [],
        policies: [{ method: 'GET', path: '/new' }],
        dataScope: { authorityId: 1, scope: 'self', departmentIds: [] },
      },
    };
    expect(historyDifference(change)).toEqual([
      '新增菜单 #3',
      '撤销菜单 #1',
      '撤销按钮 #2',
      '新增接口 GET /new',
      '撤销接口 GET /old',
      '数据范围：custom → self',
      '撤销范围部门 #4',
    ]);
  });
});
