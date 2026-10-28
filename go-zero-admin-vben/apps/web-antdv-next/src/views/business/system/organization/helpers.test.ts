import type { Department } from '#/api/business/system/organization';

import { describe, expect, it } from 'vitest';

import {
  departmentDescendants,
  departmentOptions,
  membershipDepartmentOptions,
  membershipPositionOptions,
  sameMembership,
} from './helpers';

function department(
  id: number,
  status: number,
  children: Department[] = [],
): Department {
  return {
    id,
    name: `D${id}`,
    code: `D${id}`,
    parentId: 0,
    status,
    sort: 0,
    children,
  };
}
describe('organization selection boundaries', () => {
  it('blocks a disabled ancestor and every descendant from new assignments', () => {
    expect(
      departmentOptions([
        department(1, 2, [department(2, 1)]),
        department(3, 1),
      ]),
    ).toEqual([
      { value: 1, label: 'D1（停用）', disabled: true },
      { value: 2, label: 'D1 / D2（停用）', disabled: true },
      { value: 3, label: 'D3', disabled: false },
    ]);
  });
  it('excludes the complete descendant chain when changing a parent', () => {
    const tree = [
      department(1, 1, [department(2, 1, [department(3, 1)])]),
      department(4, 1),
    ];
    expect([...departmentDescendants(tree, 2)]).toEqual([2, 3]);
    expect([...departmentDescendants(tree, 9)]).toEqual([]);
  });
  it('does not revoke sessions solely because positions have a different order', () => {
    const current = { userId: 3, departmentId: 1, positionIds: [1, 2] };
    expect(
      sameMembership(current, { ...current, positionIds: [2, 1, 1] }),
    ).toBe(true);
    expect(sameMembership(current, { ...current, positionIds: [] })).toBe(
      false,
    );
    expect(sameMembership(current, { ...current, departmentId: 0 })).toBe(
      false,
    );
  });
  it('lets the original disabled department be restored without enabling new disabled branches', () => {
    const options = membershipDepartmentOptions(
      [
        department(1, 2, [department(2, 2), department(3, 2)]),
        department(4, 1),
      ],
      2,
    );
    expect(options.map(({ value, disabled }) => ({ value, disabled }))).toEqual(
      [
        { value: 0, disabled: false },
        { value: 1, disabled: true },
        { value: 2, disabled: false },
        { value: 3, disabled: true },
        { value: 4, disabled: false },
      ],
    );
    expect(membershipDepartmentOptions([], 9)).toContainEqual({
      value: 9,
      label: '已失效部门 #9',
      disabled: true,
    });
  });
  it('only makes existing assigned disabled positions editable and resets that whitelist after removal', () => {
    const positions = [1, 2, 3, 4].map((id) => ({
      id,
      name: `P${id}`,
      code: `P${id}`,
      sort: 0,
      status: id === 4 ? 1 : 2,
    }));
    expect(membershipPositionOptions(positions, [1, 2, 9])).toEqual([
      { value: 1, label: 'P1（停用）', disabled: false },
      { value: 2, label: 'P2（停用）', disabled: false },
      { value: 3, label: 'P3（停用）', disabled: true },
      { value: 4, label: 'P4', disabled: false },
      { value: 9, label: '已失效岗位 #9', disabled: true },
    ]);
    expect(
      membershipPositionOptions(positions, [2]).find(({ value }) => value === 1)
        ?.disabled,
    ).toBe(true);
  });
});
