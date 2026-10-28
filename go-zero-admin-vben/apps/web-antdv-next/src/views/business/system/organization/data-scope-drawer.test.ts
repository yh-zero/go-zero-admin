/* eslint-disable vue/one-component-per-file -- Lightweight component stubs isolate the scope workflow. */
import type { App } from 'vue';

import type { RoleDataScope } from '#/api/business/system/organization';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import DataScopeDrawer from './data-scope-drawer.vue';

const mocks = vi.hoisted(() => ({
  token: { accessToken: 'test-token' },
  getDepartments: vi.fn(),
  getRoleDataScope: vi.fn(),
  updateRoleDataScope: vi.fn(),
  warning: vi.fn(),
}));
vi.mock('@vben/stores', () => ({ useAccessStore: () => mocks.token }));
vi.mock('#/api/business/system/permissions', () => ({
  getPermissionEdit: async (id: number) => {
    const [dataScope, departments] = await Promise.all([
      mocks.getRoleDataScope(id),
      mocks.getDepartments(),
    ]);
    return { revision: '7', authorityId: id, dataScope, departments };
  },
}));
vi.mock('#/api/business/system/organization', () => mocks);
vi.mock('../shared', () => ({
  confirmAction: (_title: string, action: () => Promise<void>) => action(),
}));
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  const panel = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', [slots.default?.(), slots.footer?.(), slots.action?.()]),
  });
  return {
    Drawer: panel,
    Spin: panel,
    Space: panel,
    Alert: panel,
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
    Select: defineComponent({
      props: { value: { type: String, default: 'self' }, disabled: Boolean },
      emits: ['update:value'],
      setup:
        (props, { emit }) =>
        () =>
          h(
            'select',
            {
              value: props.value,
              disabled: props.disabled,
              onChange: (event: Event) =>
                emit('update:value', (event.target as HTMLSelectElement).value),
            },
            [
              'all',
              'self',
              'department',
              'department_and_children',
              'custom',
            ].map((value) => h('option', { value }, value)),
          ),
    }),
    Tree: defineComponent({
      props: {
        checkedKeys: { type: Array, default: () => [] },
        treeData: { type: Array, default: () => [] },
      },
      emits: ['check'],
      setup:
        (props, { emit }) =>
        () =>
          h(
            'div',
            {
              'data-checked': JSON.stringify(props.checkedKeys),
              'data-tree': JSON.stringify(props.treeData),
            },
            [
              h(
                'button',
                {
                  'data-uncheck': true,
                  onClick: () => emit('check', { checked: [] }),
                },
                '取消勾选',
              ),
            ],
          ),
    }),
    message: { success: vi.fn(), info: vi.fn(), warning: mocks.warning },
  };
});
function deferred() {
  let resolve!: (value: RoleDataScope) => void;
  const promise = new Promise<RoleDataScope>((done) => {
    resolve = done;
  });
  return { resolve, promise };
}
describe('role data scope workflow', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.getDepartments.mockResolvedValue([
      { id: 1, name: '研发', code: 'dev', parentId: 0, sort: 0, status: 1 },
    ]);
    mocks.updateRoleDataScope.mockResolvedValue(null);
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  function mount() {
    app = createApp(DataScopeDrawer);
    return app.mount(element) as unknown as {
      show: (id: number, name: string) => Promise<void>;
    };
  }
  function saveButton() {
    return [...element.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('保存数据范围'),
    )!;
  }
  async function selectScope(value: string) {
    const input = element.querySelector('select')!;
    input.value = value;
    input.dispatchEvent(new Event('change'));
    await nextTick();
  }
  async function flush() {
    await Promise.resolve();
    await Promise.resolve();
    await nextTick();
  }
  it('ignores a late response for an older role and clears custom ids when switching scope', async () => {
    const old = deferred();
    mocks.getRoleDataScope.mockImplementation((id: number) =>
      id === 11
        ? old.promise
        : Promise.resolve({
            authorityId: id,
            scope: 'custom',
            departmentIds: [1],
          }),
    );
    const view = mount();
    const first = view.show(11, '旧角色');
    await view.show(22, '新角色');
    old.resolve({ authorityId: 11, scope: 'self', departmentIds: [] });
    await first;
    await nextTick();
    expect(
      element.querySelector('[data-checked]')?.getAttribute('data-checked'),
    ).toBe('[1]');
    await selectScope('self');
    saveButton().click();
    await flush();
    expect(mocks.updateRoleDataScope).toHaveBeenCalledExactlyOnceWith(
      {
        authorityId: 22,
        expectedRevision: '7',
        scope: 'self',
        departmentIds: [],
      },
      'test-token',
    );
  });
  it('does not overwrite configuration when reading it fails', async () => {
    mocks.getRoleDataScope.mockRejectedValue(new Error('permission denied'));
    await mount().show(22, '受限角色');
    await nextTick();
    expect(saveButton().disabled).toBe(true);
    saveButton().click();
    await flush();
    expect(mocks.updateRoleDataScope).not.toHaveBeenCalled();
  });
  it('allows restoring an original disabled custom grant within the same edit', async () => {
    mocks.getDepartments.mockResolvedValue([
      { id: 1, name: '原部门', code: 'old', parentId: 0, sort: 0, status: 2 },
      {
        id: 2,
        name: '未授权部门',
        code: 'new',
        parentId: 0,
        sort: 0,
        status: 2,
      },
    ]);
    mocks.getRoleDataScope.mockResolvedValue({
      authorityId: 22,
      scope: 'custom',
      departmentIds: [1],
    });
    await mount().show(22, '普通角色');
    await nextTick();
    element.querySelector<HTMLButtonElement>('[data-uncheck]')!.click();
    await nextTick();
    const tree = element.querySelector('[data-tree]')!;
    expect(tree.getAttribute('data-checked')).toBe('[]');
    expect(JSON.parse(tree.getAttribute('data-tree')!)).toEqual([
      { key: 1, title: '原部门（停用）', disabled: false, children: [] },
      { key: 2, title: '未授权部门（停用）', disabled: true, children: [] },
    ]);
  });
  it('blocks empty custom scopes and treats administrator scope as read-only', async () => {
    mocks.getRoleDataScope.mockImplementation((id: number) =>
      Promise.resolve({ authorityId: id, scope: 'self', departmentIds: [] }),
    );
    const view = mount();
    await view.show(22, '普通角色');
    await nextTick();
    await selectScope('custom');
    saveButton().click();
    await flush();
    expect(mocks.warning).toHaveBeenCalledWith('指定部门范围至少选择一个部门');
    expect(mocks.updateRoleDataScope).not.toHaveBeenCalled();
    await view.show(1, '管理员');
    await nextTick();
    expect(saveButton().disabled).toBe(true);
    expect(element.querySelector('select')?.value).toBe('all');
  });
  it('prevents duplicate writes while saving', async () => {
    mocks.getRoleDataScope.mockResolvedValue({
      authorityId: 22,
      scope: 'self',
      departmentIds: [],
    });
    const pending = deferred();
    mocks.updateRoleDataScope.mockReturnValue(pending.promise);
    await mount().show(22, '普通角色');
    await nextTick();
    await selectScope('all');
    saveButton().click();
    saveButton().click();
    await flush();
    expect(mocks.updateRoleDataScope).toHaveBeenCalledTimes(1);
    pending.resolve({ authorityId: 22, scope: 'all', departmentIds: [] });
    await flush();
  });
});
