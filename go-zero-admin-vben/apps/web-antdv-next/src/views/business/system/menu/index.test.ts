/* eslint-disable vue/one-component-per-file -- Mount the real page with lightweight UI controls. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Menus from './index.vue';
const mocks = vi.hoisted(() => ({
  access: { accessToken: 'token' },
  formState: { value: '' },
  detail: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  preview: vi.fn(),
  previewPost: vi.fn(),
  values: vi.fn(),
  setValues: vi.fn(),
  reset: vi.fn(),
  validate: vi.fn(),
  confirm: vi.fn(),
  query: vi.fn(),
  refresh: vi.fn(),
  rows: [
    {
      ID: 1,
      parentId: 0,
      name: 'first',
      path: 'first',
      component: 'business/system/menu/index',
      sort: 1,
      hidden: false,
      meta: { title: 'First', icon: '', closeTab: false },
      parameters: [{ key: 'keep', value: 'one' }],
      menuBtn: [],
    },
    {
      ID: 2,
      parentId: 0,
      name: 'second',
      path: 'second',
      component: 'business/system/menu/index',
      sort: 2,
      hidden: false,
      meta: { title: 'Second', icon: '', closeTab: false },
      parameters: [],
      menuBtn: [],
    },
  ],
}));
vi.mock('@vben/stores', async () => {
  const { reactive } = await import('vue');
  mocks.access = reactive(mocks.access);
  return { useAccessStore: () => mocks.access };
});
vi.mock('#/adapter/business/pages', () => ({
  businessPages: [{ label: 'Menus', value: 'business/system/menu/index' }],
}));
vi.mock('#/api/business/system/menu', () => ({
  getMenu: mocks.detail,
  getMenus: vi.fn(),
  createMenu: mocks.create,
  updateMenu: mocks.update,
  deleteMenu: mocks.remove,
  previewMenuMove: mocks.preview,
}));
vi.mock('#/api/request', () => ({
  requestClient: { post: mocks.previewPost },
}));
vi.mock('../shared', () => ({
  usePermission: () => () => true,
  flattenTree: (rows: unknown[]) => rows,
  refreshAccess: mocks.refresh,
  confirmAction: mocks.confirm,
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
  const schema = { regex: () => schema, min: () => schema };
  return {
    z: { string: () => schema },
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
                h('section', { 'data-menu': row.ID }, slots.actions?.({ row })),
              ),
            ]),
      }),
      { query: mocks.query },
    ],
  };
});
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  const panel = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.()),
  });
  return {
    Space: panel,
    Alert: panel,
    Input: panel,
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
    message: { success: vi.fn(), warning: vi.fn() },
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

describe('menu page edit and movement confirmation isolation', () => {
  let app: App;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.access.accessToken = 'token';
    mocks.formState.value = '';
    mocks.detail.mockImplementation(async (id: number) => ({
      sysBaseMenu: mocks.rows.find((row) => row.ID === id),
    }));
    mocks.preview.mockResolvedValue({
      version: 'preview-for-parent-42',
      authorityIds: [11],
      ancestorIds: [42],
      addedLinks: [{ authorityId: 11, menuId: 42 }],
    });
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
    app = createApp(Menus);
    app.mount(element);
  });
  afterEach(() => {
    app.unmount();
    element.remove();
  });
  function rowAction(id: number, text = '编辑') {
    const row = element.querySelector(`[data-menu="${id}"]`)!;
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
  async function moveFirst() {
    rowAction(1);
    await flush();
    mocks.values.mockResolvedValue({ ...form(), parentId: 42 });
    save();
    await flush();
  }

  it('refreshes the menu grid after saving a definition without a full page reload', async () => {
    rowAction(1);
    await flush();
    mocks.values.mockResolvedValue({ ...form(), title: 'Changed menu title' });
    save();
    await flush();
    expect(mocks.update).toHaveBeenCalledTimes(1);
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });

  it.each([
    {
      scenario: 'a menu without assigned roles',
      initialParent: 0,
      targetParent: 42,
      authorityIds: null,
      ancestorIds: [42],
    },
    {
      scenario: 'a move to the root level',
      initialParent: 42,
      targetParent: 0,
      authorityIds: [11],
      ancestorIds: null,
    },
  ])(
    'offers movement confirmation for $scenario with empty protobuf collections',
    async (scenario) => {
      const api = await vi.importActual<
        typeof import('#/api/business/system/menu')
      >('#/api/business/system/menu');
      mocks.preview.mockImplementation(api.previewMenuMove);
      mocks.previewPost.mockResolvedValue({
        version: 'bound-empty-preview',
        authorityIds: scenario.authorityIds,
        ancestorIds: scenario.ancestorIds,
        addedLinks: null,
      });
      mocks.detail.mockResolvedValueOnce({
        sysBaseMenu: { ...mocks.rows[0]!, parentId: scenario.initialParent },
      });
      rowAction(1);
      await flush();
      mocks.values.mockResolvedValue({
        ...form(),
        parentId: scenario.targetParent,
      });
      save();
      await flush();

      expect(mocks.confirm).toHaveBeenCalledTimes(1);
      expect(mocks.confirm.mock.calls[0]![2]).toContain('新增关联：[]');
      await mocks.confirm.mock.calls[0]![1]();
      expect(mocks.update).toHaveBeenCalledWith(
        expect.objectContaining({
          ID: 1,
          parentId: scenario.targetParent,
          movePreviewVersion: 'bound-empty-preview',
        }),
        'token',
      );
    },
  );

  it('keeps the new menu selected when an earlier detail response arrives late', async () => {
    const old = deferred<unknown>();
    mocks.detail.mockReturnValueOnce(old.promise);
    rowAction(1);
    rowAction(2);
    await flush();
    expect(form().ID).toBe(2);
    old.resolve({ sysBaseMenu: mocks.rows[0] });
    await flush();
    expect(form().ID).toBe(2);
    expect(
      element.querySelector<HTMLButtonElement>('[data-save]')!.disabled,
    ).toBe(false);
  });
  it('does not restore loaded or saving after cancel while a detail request is pending', async () => {
    const old = deferred<unknown>();
    const current = deferred<unknown>();
    mocks.detail
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(current.promise);
    rowAction(1);
    await flush();
    cancel();
    await flush();
    rowAction(2);
    await flush();
    old.resolve({ sysBaseMenu: mocks.rows[0] });
    await flush();
    expect(
      element.querySelector<HTMLButtonElement>('[data-save]')!.disabled,
    ).toBe(true);
    expect(saving()).toBe('false');
    current.resolve({ sysBaseMenu: mocks.rows[1] });
    await flush();
    expect(form().ID).toBe(2);
  });
  it('binds the movement preview and confirmed payload to the original parent target', async () => {
    await moveFirst();
    expect(mocks.preview).toHaveBeenCalledWith(1, 42, 'token');
    const confirm = mocks.confirm.mock.calls[0]![1];
    mocks.values.mockResolvedValue({
      ...form(),
      parentId: 99,
      name: 'later-draft',
    });
    await confirm();
    await flush();
    expect(mocks.update).toHaveBeenCalledWith(
      expect.objectContaining({
        ID: 1,
        parentId: 42,
        name: 'first',
        movePreviewVersion: 'preview-for-parent-42',
        parameters: [{ key: 'keep', value: 'one' }],
      }),
      'token',
    );
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
  });
  it('does not apply the pending movement confirmation after the editor is cancelled', async () => {
    await moveFirst();
    const confirm = mocks.confirm.mock.calls[0]![1];
    cancel();
    await flush();
    await confirm();
    expect(mocks.update).not.toHaveBeenCalled();
    expect(mocks.refresh).not.toHaveBeenCalled();
  });
  it('does not submit a pending movement confirmation under a changed login', async () => {
    await moveFirst();
    const confirm = mocks.confirm.mock.calls[0]![1];
    await Promise.resolve();
    mocks.access.accessToken = 'new-token';
    await confirm();
    expect(mocks.update).not.toHaveBeenCalled();
    expect(mocks.refresh).not.toHaveBeenCalled();
  });
  it('discards a movement preview arriving after a different menu is opened', async () => {
    const preview = deferred<unknown>();
    mocks.preview.mockReturnValueOnce(preview.promise);
    await moveFirst();
    expect(saving()).toBe('true');
    rowAction(2);
    await flush();
    preview.resolve({
      version: 'old',
      authorityIds: [11],
      ancestorIds: [42],
      addedLinks: [],
    });
    await flush();
    expect(form().ID).toBe(2);
    expect(saving()).toBe('false');
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(mocks.update).not.toHaveBeenCalled();
  });
  it('does not close or refresh a new menu after an old write response completes', async () => {
    const write = deferred<null>();
    mocks.update.mockReturnValueOnce(write.promise);
    rowAction(1);
    await flush();
    save();
    await flush();
    expect(saving()).toBe('true');
    rowAction(2);
    await flush();
    write.resolve(null);
    await flush();
    expect(form().ID).toBe(2);
    expect(saving()).toBe('false');
    expect(mocks.refresh).not.toHaveBeenCalled();
    expect(mocks.query).not.toHaveBeenCalled();
  });
  it('does not delete a menu when the confirmation belongs to a previous login', async () => {
    rowAction(1, '删除');
    const confirm = mocks.confirm.mock.calls[0]![1];
    await Promise.resolve();
    mocks.access.accessToken = 'new-token';
    await confirm();
    expect(mocks.remove).not.toHaveBeenCalled();
    expect(mocks.refresh).not.toHaveBeenCalled();
    expect(mocks.query).not.toHaveBeenCalled();
  });
});
