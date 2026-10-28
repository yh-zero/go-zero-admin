import type { Router } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { startProgress, stopProgress } from '@vben/utils';

import {
  isAuthenticationError,
  MissingSessionError,
  readSession,
  safeRedirect,
  SESSION_HOME,
  StaleSessionResponseError,
} from '#/adapter/business/session';
import { coreRouteNames } from '#/router/routes';
import { useAuthStore } from '#/store';

import { permissionNavigationTarget } from './permission-location';
import { installPermissionRefresh } from './permission-refresh';

/**
 * 通用守卫配置
 * @param router
 */
function setupCommonGuard(router: Router) {
  // 记录已经加载的页面
  const loadedPaths = new Set<string>();

  router.beforeEach((to) => {
    to.meta.loaded = loadedPaths.has(to.path);

    // 页面加载进度条
    if (!to.meta.loaded && preferences.transition.progress) {
      startProgress();
    }
    return true;
  });

  router.afterEach((to) => {
    // 记录页面是否加载,如果已经加载，后续的页面切换动画等效果不在重复执行

    loadedPaths.add(to.path);

    // 关闭页面加载进度条
    if (preferences.transition.progress) {
      stopProgress();
    }
  });
}

/**
 * 权限访问守卫配置
 * @param router
 */
function setupAccessGuard(router: Router) {
  let refresh: (() => Promise<void>) | undefined;
  router.beforeEach(async (to) => {
    const accessStore = useAccessStore();
    const userStore = useUserStore();
    const authStore = useAuthStore();
    const login = () => ({
      path: LOGIN_PATH,
      query: {
        redirect: encodeURIComponent(safeRedirect(to.fullPath, SESSION_HOME)),
      },
      replace: true,
    });
    if (accessStore.accessToken) {
      try {
        readSession(accessStore.accessToken);
      } catch {
        authStore.clearSession();
      }
    }
    if (coreRouteNames.includes(to.name as string)) {
      if (to.path === LOGIN_PATH && accessStore.accessToken)
        return safeRedirect(to.query.redirect, SESSION_HOME);
      if (to.name === 'AccessError' && !accessStore.accessToken) return login();
      return true;
    }
    if (!accessStore.accessToken) return login();

    const requestToken = accessStore.accessToken;
    try {
      const alreadyChecked = accessStore.isAccessChecked;
      refresh ??= installPermissionRefresh(router);
      await refresh();
      if (accessStore.accessToken !== requestToken) return false;
      if (alreadyChecked) {
        return (
          permissionNavigationTarget(
            router,
            to,
            userStore.userInfo?.homePath || SESSION_HOME,
          ) ?? true
        );
      }
      return {
        path:
          to.path === SESSION_HOME
            ? userStore.userInfo?.homePath || SESSION_HOME
            : to.path,
        query: to.query,
        hash: to.hash,
        replace: true,
      };
    } catch (error) {
      if (
        error instanceof StaleSessionResponseError ||
        (accessStore.accessToken && accessStore.accessToken !== requestToken)
      )
        return false;
      if (
        error instanceof MissingSessionError ||
        isAuthenticationError(error) ||
        !accessStore.accessToken
      ) {
        authStore.clearSession();
        return login();
      }
      return { name: 'AccessError', replace: true };
    }
  });
}

/**
 * 项目守卫配置
 * @param router
 */
function createRouterGuard(router: Router) {
  /** 通用 */
  setupCommonGuard(router);
  /** 权限访问 */
  setupAccessGuard(router);
}

export { createRouterGuard };
