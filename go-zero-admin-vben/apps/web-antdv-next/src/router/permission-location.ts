import type {
  RouteLocationNormalized,
  RouteLocationRaw,
  Router,
} from 'vue-router';
export function permissionNavigationTarget(
  router: Router,
  location: RouteLocationNormalized,
  home: string,
): RouteLocationRaw | undefined {
  const resolved = router.resolve(location.fullPath);
  const live =
    resolved.matched.length > 0 && resolved.name !== 'FallbackNotFound';
  if (live && resolved.name === location.name) {
    const sameRecords =
      resolved.matched.length === location.matched.length &&
      resolved.matched.every(
        (record, index) => record === location.matched[index],
      );
    if (sameRecords) return;
    return {
      path: resolved.path,
      query: location.query,
      hash: location.hash,
      replace: true,
    };
  }
  if (
    location.name &&
    location.name !== 'FallbackNotFound' &&
    router.hasRoute(location.name)
  ) {
    try {
      const named = router.resolve({
        name: location.name,
        params: location.params,
        query: location.query,
        hash: location.hash,
      });
      return {
        path: named.path,
        query: named.query,
        hash: named.hash,
        replace: true,
      };
    } catch {
      /* A changed dynamic parameter contract cannot be reconstructed. */
    }
  }
  if (live)
    return {
      path: resolved.path,
      query: location.query,
      hash: location.hash,
      replace: true,
    };
  return { path: home, replace: true };
}
export function synchronizePermissionTabs<T extends RouteLocationNormalized>(
  router: Router,
  tabs: T[],
): T[] {
  return tabs.flatMap((tab) => {
    const resolved = router.resolve(tab.fullPath);
    if (
      !tab.name ||
      resolved.name !== tab.name ||
      !resolved.matched.length ||
      resolved.name === 'FallbackNotFound' ||
      resolved.meta.hideInTab
    )
      return [];
    return [
      {
        ...tab,
        matched: resolved.matched,
        meta: {
          ...resolved.meta,
          ...(tab.meta.newTabTitle === undefined
            ? {}
            : { newTabTitle: tab.meta.newTabTitle }),
        },
      },
    ];
  });
}
