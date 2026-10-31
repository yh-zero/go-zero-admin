import type { MenuInfo } from '../../api/types';

export function getDescendantIds(resources: MenuInfo[], id: number): Set<number> {
  const children = new Map<number, number[]>();
  for (const resource of resources) {
    const siblings = children.get(resource.parent_id) ?? [];
    siblings.push(resource.id);
    children.set(resource.parent_id, siblings);
  }
  const result = new Set<number>();
  const pending = [id];
  for (let index = 0; index < pending.length; index++) {
    const current = pending[index];
    if (result.has(current)) continue;
    result.add(current);
    pending.push(...(children.get(current) ?? []));
  }
  return result;
}

export function filterNavigationMenus(resources: MenuInfo[], keyword: string, status?: number): MenuInfo[] {
  const navigation = resources.filter(resource => resource.type === 1 || resource.type === 2);
  const byId = new Map(navigation.map(resource => [resource.id, resource]));
  const selected = new Set<number>();
  const search = keyword.trim().toLocaleLowerCase();
  for (const resource of navigation) {
    if (status !== undefined && resource.status !== status) continue;
    const branch: MenuInfo[] = [];
    const seen = new Set<number>();
    let cursor: MenuInfo | undefined = resource;
    while (cursor && !seen.has(cursor.id)) {
      seen.add(cursor.id);
      branch.push(cursor);
      cursor = byId.get(cursor.parent_id);
    }
    if (!search || branch.some(item => item.title.toLocaleLowerCase().includes(search))) {
      branch.forEach(item => selected.add(item.id));
    }
  }
  return navigation.filter(resource => selected.has(resource.id));
}

function sortResources(resources: MenuInfo[]): MenuInfo[] {
  return resources.sort((left, right) => left.sort - right.sort || left.id - right.id);
}

export function getPagePermissions(resources: MenuInfo[], pageId: number): MenuInfo[] {
  return sortResources(resources.filter(resource => resource.type === 3 && resource.parent_id === pageId));
}

export function getUnassignedPermissions(resources: MenuInfo[]): MenuInfo[] {
  const pages = new Set(resources.filter(resource => resource.type === 2).map(resource => resource.id));
  return sortResources(resources.filter(resource => resource.type === 3 && !pages.has(resource.parent_id)));
}
