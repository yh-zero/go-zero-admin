type HierarchyItem = { id: number; parent_id: number; sort?: number };
export type TreeNode<T> = T & { children?: TreeNode<T>[] };
export function buildTree<T extends HierarchyItem>(items: T[]): TreeNode<T>[] {
  const nodes = new Map<number, TreeNode<T>>();
  for (const item of items) nodes.set(item.id, { ...item, children: undefined });
  const roots: TreeNode<T>[] = [];
  for (const node of nodes.values()) {
    let cursor = nodes.get(node.parent_id);
    const parent = cursor;
    const seen = new Set([node.id]);
    let cycle = false;
    while (cursor) {
      if (seen.has(cursor.id)) { cycle = true; break; }
      seen.add(cursor.id);
      cursor = nodes.get(cursor.parent_id);
    }
    if (!parent || cycle) roots.push(node);
    else (parent.children ??= []).push(node);
  }
  const sort = (branch: TreeNode<T>[]) => {
    branch.sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0) || a.id - b.id);
    branch.forEach(node => { if (node.children) sort(node.children); });
  };
  sort(roots);
  return roots;
}
