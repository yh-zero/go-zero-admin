/* eslint-disable vue/one-component-per-file -- Lightweight component stubs isolate the password workflow. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Account from './index.vue';

const mocks = vi.hoisted(() => ({
  validate: vi.fn(),
  getValues: vi.fn(),
  reset: vi.fn(),
  changeMyPassword: vi.fn(),
  getCurrentUser: vi.fn(),
  logout: vi.fn(),
  access: { accessToken: 'old-token' },
  mountDevices: vi.fn(),
}));
vi.mock('#/api/business/account', () => ({
  changeMyPassword: mocks.changeMyPassword,
  getCurrentUser: mocks.getCurrentUser,
}));
vi.mock('#/store', () => ({ useAuthStore: () => ({ logout: mocks.logout }) }));
vi.mock('@vben/stores', () => ({ useAccessStore: () => mocks.access }));
vi.mock('#/components/business/device-sessions.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      setup: () => {
        mocks.mountDevices();
        return () => h('div');
      },
    }),
  };
});
vi.mock('@vben/common-ui', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    Page: defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('div', slots.default?.()),
    }),
  };
});
vi.mock('#/adapter/form', async () => {
  const { defineComponent, h } = await import('vue');
  const schema = { refine: () => schema };
  return {
    z: { string: () => schema },
    useVbenForm: () => [
      defineComponent({ setup: () => () => h('div') }),
      {
        validate: mocks.validate,
        getValues: mocks.getValues,
        reset: mocks.reset,
      },
    ],
  };
});
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  const panel = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', [slots.default?.(), slots.extra?.()]),
  });
  return {
    Card: panel,
    Descriptions: panel,
    DescriptionsItem: panel,
    Spin: panel,
    Button: defineComponent({
      setup:
        (_, { attrs, slots }) =>
        () =>
          h('button', attrs, slots.default?.()),
    }),
    message: { success: vi.fn(), error: vi.fn() },
  };
});
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { resolve, promise };
}
describe('account password session safety', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.access.accessToken = 'old-token';
    mocks.validate.mockResolvedValue({ valid: true });
    mocks.getValues.mockResolvedValue({
      oldPassword: 'old',
      newPassword: 'Strong@123',
      confirmPassword: 'Strong@123',
    });
    mocks.getCurrentUser.mockResolvedValue({ userName: 'tester' });
    mocks.changeMyPassword.mockResolvedValue(null);
    mocks.reset.mockResolvedValue(null);
    mocks.logout.mockResolvedValue(null);
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  async function mount() {
    app = createApp(Account);
    app.mount(element);
    await nextTick();
  }
  function saveButton() {
    const found = [...element.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('修改密码并退出'),
    );
    if (!found) throw new Error('Password submit button should be available');
    return found;
  }
  async function flush() {
    for (let count = 0; count < 8; count++) await Promise.resolve();
    await nextTick();
  }
  it('shows account details, password controls and device management', async () => {
    await mount();
    await flush();
    expect(element.textContent).toContain('tester');
    expect(saveButton()).toBeDefined();
    expect(mocks.mountDevices).toHaveBeenCalledOnce();
    expect(mocks.getCurrentUser).toHaveBeenCalledOnce();
  });
  it('blocks repeat submissions while validation and the password write are pending', async () => {
    const validation = deferred<{ valid: boolean }>();
    const write = deferred<null>();
    mocks.validate.mockReturnValue(validation.promise);
    mocks.changeMyPassword.mockReturnValue(write.promise);
    await mount();
    saveButton().click();
    saveButton().click();
    expect(mocks.validate).toHaveBeenCalledTimes(1);
    expect(mocks.changeMyPassword).not.toHaveBeenCalled();
    validation.resolve({ valid: true });
    await flush();
    saveButton().click();
    expect(mocks.changeMyPassword).toHaveBeenCalledExactlyOnceWith({
      oldPassword: 'old',
      newPassword: 'Strong@123',
    });
    write.resolve(null);
    await flush();
    expect(mocks.logout).toHaveBeenCalledExactlyOnceWith(false, false);
  });
  it('does not clear a newer login when an older password response arrives', async () => {
    const write = deferred<null>();
    mocks.changeMyPassword.mockReturnValue(write.promise);
    await mount();
    saveButton().click();
    await flush();
    expect(mocks.changeMyPassword).toHaveBeenCalledTimes(1);
    mocks.access.accessToken = 'new-token';
    write.resolve(null);
    await flush();
    expect(mocks.logout).not.toHaveBeenCalled();
  });
});
