import { useAccessStore } from '@vben/stores';

import {
  refreshSessionUser,
  StaleSessionResponseError,
} from '#/adapter/business/session';
import { getCurrentUser } from '#/api/business/account';

export async function getUserInfoApi() {
  const access = useAccessStore();
  const token = access.accessToken;
  try {
    const user = await getCurrentUser();
    if (!token || access.accessToken !== token)
      throw new StaleSessionResponseError();
    return refreshSessionUser(user, token);
  } catch (error) {
    if (access.accessToken !== token) throw new StaleSessionResponseError();
    throw error;
  }
}
