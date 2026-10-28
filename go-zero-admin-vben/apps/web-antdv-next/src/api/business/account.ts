import type { BackendUser } from './session';

import { requestClient } from '#/api/request';

export const getCurrentUser = () =>
  requestClient.get<BackendUser>('/v1/sys/me');
export const changeMyPassword = (data: {
  oldPassword: string;
  newPassword: string;
}) => requestClient.put('/v1/sys/changePassword', data);
