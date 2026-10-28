import { onBeforeUnmount, watch } from 'vue';

import { useAccessStore } from '@vben/stores';

export const userStatuses = [
  { label: '待审核', value: 'pending' },
  { label: '已开通', value: 'active' },
  { label: '已拒绝', value: 'rejected' },
  { label: '已禁用', value: 'disabled' },
];
export const grantStatuses: Record<string, string> = {
  scheduled: '待生效',
  active: '有效',
  expired: '已到期',
  revoked: '已撤销',
};

// A form and every async continuation belong to the login that created them.
export function useRequestScope(reset: () => void) {
  const access = useAccessStore();
  let version = 0;
  let alive = true;
  watch(
    () => access.accessToken,
    () => {
      version++;
      reset();
    },
    { flush: 'sync' },
  );
  onBeforeUnmount(() => {
    alive = false;
    version++;
  });
  return {
    capture: () => ({ token: access.accessToken ?? '', version }),
    current: (scope: { token: string; version: number }) =>
      alive && scope.version === version && scope.token === access.accessToken,
  };
}
