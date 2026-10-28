/* eslint-disable vue/one-component-per-file -- Lightweight component stubs isolate the authorization workflow. */
import type { App } from 'vue';

import type { Menu } from '#/api/business/system/types';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import MenuPermissions from './menu-permissions.vue';

const mocks = vi.hoisted(() => ({
  getMenuTree: vi.fn(),
  token: { accessToken: 'test-token' },
  confirm: vi.fn(),
  getAuthorityMenus: vi.fn(),
  saveAuthorityMenus: vi.fn(),
  refreshRoleAccess: vi.fn(),
}));
vi.mock('@vben/stores', () => ({ useAccessStore: () => mocks.token }));
vi.mock('#/api/business/system/permissions', () => ({
  getPermissionEdit: async (id: number) => {
    const [all, assigned] = await Promise.all([
      mocks.getMenuTree(),
      mocks.getAuthorityMenus(id),
    ]);
    return {
      revision: '7',
      authorityId: id,
      menus: all.list,
      menuIds: assigned.list.map((menu: Menu) => menu.ID),
    };
  },
}));
vi.mock('#/api/business/system/menu', () => ({
  getMenuTree: mocks.getMenuTree,
  getAuthorityMenus: mocks.getAuthorityMenus,
}));
vi.mock('#/api/business/system/authority', () => ({
  saveAuthorityMenus: mocks.saveAuthorityMenus,
}));
vi.mock('../shared', () => ({
  flattenTree: (nodes: Menu[]): Menu[] =>
    nodes.flatMap((node) => [node, ...(node.children ?? [])]),
  refreshRoleAccess: mocks.refreshRoleAccess,
  confirmAction: (_title: string, action: () => Promise<void>) =>
    mocks.confirm(action),
}));
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  const panel = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', [slots.default?.(), slots.footer?.()]),
  });
  return {
    Alert: panel,
    Drawer: panel,
    Spin: panel,
    Space: panel,
    Button: defineComponent({
      props: { disabled: Boolean },
      setup:
        (props, { attrs, slots }) =>
        () =>
          h(
            'button',
            { ...attrs, disabled: props.disabled },
            slots.default?.(),
          ),
    }),
    Tree: defineComponent({
      props: { checkedKeys: { type: Array, default: () => [] } },
      setup: (props) => () =>
        h('div', { 'data-checked': JSON.stringify(props.checkedKeys) }),
    }),
    message: { success: vi.fn() },
  };
});
function menu(id: number): Menu {
  return {
    ID: id,
    parentId: 0,
    name: `menu${id}`,
    path: `menu${id}`,
    component: '',
    sort: 0,
    hidden: false,
    meta: { title: `Menu ${id}`, icon: '' },
  };
}
function deferred() {
  let resolve!: (value: { list: Menu[] }) => void;
  const promise = new Promise<{ list: Menu[] }>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
describe('role menu authorization loading', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.token.accessToken = 'test-token';
    mocks.confirm.mockImplementation((action) => action());
    mocks.getMenuTree.mockResolvedValue({ list: [menu(1), menu(2)] });
    mocks.saveAuthorityMenus.mockResolvedValue(null);
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  function mount() {
    app = createApp(MenuPermissions);
    return app.mount(element) as unknown as {
      show: (id: number, name: string) => Promise<void>;
    };
  }
  function saveButton() {
    return [...element.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('保存菜单授权'),
    )!;
  }
  it('does not overwrite a newly selected role with an older late response', async () => {
    const old = deferred();
    mocks.getAuthorityMenus.mockImplementation((id: number) =>
      id === 11 ? old.promise : Promise.resolve({ list: [menu(2)] }),
    );
    const view = mount();
    const first = view.show(11, 'Old role');
    await view.show(22, 'New role');
    old.resolve({ list: [menu(1)] });
    await first;
    await nextTick();
    expect(
      element.querySelector('[data-checked]')?.getAttribute('data-checked'),
    ).toBe('[2]');
    saveButton().click();
    await Promise.resolve();
    await nextTick();
    expect(mocks.saveAuthorityMenus).toHaveBeenCalledExactlyOnceWith(
      22,
      [2],
      '7',
      'test-token',
    );
    expect(mocks.refreshRoleAccess).toHaveBeenCalledExactlyOnceWith(22);
  });
  it('cannot save an empty replacement when loading a role fails', async () => {
    mocks.getAuthorityMenus.mockRejectedValue(new Error('permission denied'));
    const view = mount();
    await expect(view.show(22, 'Denied role')).rejects.toThrow(
      'permission denied',
    );
    await nextTick();
    expect(saveButton().disabled).toBe(true);
    saveButton().click();
    await nextTick();
    expect(mocks.saveAuthorityMenus).not.toHaveBeenCalled();
  });
});
