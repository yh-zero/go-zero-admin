import type { PermissionChange } from '#/api/business/system/permissions';

import { permissionChanges, policyKey } from './authorization';
export function historyDifference(change: PermissionChange) {
  const lines: string[] = [];
  for (const [field, label] of [
    ['menuIds', '菜单'],
    ['menuBtnIds', '按钮'],
  ] as const) {
    const delta = permissionChanges(
      change.before[field] ?? [],
      change.after[field] ?? [],
    );
    lines.push(
      ...delta.added.map((id) => `新增${label} #${id}`),
      ...delta.removed.map((id) => `撤销${label} #${id}`),
    );
  }
  const api = permissionChanges(
    (change.before.policies ?? []).map(policyKey),
    (change.after.policies ?? []).map(policyKey),
  );
  lines.push(
    ...api.added.map((rule) => `新增接口 ${rule}`),
    ...api.removed.map((rule) => `撤销接口 ${rule}`),
  );
  const before = change.before.dataScope;
  const after = change.after.dataScope;
  if (before?.scope !== after?.scope)
    lines.push(
      `数据范围：${before?.scope ?? 'self'} → ${after?.scope ?? 'self'}`,
    );
  const departments = permissionChanges(
    before?.departmentIds ?? [],
    after?.departmentIds ?? [],
  );
  lines.push(
    ...departments.added.map((id) => `新增范围部门 #${id}`),
    ...departments.removed.map((id) => `撤销范围部门 #${id}`),
  );
  return lines;
}
