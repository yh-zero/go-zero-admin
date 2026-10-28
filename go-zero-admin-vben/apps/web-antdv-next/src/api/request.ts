import type { RequestClientOptions } from '@vben/request';

import { useAppConfig } from '@vben/hooks';
import { preferences } from '@vben/preferences';
import {
  defaultResponseInterceptor,
  errorMessageResponseInterceptor,
  RequestClient,
} from '@vben/request';
import { useAccessStore } from '@vben/stores';

import { message } from 'antdv-next';

import {
  isAuthenticationError,
  isCurrentSessionRequest,
  StaleSessionResponseError,
} from '#/adapter/business/session';
import { useAuthStore } from '#/store';
const { apiURL } = useAppConfig(import.meta.env, import.meta.env.PROD);
function createRequestClient(baseURL: string, options?: RequestClientOptions) {
  const client = new RequestClient({ ...options, baseURL });
  client.addRequestInterceptor({
    fulfilled: async (config) => {
      const token = useAccessStore().accessToken;
      const authorization = token ? 'Bearer ' + token : null;
      if (
        config.headers.Authorization !== undefined &&
        config.headers.Authorization !== authorization
      )
        throw new StaleSessionResponseError();
      config.headers.Authorization = authorization;
      config.headers['Accept-Language'] = preferences.app.locale;
      return config;
    },
  });
  client.addResponseInterceptor(
    defaultResponseInterceptor({
      codeField: 'code',
      dataField: 'result',
      successCode: 200,
    }),
  );
  client.addResponseInterceptor({
    rejected: async (error) => {
      if (
        isAuthenticationError(error) &&
        isCurrentSessionRequest(error, useAccessStore().accessToken)
      ) {
        message.error('登录已失效，请重新登录');
        await useAuthStore().logout(true, false);
      }
      throw error;
    },
  });
  client.addResponseInterceptor(
    errorMessageResponseInterceptor((msg, error) => {
      if (
        !(error instanceof StaleSessionResponseError) &&
        !isAuthenticationError(error)
      )
        message.error(error?.response?.data?.message || msg);
    }),
  );
  return client;
}
export const requestClient = createRequestClient(apiURL, {
  responseReturn: 'data',
});
export const baseRequestClient = new RequestClient({ baseURL: apiURL });
