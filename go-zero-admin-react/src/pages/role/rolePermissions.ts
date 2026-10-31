import type { MenuInfo } from '../../api/types';
import { buildTree } from '../../utils/tree';

export type RolePermissionRow = Omit<MenuInfo, 'children'> & {
  operations: MenuInfo[];
  children?: RolePermissionRow[];
};

export interface RolePermissionGroups {
  rows: RolePermissionRow[];
  otherOperations: MenuInfo[];
  otherResources: MenuInfo[];
  resourceIds: Set<number>;
}

export function groupRolePermissions(menus: MenuInfo[]): RolePermissionGroups {
  const resources = new Map(menus.map(menu => [menu.id, menu]));
  const entries = new Map<number, RolePermissionRow>();
  const otherOperations: MenuInfo[] = [];
  const otherResources: MenuInfo[] = [];
  for (const menu of resources.values()) {
    if (menu.type === 1 || menu.type === 2) entries.set(menu.id, { ...menu, children: undefined, operations: [] });
  }
  for (const menu of resources.values()) {
    if (menu.type === 1 || menu.type === 2) continue;
    if (menu.type !== 3) { otherResources.push(menu); continue; }
    let parent = resources.get(menu.parent_id);
    const seen = new Set([menu.id]);
    while (parent && parent.type !== 2 && !seen.has(parent.id)) {
      seen.add(parent.id);
      parent = resources.get(parent.parent_id);
    }
    const page = parent?.type === 2 && !seen.has(parent.id) ? entries.get(parent.id) : undefined;
    if (page) page.operations.push(menu);
    else otherOperations.push(menu);
  }
  const byOrder = (a: MenuInfo, b: MenuInfo) => a.sort - b.sort || a.id - b.id;
  for (const entry of entries.values()) entry.operations.sort(byOrder);
  return {
    rows: buildTree([...entries.values()]),
    otherOperations: otherOperations.sort(byOrder),
    otherResources: otherResources.sort(byOrder),
    resourceIds: new Set(resources.keys()),
  };
}

export function togglePermission(selectedIds: number[], id: number, checked: boolean): number[] {
  if (!checked) return selectedIds.filter(selectedId => selectedId !== id);
  return selectedIds.includes(id) ? selectedIds : [...selectedIds, id];
}
