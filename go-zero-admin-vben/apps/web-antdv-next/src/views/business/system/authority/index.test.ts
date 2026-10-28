/* eslint-disable vue/one-component-per-file -- Mount the real page with lightweight UI controls. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Roles from './index.vue';
const mocks = vi.hoisted(() => ({
  access: { accessToken: 'token' },
  formState: { value: '' },
  all: vi.fn(),
  assigned: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  values: vi.fn(),
  setValues: vi.fn(),
  reset: vi.fn(),
  validate: vi.fn(),
  confirm: vi.fn(),
  query: vi.fn(),
  reload: vi.fn(),
  refresh: vi.fn(),
  currentRole: vi.fn(),
  sessionHome: vi.fn(),
  rows: [
    {
      authorityId: 11,
      authorityName: 'One',
      parentId: 0,
      defaultRouter: 'index',
    },
    {
      authorityId: 22,
      authorityName: 'Two',
      parentId: 0,
      defaultRouter: 'index',
    },
  ],
}));
vi.mock('@vben/stores', async () => {
  const { reactive } = await import('vue');
  mocks.access = reactive(mocks.access);
  return { useAccessStore: () => mocks.access };
});
vi.mock('#/api/business/system/authority', () => ({
  getAuthorities: vi.fn(),
  getAllAuthorities: mocks.all,
  createAuthority: mocks.create,
  updateAuthority: mocks.update,
  deleteAuthority: mocks.remove,
}));
vi.mock('#/api/business/system/menu', () => ({
  getAuthorityMenus: mocks.assigned,
}));
vi.mock('#/adapter/business/session', () => ({
  updateSessionHome: mocks.sessionHome,
}));
vi.mock('../shared', () => ({
  usePermission: () => () => true,
  flattenTree: (rows: unknown[]) => rows,
  isCurrentRole: mocks.currentRole,
  refreshRoleAccess: mocks.refresh,
  confirmAction: mocks.confirm,
}));
vi.mock('./api-permissions.vue', () => ({ default: { render: () => null } }));
vi.mock('./button-permissions.vue', () => ({
  default: { render: () => null },
}));
vi.mock('./menu-permissions.vue', () => ({ default: { render: () => null } }));
vi.mock('./permission-history.vue', () => ({
  default: { render: () => null },
}));
vi.mock('../organization/data-scope-drawer.vue', () => ({
  default: { render: () => null },
}));
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
  const { defineComponent, h, reactive } = await import('vue');
  mocks.formState = reactive(mocks.formState);
  const schema = { int: () => schema, positive: () => schema };
  return {
    z: { number: () => schema },
    useVbenForm: () => [
      defineComponent({
        setup: () => () =>
          h('output', { 'data-form': true }, mocks.formState.value),
      }),
      {
        reset: mocks.reset,
        updateSchema: vi.fn(),
        setValues: mocks.setValues,
        validate: mocks.validate,
        getValues: mocks.values,
      },
    ],
  };
});
vi.mock('#/adapter/vxe-table', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    useVbenVxeGrid: () => [
      defineComponent({
        setup:
          (_, { slots }) =>
          () =>
            h('div', [
              slots['toolbar-tools']?.(),
              ...mocks.rows.map((row) =>
                h(
                  'section',
                  { 'data-role': row.authorityId },
                  slots.actions?.({ row }),
                ),
              ),
            ]),
      }),
      { query: mocks.query, reload: mocks.reload },
    ],
  };
});
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    Space: defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('div', slots.default?.()),
    }),
    Button: defineComponent({
      setup:
        (_, { attrs, slots }) =>
        () =>
          h('button', attrs, slots.default?.()),
    }),
    Modal: defineComponent({
      props: ['open', 'confirmLoading', 'okButtonProps', 'cancelButtonProps'],
      emits: ['ok', 'update:open'],
      setup:
        (props, { slots, emit }) =>
        () =>
          props.open
            ? h(
                'div',
                {
                  'data-editor': true,
                  'data-saving': String(props.confirmLoading),
                },
                [
                  slots.default?.(),
                  h(
                    'button',
                    {
                      'data-save': true,
                      disabled:
                        props.okButtonProps?.disabled || props.confirmLoading,
                      onClick: () => emit('ok'),
                    },
                    'Save',
                  ),
                  h(
                    'button',
                    {
                      'data-cancel': true,
                      disabled: props.cancelButtonProps?.disabled,
                      onClick: () => emit('update:open', false),
                    },
                    'Cancel',
                  ),
                ],
              )
            : null,
    }),
    message: { success: vi.fn() },
  };
});
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
async function flush() {
  for (let i = 0; i < 16; i++) await Promise.resolve();
  await nextTick();
}

describe('role page form session isolation', () => {
  let app: App;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.access.accessToken = 'token';
    mocks.formState.value = '';
    mocks.all.mockResolvedValue(mocks.rows);
    mocks.assigned.mockResolvedValue({ list: [] });
    mocks.currentRole.mockReturnValue(false);
    mocks.reset.mockResolvedValue(undefined);
    mocks.validate.mockResolvedValue({ valid: true });
    mocks.setValues.mockImplementation(async (value) => {
      mocks.formState.value = JSON.stringify(value);
    });
    mocks.values.mockImplementation(async () =>
      JSON.parse(mocks.formState.value),
    );
    element = document.createElement('div');
    document.body.append(element);
    app = createApp(Roles);
    app.mount(element);
  });
  afterEach(() => {
    app.unmount();
    element.remove();
  });
  function rowAction(id: number, text = '编辑') {
    const row = element.querySelector(`[data-role="${id}"]`)!;
    [...row.querySelectorAll('button')]
      .find((button) => button.textContent?.trim() === text)!
      .click();
  }
  function save() {
    element.querySelector<HTMLButtonElement>('[data-save]')!.click();
  }
  function cancel() {
    element.querySelector<HTMLButtonElement>('[data-cancel]')!.click();
  }
  function form() {
    return JSON.parse(element.querySelector('[data-form]')!.textContent!);
  }
  function saving() {
    return element.querySelector('[data-editor]')?.getAttribute('data-saving');
  }

  it('refreshes the role grid when saving the current actor role without a full page reload', async () => {
    mocks.currentRole.mockReturnValue(true);
    rowAction(11);
    await flush();
    mocks.values.mockResolvedValue({
      ...form(),
      authorityName: 'Changed role name',
    });
    save();
    await flush();
    expect(mocks.update).toHaveBeenCalledTimes(1);
    expect(mocks.refresh).toHaveBeenCalledWith(11);
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });

  it('keeps the newer role visible when the previous detail read arrives late', async () => {
    const old = deferred<unknown>();
    mocks.assigned.mockReturnValueOnce(old.promise);
    rowAction(11);
    rowAction(22);
    await flush();
    expect(form().authorityId).toBe(22);
    old.resolve({ list: [] });
    await flush();
    expect(form().authorityId).toBe(22);
    expect(
      element.querySelector<HTMLButtonElement>('[data-save]')!.disabled,
    ).toBe(false);
  });
  it('does not re-enable a new loading form after cancelling a previous read', async () => {
    const old = deferred<unknown>();
    const current = deferred<unknown>();
    mocks.assigned
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(current.promise);
    rowAction(11);
    await flush();
    cancel();
    await flush();
    rowAction(22);
    await flush();
    old.resolve({ list: [] });
    await flush();
    expect(
      element.querySelector<HTMLButtonElement>('[data-save]')!.disabled,
    ).toBe(true);
    expect(saving()).toBe('false');
    current.resolve({ list: [] });
    await flush();
    expect(form().authorityId).toBe(22);
  });
  it('binds create and edit submissions to their loaded object and captured login', async () => {
    [...element.querySelectorAll('button')]
      .find((button) => button.textContent?.trim() === '新增角色')!
      .click();
    await flush();
    mocks.values.mockResolvedValue({
      authorityId: 33,
      authorityName: 'New',
      parentId: 0,
      defaultRouter: 'index',
    });
    save();
    await flush();
    expect(mocks.create).toHaveBeenCalledWith(
      {
        authorityId: 33,
        authorityName: 'New',
        parentId: 0,
        defaultRouter: 'index',
      },
      'token',
    );
    rowAction(22);
    await flush();
    mocks.values.mockResolvedValue({
      authorityId: 22,
      authorityName: 'Changed',
      parentId: 11,
      defaultRouter: '',
    });
    save();
    await flush();
    expect(mocks.update).toHaveBeenCalledWith(
      {
        authorityId: 22,
        authorityName: 'Changed',
        parentId: 11,
        defaultRouter: '',
      },
      'token',
    );
  });
  it('does not submit if the account changes in the validation microtask', async () => {
    const validation = deferred<{ valid: boolean }>();
    mocks.validate.mockReturnValue(validation.promise);
    rowAction(11);
    await flush();
    save();
    await Promise.resolve();
    mocks.access.accessToken = 'new-token';
    validation.resolve({ valid: true });
    await flush();
    expect(mocks.update).not.toHaveBeenCalled();
    expect(mocks.create).not.toHaveBeenCalled();
    expect(element.querySelector('[data-editor]')).toBeNull();
  });
  it('does not execute a deletion confirmation using a different login', async () => {
    rowAction(11, '删除');
    const action = mocks.confirm.mock.calls[0]![1];
    await Promise.resolve();
    mocks.access.accessToken = 'new-token';
    await action();
    expect(mocks.remove).not.toHaveBeenCalled();
    expect(mocks.reload).not.toHaveBeenCalled();
  });
  it('does not close or reload a newly opened role after an old write completes', async () => {
    const write = deferred<null>();
    mocks.update.mockReturnValueOnce(write.promise);
    rowAction(11);
    await flush();
    save();
    await flush();
    expect(saving()).toBe('true');
    rowAction(22);
    await flush();
    expect(form().authorityId).toBe(22);
    write.resolve(null);
    await flush();
    expect(form().authorityId).toBe(22);
    expect(saving()).toBe('false');
    expect(mocks.query).not.toHaveBeenCalled();
    expect(mocks.reload).not.toHaveBeenCalled();
    expect(mocks.refresh).not.toHaveBeenCalled();
  });
  it('does not clear the saving state of a new role write when the prior write finishes', async () => {
    const first = deferred<null>();
    const second = deferred<null>();
    mocks.update
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    rowAction(11);
    await flush();
    save();
    await flush();
    rowAction(22);
    await flush();
    save();
    await flush();
    first.resolve(null);
    await flush();
    expect(saving()).toBe('true');
    expect(form().authorityId).toBe(22);
    second.resolve(null);
    await flush();
    expect(element.querySelector('[data-editor]')).toBeNull();
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });
});
