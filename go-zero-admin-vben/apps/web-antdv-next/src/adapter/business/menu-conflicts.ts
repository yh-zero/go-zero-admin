interface MenuRoute {
  children?: MenuRoute[];
  name?: string | symbol;
  path: string;
}

/** Mixed mode merges routes by name; reject collisions to avoid replacing an authorized page. */
export function assertIndependentMenus(
  local: MenuRoute[],
  backend: MenuRoute[],
) {
  const names = new Set<string | symbol>();
  const paths = new Set<string>();

  function walk(routes: MenuRoute[], check: boolean, parent = '') {
    for (const route of routes) {
      const path = (
        route.path.startsWith('/') ? route.path : `${parent}/${route.path}`
      )
        .replace(/\/{2,}/g, '/')
        .replace(/\/$/, '')
        .toLowerCase();
      if (check) {
        if ((route.name && names.has(route.name)) || paths.has(path)) {
          throw new Error(
            `业务菜单与原有页面路由冲突：${String(route.name || path)}`,
          );
        }
      } else {
        if (route.name) names.add(route.name);
        paths.add(path);
      }
      walk(route.children || [], check, path);
    }
  }

  walk(local, false);
  walk(backend, true);
}
