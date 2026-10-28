import type { PageQuery, PageResult, User } from './types';

import { sessionRequestOptions } from '#/adapter/business/request-session';
import { requestClient } from '#/api/request';
export interface CreateUser {
  userName: string;
  passWord: string;
  nickName: string;
  headerImg?: string;
  phone?: string;
  email?: string;
  enable: number;
  authorityId: number;
  authorityIds: number[];
}
export type UpdateUser = Partial<Omit<CreateUser, 'passWord' | 'userName'>> & {
  ID: number;
};
export const getUsers = (params: PageQuery, token?: string) =>
  requestClient.get<PageResult<User>>('/v1/sys/getUserList', {
    ...sessionRequestOptions(token),
    params,
  });
export const createUser = (data: CreateUser, token?: string) =>
  token === undefined
    ? requestClient.post('/v1/sys/register', data)
    : requestClient.post(
        '/v1/sys/register',
        data,
        sessionRequestOptions(token),
      );
export const updateUser = (data: UpdateUser, token?: string) =>
  token === undefined
    ? requestClient.put('/v1/sys/updateUserInfo', data)
    : requestClient.put(
        '/v1/sys/updateUserInfo',
        data,
        sessionRequestOptions(token),
      );
export const deleteUser = (userId: number, token?: string) =>
  requestClient.delete('/v1/sys/deleteUser', {
    ...sessionRequestOptions(token),
    data: { userId },
  });
export const resetUserPassword = (userId: number, token?: string) =>
  token === undefined
    ? requestClient.put('/v1/sys/resetUserPassword', { userId })
    : requestClient.put(
        '/v1/sys/resetUserPassword',
        { userId },
        sessionRequestOptions(token),
      );

export interface UserResourcePreview {
  userId: number;
  username: string;
  files: number;
  aiConversations: number;
  aiMessages: number;
  aiRuns: number;
  aiAvailable: boolean;
  aiReason: string;
  transferMode: string;
}
export interface TransferUserResources {
  userId: number;
  targetUserId: number;
  includeFiles: boolean;
  includeAI: boolean;
}
export interface UserResourceTransferResult {
  files: number;
  aiConversations: number;
  aiMessages: number;
  aiRuns: number;
  transactionMode: string;
}
export const getUserResourcePreview = (userId: number, token?: string) =>
  requestClient.get<UserResourcePreview>('/v1/sys/user/resources', {
    ...sessionRequestOptions(token),
    params: { userId },
  });
export const transferUserResources = (
  data: TransferUserResources,
  token?: string,
) =>
  token === undefined
    ? requestClient.post<UserResourceTransferResult>(
        '/v1/sys/user/resources/transfer',
        data,
      )
    : requestClient.post<UserResourceTransferResult>(
        '/v1/sys/user/resources/transfer',
        data,
        sessionRequestOptions(token),
      );
