import { onBeforeUnmount, type Ref, watch } from 'vue';

import { useAccessStore } from '@vben/stores';

import { createEditSession } from '#/adapter/business/permission-runtime';
export function useEditSession(
  open: Ref<boolean>,
  loaded: Ref<boolean>,
  saving: Ref<boolean>,
) {
  const access = useAccessStore();
  const session = createEditSession(() => access.accessToken);
  const invalidate = () => {
    session.invalidate();
    loaded.value = false;
    saving.value = false;
  };
  watch(
    open,
    (value) => {
      if (!value) invalidate();
    },
    { flush: 'sync' },
  );
  watch(
    () => access.accessToken,
    () => {
      invalidate();
      open.value = false;
    },
    { flush: 'sync' },
  );
  onBeforeUnmount(invalidate);
  return session;
}
