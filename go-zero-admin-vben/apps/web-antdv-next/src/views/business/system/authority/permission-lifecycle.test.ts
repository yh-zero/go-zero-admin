import type { App, Component } from 'vue';

import type { Department } from '#/api/business/system/organization';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import DataScopeDrawer from '../organization/data-scope-drawer.vue';
import ApiPermissions from './api-permissions.vue';
import ButtonPermissions from './button-permissions.vue';
import MenuPermissions from './menu-permissions.vue';
import PermissionHistory from './permission-history.vue';
const mocks = vi.hoisted(() => ({
  access: { accessToken: 'token' },
  edit: vi.fn(),
  write: vi.fn(),
  confirm: vi.fn(),
  refresh: vi.fn(),
  history: vi.fn(),
  preview: vi.fn(),
  rollback: vi.fn(),
}));
vi.mock('@vben/stores', () => ({ useAccessStore: () => mocks.access }));
vi.mock('#/api/business/system/permissions', () => ({
  getPermissionEdit: mocks.edit,
  getPermissionHistory: mocks.history,
  previewPermissionRollback: mocks.preview,
  applyPermissionRollback: mocks.rollback,
}));
vi.mock('#/api/business/system/authority', () => ({
  saveAuthorityMenus: mocks.write,
}));
vi.mock('#/api/business/system/menu', () => ({
  saveAuthorityButtons: mocks.write,
}));
vi.mock('#/api/business/system/casbin', () => ({
  savePolicies: mocks.write,
  savePolicyApiIds: mocks.write,
}));
vi.mock('#/api/business/system/organization', () => ({
  updateRoleDataScope: mocks.write,
}));
vi.mock('../shared', () => ({
  flattenTree: (rows: unknown[]) => rows,
  refreshRoleAccess: mocks.refresh,
  confirmAction: (_title: string, action: () => Promise<void>) =>
    mocks.confirm(action),
}));
vi.mock('./permission-options.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: ['modelValue'],
      emits: ['update:modelValue'],
      setup:
        (props, { emit }) =>
        () =>
          h('div', { 'data-selection': JSON.stringify(props.modelValue) }, [
            h(
              'button',
              {
                'data-clear': true,
                onClick: () => emit('update:modelValue', []),
              },
              'clear',
            ),
          ]),
    }),
  };
});

