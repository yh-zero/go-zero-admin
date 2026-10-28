import type { Router } from 'vue-router';

import { commitAccessibleRoutes } from '@vben/access';
import { useAccessStore, useTabbarStore, useUserStore } from '@vben/stores';

import { adaptMenus } from '#/adapter/business/menu';
import { createPermissionRefresh } from '#/adapter/business/permission-runtime';
import { readSession } from '#/adapter/business/session';
import { getPermissionSnapshot } from '#/api/business/system/permissions';

import { generateAccess } from './access';
import {
  permissionNavigationTarget,
  synchronizePermissionTabs,
} from './permission-location';
import {
  refreshCurrentPermissions,
  registerPermissionRefresh,
} from './permission-refresh-events';
import { accessRoutes } from './routes';
export function installPermissionRefresh(router: Router) {
  const access = useAccessStore();
  const user = useUserStore();
  const tabs = useTabbarStore();
  const coordinator = createPermissionRefresh({
    token: () => access.accessToken,
    load: () => getPermissionSnapshot(access.accessToken ?? ''),
    async build(snapshot) {
      const session = readSession(access.accessToken);
      const adapted = adaptMenus(
        snapshot.menus,
        snapshot.defaultRouter,
        String(snapshot.authorityId),
      );
      const candidate = await generateAccess(
        { roles: [String(snapshot.authorityId)], router, routes: accessRoutes },
        adapted.routes,
      );
      return {
        ...candidate,
        homePath: adapted.homePath,
        user: { ...session.user, roles: [String(snapshot.authorityId)] },
        codes: [...new Set([...snapshot.codes, ...adapted.codes])],
      };
    },
    commit(candidate) {
      commitAccessibleRoutes(
        router,
        candidate.accessibleRoutes,
        access.accessRoutes,
      );
      access.setAccessCodes(candidate.codes);
      user.setUserInfo({ ...candidate.user, homePath: candidate.homePath });
      access.setAccessMenus(candidate.accessibleMenus);
      access.setAccessRoutes(candidate.accessibleRoutes);
      access.setIsAccessChecked(true);
      tabs.tabs = synchronizePermissionTabs(router, tabs.tabs);
      tabs.visitHistory.retain(tabs.tabs.map((tab) => tab.key ?? tab.fullPath));
      tabs.cachedRoutes.clear();
      tabs.cachedTabs.clear();
    },
  });
  let pending: Promise<void> | undefined;
  let latest: Promise<void> | undefined;
  const refresh = async (force = false) => {
    let request = force
      ? coordinator.refresh()
      : (pending ?? coordinator.refresh());
    pending = request;
    latest = request;
    try {
      for (;;) {
        try {
          await request;
        } catch (error) {
          if (!latest || latest === request) throw error;
        }
        // Navigation awaiting a superseded read must await the latest commit.
        if (!latest || latest === request) return;
        request = latest;
      }
    } finally {
      if (pending === request) pending = undefined;
    }
  };
  const reset = () => {
    coordinator.reset();
    pending = undefined;
    latest = undefined;
  };
  const repair = () => {
    if (!access.accessToken || !access.isAccessChecked) return;
    const current = router.currentRoute.value;
    const target = permissionNavigationTarget(
      router,
      current,
      user.userInfo?.homePath ?? '/_session/home',
    );
    if (target) void router.replace(target);
  };
  const refreshOnEvent = (force = false) => {
    if (access.accessToken)
      void refreshCurrentPermissions(force).catch(() => {
        /* Request client retains auth/error handling. */
      });
  };
  registerPermissionRefresh({ refresh, reset, repair });
  window.addEventListener('focus', () => refreshOnEvent());
  window.addEventListener('storage', (event) => {
    if (event.key === 'business-permission-change') refreshOnEvent(true);
  });
  return refresh;
}
