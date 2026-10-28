import type { Recordable } from '@vben/types';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { notification } from 'antdv-next';
import { defineStore } from 'pinia';

import {
  clearSessionCache,
  safeRedirect,
  saveSession,
  SESSION_HOME,
  StaleSessionResponseError,
} from '#/adapter/business/session';
import { getUserInfoApi, loginApi, logoutApi } from '#/api';
import { resetRoutes } from '#/router';
import { resetPermissionRefresh } from '#/router/permission-refresh-events';
export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const router = useRouter();
  const loginLoading = ref(false);
  function clearSession() {
    resetPermissionRefresh();
    clearSessionCache();
    resetRoutes();
    resetAllStores();
  }
  async function authLogin(
    params: Recordable<any>,
    onSuccess?: () => Promise<void> | void,
  ) {
    const destination = safeRedirect(
      router.currentRoute.value.query.redirect,
      SESSION_HOME,
    );
    clearSession();
    loginLoading.value = true;
    try {
      const result = await loginApi(params);
      const userInfo = saveSession(result);
      accessStore.setAccessToken(result.accessToken);
      userStore.setUserInfo(userInfo);
      await (onSuccess ? onSuccess() : router.replace(destination));
      if (accessStore.accessToken)
        notification.success({
          title: '登录成功',
          description: userInfo.realName,
          duration: 3,
        });
      return { userInfo };
    } finally {
      loginLoading.value = false;
    }
  }
  async function logout(redirect = true, revoke = true) {
    const destination = safeRedirect(
      router.currentRoute.value.fullPath,
      SESSION_HOME,
    );
    const token = accessStore.accessToken;
    try {
      if (revoke && token) await logoutApi(token);
    } catch {
      if (accessStore.accessToken === token)
        notification.warning({
          title: '已退出当前页面',
          description: '服务器会话撤销未确认，网络恢复后可重新登录并再次退出。',
        });
    } finally {
      if (accessStore.accessToken === token) clearSession();
    }
    // Never clear or navigate away from a newer login while logout was pending.
    if (accessStore.accessToken) return;
    await router.replace({
      path: LOGIN_PATH,
      query: redirect ? { redirect: encodeURIComponent(destination) } : {},
    });
  }
  async function fetchUserInfo() {
    const token = accessStore.accessToken;
    const userInfo = await getUserInfoApi();
    if (accessStore.accessToken !== token)
      throw new StaleSessionResponseError();
    userStore.setUserInfo(userInfo);
    return userInfo;
  }
  function $reset() {
    loginLoading.value = false;
  }
  return {
    $reset,
    authLogin,
    clearSession,
    fetchUserInfo,
    loginLoading,
    logout,
  };
});