describe('permission history preview and apply lifecycle', () => {
  let app: App;
  let element: HTMLDivElement;
  let view: { show: (id: number, name: string) => Promise<void> };
  const change = {
    id: 3,
    authorityId: 11,
    kind: 'menu',
    beforeRevision: '6',
    afterRevision: '7',
    before: { menuIds: [1] },
    after: { menuIds: [] },
    actorId: 1,
    actorName: 'Admin',
    traceId: 'trace',
    createdAt: 'today',
  };
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.access.accessToken = 'token';
    mocks.history.mockResolvedValue({ list: [change], total: 1 });
    mocks.preview.mockResolvedValue({
      allowed: true,
      reason: '',
      version: 'bound-preview',
      change,
    });
    mocks.rollback.mockResolvedValue(null);
    element = document.createElement('div');
    document.body.append(element);
    app = createApp(PermissionHistory);
    view = app.mount(element) as unknown as typeof view;
  });
  afterEach(() => {
    app.unmount();
    element.remove();
  });
  async function flush() {
    await Promise.resolve();
    await Promise.resolve();
    await nextTick();
  }
  async function preview() {
    await view.show(11, 'A');
    await nextTick();
    [...element.querySelectorAll('button')]
      .find((button) => button.textContent?.includes('预览回滚'))!
      .click();
    await flush();
  }
  it('shows conservative refusal without offering apply', async () => {
    mocks.preview.mockResolvedValue({
      allowed: false,
      reason: '存在后续授权变更',
      version: '',
      change,
    });
    await preview();
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(mocks.rollback).not.toHaveBeenCalled();
  });
  it('binds apply to event afterRevision and server preview version', async () => {
    await preview();
    await mocks.confirm.mock.calls[0]![0]();
    expect(mocks.rollback).toHaveBeenCalledExactlyOnceWith(
      3,
      '7',
      'bound-preview',
      'token',
    );
  });
  it('rejects a confirmation after switching role or account', async () => {
    await preview();
    const action = mocks.confirm.mock.calls[0]![0];
    await view.show(22, 'B');
    await action();
    expect(mocks.rollback).not.toHaveBeenCalled();
    await preview();
    mocks.access.accessToken = 'other';
    await mocks.confirm.mock.calls[1]![0]();
    expect(mocks.rollback).not.toHaveBeenCalled();
  });
  it('does not refresh a new role after an old rollback response', async () => {
    const pending = deferred<null>();
    mocks.rollback.mockReturnValue(pending.promise);
    await preview();
    const operation = mocks.confirm.mock.calls[0]![0]();
    await view.show(22, 'B');
    pending.resolve(null);
    await operation;
    expect(mocks.refresh).not.toHaveBeenCalled();
    expect(mocks.history).toHaveBeenCalledTimes(2);
  });
});
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  const panel = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', [slots.default?.(), slots.description?.(), slots.footer?.()]),
  });
  return {
    Drawer: panel,
    Alert: panel,
    Spin: panel,
    Space: panel,
    TextArea: panel,
    message: { success: vi.fn(), info: vi.fn(), warning: vi.fn() },
    Button: defineComponent({
      props: ['disabled'],
      setup:
        (props, { attrs, slots }) =>
        () =>
          h(
            'button',
            { ...attrs, disabled: props.disabled },
            slots.default?.(),
          ),
    }),
    Select: defineComponent({
      props: ['value'],
      emits: ['update:value', 'change'],
      setup:
        (props, { emit }) =>
        () =>
          h(
            'select',
            {
              value: props.value,
              onChange: (event: Event) =>
                emit('update:value', (event.target as HTMLSelectElement).value),
            },
            [
              h('option', { value: 'self' }, 'self'),
              h('option', { value: 'all' }, 'all'),
              h('option', { value: 'custom' }, 'custom'),
            ],
          ),
    }),
    Tree: defineComponent({
      props: ['checkedKeys'],
      emits: ['check'],
      setup:
        (props, { emit }) =>
        () =>
          h('div', { 'data-selection': JSON.stringify(props.checkedKeys) }, [
            h(
              'button',
              { 'data-clear': true, onClick: () => emit('check', []) },
              'clear',
            ),
            h(
              'button',
              {
                'data-select-departments': true,
                onClick: () => emit('check', [6, 7]),
              },
              'select departments',
            ),
          ]),
    }),
  };
});
function snapshot(id: number, kind: string, revision = '7') {
  return {
    authorityId: id,
    kind,
    revision,
    menus: [
      {
        ID: 1,
        name: 'user',
        parentId: 0,
        path: 'user',
        sort: 0,
        hidden: false,
        component: '',
        meta: { title: 'Users' },
        menuBtn: [
          {
            ID: 2,
            name: 'create',
            desc: 'Create',
            permissionKey: 'user:create',
          },
        ],
      },
    ],
    menuIds: [1],
    menuBtnIds: [2],
    apis: [{ ID: 3, method: 'GET', path: '/test', description: 'test' }],
    policies: [{ method: 'GET', path: '/test' }],
    departments: [] as Department[],
    dataScope: { authorityId: id, scope: 'self', departmentIds: [] },
  };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
describe.each<[string, Component, string]>([
  ['menu', MenuPermissions, '保存菜单授权'],
  ['button', ButtonPermissions, '保存按钮授权'],
  ['api', ApiPermissions, '保存所选 API 权限'],
  ['dataScope', DataScopeDrawer, '保存数据范围'],
])('%s edit lifecycle', (kind, component, label) => {
  let app: App;
  let element: HTMLDivElement;
  let view: { show: (id: number, name: string) => Promise<void> };
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.access.accessToken = 'token';
    mocks.edit.mockImplementation((id, requested) =>
      Promise.resolve(snapshot(id, requested)),
    );
    mocks.write.mockResolvedValue(null);
    element = document.createElement('div');
    document.body.append(element);
    app = createApp(component);
    view = app.mount(element) as unknown as typeof view;
  });
  afterEach(() => {
    app.unmount();
    element.remove();
  });
  const flush = async () => {
    await Promise.resolve();
    await Promise.resolve();
    await nextTick();
  };
  function button(text: string) {
    return [...element.querySelectorAll('button')].find((button) =>
      button.textContent?.includes(text),
    )!;
  }
  async function change() {
    if (kind === 'dataScope') {
      const select = element.querySelector('select')!;
      select.value = 'all';
      select.dispatchEvent(new Event('change'));
    } else element.querySelector<HTMLButtonElement>('[data-clear]')!.click();
    await nextTick();
  }
  it('blocks an old confirmation after cancelling', async () => {
    await view.show(11, 'A');
    await change();
    button(label).click();
    await flush();
    const confirm = mocks.confirm.mock.calls[0]![0];
    button('取消').click();
    await nextTick();
    await confirm();
    expect(mocks.write).not.toHaveBeenCalled();
  });
  it('blocks an old confirmation after account changes or unmount', async () => {
    await view.show(11, 'A');
    await change();
    button(label).click();
    await flush();
    const confirm = mocks.confirm.mock.calls[0]![0];
    mocks.access.accessToken = 'other';
    await confirm();
    expect(mocks.write).not.toHaveBeenCalled();
    mocks.access.accessToken = 'token';
    app.unmount();
    await confirm();
    expect(mocks.write).not.toHaveBeenCalled();
  });
  it('ignores an old write response after switching edit objects', async () => {
    const pending = deferred<null>();
    mocks.write.mockReturnValue(pending.promise);
    mocks.confirm.mockImplementation((action) => action());
    await view.show(11, 'A');
    await change();
    button(label).click();
    await flush();
    await view.show(22, 'B');
    await nextTick();
    pending.resolve(null);
    await flush();
    expect(mocks.refresh).not.toHaveBeenCalled();
    expect(button(label).disabled).toBe(false);
  });
  it('preserves local edits after conflicts and requires explicit rebase', async () => {
    mocks.confirm.mockImplementation((action) => action());
    await view.show(11, 'A');
    await change();
    mocks.write.mockRejectedValueOnce(new Error('权限已变更'));
    mocks.edit.mockResolvedValue(snapshot(11, kind, '8'));
    button(label).click();
    await flush();
    await flush();
    expect(mocks.write).toHaveBeenCalledTimes(1);
    const rebase = button('已比较');
    expect(rebase).toBeDefined();
    if (kind !== 'dataScope')
      expect(
        element
          .querySelector('[data-selection]')
          ?.getAttribute('data-selection'),
      ).toBe('[]');
    rebase.click();
    await nextTick();
    button(label).click();
    await flush();
    const args = mocks.write.mock.calls[1]!;
    expect(kind === 'dataScope' ? args[0].expectedRevision : args[2]).toBe('8');
  });
  {
    it.each(
      kind === 'menu'
        ? ['removed menu']
        : kind === 'button'
          ? ['removed button', 'revoked menu']
          : kind === 'api'
            ? ['removed API']
            : ['removed department', 'disabled department'],
    )(
      'keeps local edits visible and explicitly removable after %s conflicts',
      async (scenario) => {
        const before = snapshot(11, kind);
        const server = snapshot(11, kind, '8');
        let invalidId: number;
        let retainedId: number;
        if (kind === 'menu') {
          const validMenu = { ...before.menus[0]!, ID: 10, name: 'valid' };
          before.menus.push(validMenu);
          before.menuIds = [1, 10];
          server.menus = [validMenu];
          server.menuIds = [10];
          invalidId = 1;
          retainedId = 10;
        } else if (kind === 'button') {
          const validMenu = {
            ...before.menus[0]!,
            ID: 10,
            name: 'valid',
            menuBtn: [
              {
                ID: 4,
                name: 'update',
                desc: 'Valid',
                permissionKey: 'valid:update',
              },
            ],
          };
          before.menus.push(validMenu);
          before.menuIds = [1, 10];
          before.menuBtnIds = [2, 4];
          server.menus.push(validMenu);
          server.menuIds = scenario === 'revoked menu' ? [10] : [1, 10];
          server.menuBtnIds = [4];
          if (scenario === 'removed button') server.menus[0]!.menuBtn = [];
          invalidId = 2;
          retainedId = 4;
        } else if (kind === 'api') {
          const validApi = {
            ID: 5,
            method: 'POST',
            path: '/valid',
            description: 'valid',
          };
          before.apis.push(validApi);
          before.policies.push({ method: 'POST', path: '/valid' });
          server.apis = [validApi];
          server.policies = [{ method: 'POST', path: '/valid' }];
          invalidId = 3;
          retainedId = 5;
        } else {
          const department = (id: number, status = 1) => ({
            id,
            status,
            name: `Dept ${id}`,
            code: `dept${id}`,
            parentId: 0,
            sort: 0,
          });
          before.departments = [department(6), department(7)];
          server.departments =
            scenario === 'disabled department'
              ? [department(6, 2), department(7)]
              : [department(7)];
          invalidId = 6;
          retainedId = 7;
        }
        mocks.edit.mockResolvedValueOnce(before).mockResolvedValue(server);
        mocks.confirm.mockImplementation((action) => action());
        mocks.write.mockRejectedValueOnce(new Error('权限已变更'));
        await view.show(11, 'A');
        await nextTick();
        if (kind === 'dataScope') {
          const select = element.querySelector('select')!;
          select.value = 'custom';
          select.dispatchEvent(new Event('change'));
          await nextTick();
          element
            .querySelector<HTMLButtonElement>('[data-select-departments]')!
            .click();
          await nextTick();
        }
        button(label).click();
        await flush();
        await flush();
        button('已比较').click();
        await nextTick();
        const selected = () =>
          JSON.parse(
            element
              .querySelector('[data-selection]')!
              .getAttribute('data-selection')!,
          );
        expect(selected()).toEqual([invalidId, retainedId]);
        expect(button(label).disabled).toBe(true);
        const remove = element.querySelector<HTMLButtonElement>(
          `[data-remove-unavailable="${invalidId}"]`,
        );
        expect(remove).not.toBeNull();
        remove!.click();
        await nextTick();
        expect(selected()).toEqual([retainedId]);
        expect(button(label).disabled).toBe(false);
        button(label).click();
        await flush();
        const args = mocks.write.mock.calls[1]!;
        expect(kind === 'dataScope' ? args[0].departmentIds : args[1]).toEqual([
          retainedId,
        ]);
        expect(kind === 'dataScope' ? args[0].expectedRevision : args[2]).toBe(
          '8',
        );
      },
    );
  }
});
