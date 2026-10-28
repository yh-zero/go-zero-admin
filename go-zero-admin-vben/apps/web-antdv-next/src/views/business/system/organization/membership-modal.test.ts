/* eslint-disable vue/one-component-per-file -- Lightweight component stubs isolate the membership workflow. */
import type { App } from 'vue';

import type { Membership } from '#/api/business/system/organization';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import MembershipModal from './membership-modal.vue';

const mocks = vi.hoisted(() => ({
  getDepartments: vi.fn(),
  getPositions: vi.fn(),
  getMembership: vi.fn(),
  updateMembership: vi.fn(),
  logout: vi.fn(),
  values: { departmentId: 0, positionIds: [] as number[] },
  userInfo: { userId: '99' },
  access: { accessToken: 'old-token' },
}));
vi.mock('#/api/business/system/organization', () => mocks);
vi.mock('#/store', () => ({ useAuthStore: () => ({ logout: mocks.logout }) }));
vi.mock('@vben/stores', () => ({
  useUserStore: () => ({ userInfo: mocks.userInfo }),
  useAccessStore: () => mocks.access,
}));
vi.mock('../shared', () => ({
  confirmAction: (_title: string, action: () => Promise<void>) => action(),
}));
vi.mock('#/adapter/form', async () => {
  const { defineComponent, h } = await import('vue');
  const schema = {
    int: () => schema,
    min: () => schema,
    positive: () => schema,
  };
  return {
    z: { number: () => schema, array: () => schema },
    useVbenForm: () => [
      defineComponent({ setup: () => () => h('div', { 'data-form': true }) }),
      {
        reset: async () => {},
        updateSchema: () => {},
        validate: async () => ({ valid: true }),
        setValues: async (values: typeof mocks.values) => {
          mocks.values = { ...values, positionIds: [...values.positionIds] };
        },
        getValues: async () => mocks.values,
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
        h('div', [slots.default?.(), slots.action?.()]),
  });
  return {
    Alert: panel,
    Spin: panel,
    Modal: defineComponent({
      props: { okButtonProps: { type: Object, default: () => ({}) } },
      emits: ['ok'],
      setup:
        (props, { slots, emit }) =>
        () =>
          h('div', [
            slots.default?.(),
            h(
              'button',
              {
                'data-save': true,
                disabled: props.okButtonProps.disabled,
                onClick: () => emit('ok'),
              },
              '保存归属',
            ),
          ]),
    }),
    Button: defineComponent({
      setup:
        (_, { attrs, slots }) =>
        () =>
          h('button', attrs, slots.default?.()),
    }),
    message: { success: vi.fn(), info: vi.fn() },
  };
});
function deferred() {
  let resolve!: (value: Membership) => void;
  const promise = new Promise<Membership>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
describe('user organization membership workflow', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.values = { departmentId: 0, positionIds: [] };
    mocks.userInfo.userId = '99';
    mocks.access.accessToken = 'old-token';
    mocks.getDepartments.mockResolvedValue([]);
    mocks.getPositions.mockResolvedValue([]);
    mocks.updateMembership.mockResolvedValue(null);
    mocks.logout.mockResolvedValue(null);
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  function mount() {
    app = createApp(MembershipModal);
    return app.mount(element) as unknown as {
      show: (id: number, name: string) => Promise<void>;
    };
  }
  function saveButton() {
    return element.querySelector<HTMLButtonElement>('[data-save]')!;
  }
  async function flush() {
    for (let count = 0; count < 8; count++) await Promise.resolve();
    await nextTick();
  }
  it('ignores stale membership reads when opening a different user', async () => {
    const old = deferred();
    mocks.getMembership.mockImplementation((id: number) =>
      id === 11
        ? old.promise
        : Promise.resolve({ userId: id, departmentId: 3, positionIds: [4] }),
    );
    const view = mount();
    const first = view.show(11, '旧用户');
    await view.show(22, '新用户');
    old.resolve({ userId: 11, departmentId: 1, positionIds: [2] });
    await first;
    expect(mocks.values).toEqual({ departmentId: 3, positionIds: [4] });
    mocks.values = { departmentId: 0, positionIds: [] };
    await nextTick();
    saveButton().click();
    await flush();
    expect(mocks.updateMembership).toHaveBeenCalledExactlyOnceWith({
      userId: 22,
      departmentId: 0,
      positionIds: [],
    });
    expect(mocks.logout).not.toHaveBeenCalled();
  });
  it('disables writes after a failed read', async () => {
    mocks.getMembership.mockRejectedValue(new Error('permission denied'));
    await mount().show(22, '用户');
    await nextTick();
    expect(saveButton().disabled).toBe(true);
    saveButton().click();
    await flush();
    expect(mocks.updateMembership).not.toHaveBeenCalled();
  });
  it('does not revoke sessions for unchanged assignments', async () => {
    mocks.getMembership.mockResolvedValue({
      userId: 99,
      departmentId: 3,
      positionIds: [4, 5],
    });
    await mount().show(99, '本人');
    mocks.values.positionIds = [5, 4];
    await nextTick();
    saveButton().click();
    await flush();
    expect(mocks.updateMembership).not.toHaveBeenCalled();
    expect(mocks.logout).not.toHaveBeenCalled();
  });
  it('clears only the local login when the current user membership changes', async () => {
    mocks.getMembership.mockResolvedValue({
      userId: 99,
      departmentId: 3,
      positionIds: [4],
    });
    await mount().show(99, '本人');
    mocks.values = { departmentId: 0, positionIds: [] };
    await nextTick();
    saveButton().click();
    await flush();
    expect(mocks.updateMembership).toHaveBeenCalledExactlyOnceWith({
      userId: 99,
      departmentId: 0,
      positionIds: [],
    });
    expect(mocks.logout).toHaveBeenCalledExactlyOnceWith(false, false);
  });
  it('does not log out a newer login while the membership write was pending', async () => {
    mocks.getMembership.mockResolvedValue({
      userId: 99,
      departmentId: 3,
      positionIds: [4],
    });
    const pending = deferred();
    mocks.updateMembership.mockReturnValue(pending.promise);
    await mount().show(99, '本人');
    mocks.values = { departmentId: 0, positionIds: [] };
    await nextTick();
    saveButton().click();
    await flush();
    mocks.access.accessToken = 'new-token';
    pending.resolve({ userId: 99, departmentId: 0, positionIds: [] });
    await flush();
    expect(mocks.logout).not.toHaveBeenCalled();
  });
});
