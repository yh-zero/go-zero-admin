import type {
  Department,
  Membership,
  Position,
} from '#/api/business/system/organization';

export interface DepartmentOption {
  disabled: boolean;
  label: string;
  value: number;
}

// A disabled ancestor also makes its descendants unavailable for new assignments.
export function departmentOptions(
  nodes: Department[],
  prefix = '',
  ancestorDisabled = false,
): DepartmentOption[] {
  return nodes.flatMap((node) => {
    const path = prefix ? `${prefix} / ${node.name}` : node.name;
    const disabled = ancestorDisabled || node.status !== 1;
    return [
      {
        value: node.id,
        label: `${path}${disabled ? '（停用）' : ''}`,
        disabled,
      },
      ...departmentOptions(node.children ?? [], path, disabled),
    ];
  });
}

// The server allows an existing assignment to be retained while it is disabled.
// Keep those options removable and selectable until this edit is persisted.
export function membershipDepartmentOptions(
  nodes: Department[],
  assignedId: number,
): DepartmentOption[] {
  const options = [
    { value: 0, label: '未分配部门', disabled: false },
    ...departmentOptions(nodes).map((item) => ({
      ...item,
      disabled: item.disabled && item.value !== assignedId,
    })),
  ];
  if (!options.some((item) => item.value === assignedId)) {
    options.push({
      value: assignedId,
      label: `已失效部门 #${assignedId}`,
      disabled: true,
    });
  }
  return options;
}

export function membershipPositionOptions(
  positions: Position[],
  assignedIds: number[],
): DepartmentOption[] {
  const assigned = new Set(assignedIds);
  const options = positions.map((item) => ({
    value: item.id,
    label: `${item.name}${item.status !== 1 ? '（停用）' : ''}`,
    disabled: item.status !== 1 && !assigned.has(item.id),
  }));
  for (const id of assigned) {
    if (!options.some((item) => item.value === id)) {
      options.push({ value: id, label: `已失效岗位 #${id}`, disabled: true });
    }
  }
  return options;
}

export function departmentDescendants(
  nodes: Department[],
  id: number,
): Set<number> {
  for (const node of nodes) {
    if (node.id === id)
      return new Set([
        id,
        ...departmentOptions(node.children ?? []).map((item) => item.value),
      ]);
    const children = departmentDescendants(node.children ?? [], id);
    if (children.size) return children;
  }
  return new Set();
}

export function sameIds(first: number[], second: number[]) {
  const left = [...new Set(first)].sort((a, b) => a - b);
  const right = [...new Set(second)].sort((a, b) => a - b);
  return (
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  );
}
export function sameMembership(first: Membership, second: Membership) {
  return (
    first.userId === second.userId &&
    first.departmentId === second.departmentId &&
    sameIds(first.positionIds, second.positionIds)
  );
}
