/* eslint-disable vue/one-component-per-file -- Isolate the real user workflow with lightweight UI stubs. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { StaleSessionResponseError } from '#/adapter/business/session';

import Users from './index.vue';

const mocks = vi.hoisted(() => ({
  realForm: false,
  upload: vi.fn(),
  access: { accessToken: 'old-token' },
  authorities: vi.fn(),
  setValues: vi.fn(),
  getValues: vi.fn(),
  resetPassword: vi.fn(),
  updateUser: vi.fn(),
  resourcePreview: vi.fn(),
  transferResources: vi.fn(),
  deleteUser: vi.fn(),
  logout: vi.fn(),
  fetchUserInfo: vi.fn(),
  confirm: vi.fn(),
  query: vi.fn(),
  users: vi.fn(),
  listQuery:
    vi.fn<
      (
        page: { page: { currentPage: number; pageSize: number } },
        values: { keyword?: string },
      ) => Promise<unknown>
    >(),
  rows: [
    {
      ID: 1,
      userName: 'one',
      nickName: 'One',
      authorityId: 88,
      enable: 1,
      headerImg: '/default-avatar.svg',
    },
    { ID: 2, userName: 'two', nickName: 'Two', authorityId: 88, enable: 1 },
  ],
}));
vi.mock('@vben/stores', () => ({
  useAccessStore: () => mocks.access,
  useUserStore: () => ({ userInfo: { userId: '1' } }),
}));
vi.mock('#/store', () => ({
  useAuthStore: () => ({
    logout: mocks.logout,
    fetchUserInfo: mocks.fetchUserInfo,
  }),
}));
vi.mock('#/api/business/system/authority', () => ({
  getAllAuthorities: mocks.authorities,
}));
vi.mock('#/api/business/system/user', () => ({
  createUser: vi.fn(),
  deleteUser: mocks.deleteUser,
  getUserResourcePreview: mocks.resourcePreview,
  transferUserResources: mocks.transferResources,
  getUsers: mocks.users,
  resetUserPassword: mocks.resetPassword,
  updateUser: mocks.updateUser,
}));
vi.mock('../shared', () => ({
  usePermission: () => () => true,
  flattenTree: (rows: unknown[]) => rows,
  statusOptions: [],
  confirmAction: mocks.confirm,
}));
vi.mock('#/api/business/base', () => ({ uploadImage: mocks.upload }));
vi.mock('../organization/membership-modal.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return { default: defineComponent({ setup: () => () => h('div') }) };
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
  const { setupVbenForm } =
    await import('../../../../../../../packages/@core/ui-kit/form-ui/src/config');
  const { useVbenForm } =
    await import('../../../../../../../packages/@core/ui-kit/form-ui/src/use-vben-form');
  const { z } =
    await import('../../../../../../../packages/@core/ui-kit/form-ui/src/index');
  const Input = defineComponent({
    props: {
      modelValue: { type: [String, Number, Array], default: undefined },
    },
    emits: ['update:modelValue'],
    setup:
      (props, { emit }) =>
      () =>
        h('input', {
          value: props.modelValue ?? '',
          onInput: (event: Event) =>
            emit('update:modelValue', (event.target as HTMLInputElement).value),
        }),
  });
  return {
    z,
    useVbenForm: (options: Parameters<typeof useVbenForm>[0]) => {
      if (mocks.realForm) {
        setupVbenForm({
          rules: {
            required: (value) => !!value || 'Required',
            selectRequired: (value) => value !== undefined || 'Required',
          },
        });
        return useVbenForm({
          ...options,
          // Keep the actual custom component and form binding; stub only
          // unrelated built-in inputs normally supplied by the app adapter.
          schema: options.schema?.map((field) =>
            typeof field.component === 'string'
              ? { ...field, component: Input }
              : field,
          ),
        });
      }
      return [
        defineComponent({ setup: () => () => h('div') }),
        {
          reset: vi.fn(),
          updateSchema: vi.fn(),
          setValues: mocks.setValues,
          validate: vi.fn().mockResolvedValue({ valid: true }),
          getValues: mocks.getValues,
        },
      ];
    },
  };
});
vi.mock('#/adapter/vxe-table', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    useVbenVxeGrid: (options: {
      gridOptions: { proxyConfig: { ajax: { query: typeof mocks.listQuery } } };
    }) => {
      mocks.listQuery.mockImplementation(
        options.gridOptions.proxyConfig.ajax.query,
      );
      return [
        defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', [
                slots['toolbar-tools']?.(),
                ...mocks.rows.map((row) =>
                  h(
                    'section',
                    { 'data-user': row.ID },
                    slots.actions?.({ row }),
                  ),
                ),
              ]),
        }),
        { query: mocks.query, reload: mocks.query },
      ];
    },
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
    Input: defineComponent({
      props: { value: { type: String, default: '' } },
      emits: ['update:value'],
      setup:
        (props, { attrs, emit }) =>
        () =>
          h('input', {
            ...attrs,
            value: props.value ?? '',
            onInput: (event: Event) =>
              emit('update:value', (event.target as HTMLInputElement).value),
          }),
    }),
    Space: panel,
    Tag: panel,
    Button: defineComponent({
      setup:
        (_, { attrs, slots }) =>
        () =>
          h('button', attrs, slots.default?.()),
    }),
    Modal: defineComponent({
      props: { open: Boolean },
      emits: ['ok'],
      setup:
        (props, { slots, emit }) =>
        () =>
          props.open
            ? h('div', { 'data-modal': '' }, [
                slots.default?.(),
                h('button', { onClick: () => emit('ok') }, '保存测试'),
              ])
            : null,
    }),
    message: { warning: vi.fn(), info: vi.fn(), success: vi.fn() },
  };
});
function required<T>(value: null | T | undefined): T {
  if (value === null || value === undefined)
    throw new Error('Expected test fixture or rendered element to exist');
  return value;
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
async function flush() {
  for (let count = 0; count < 12; count++) await Promise.resolve();
  await nextTick();
}

describe('user management request isolation', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.realForm = false;
    mocks.access.accessToken = 'old-token';
    mocks.authorities.mockResolvedValue([
      { authorityId: 88, authorityName: 'role' },
    ]);
    mocks.getValues.mockResolvedValue({
      authorityId: 99,
      authorityIds: [99],
      nickName: 'One',
      enable: 1,
    });
    mocks.resourcePreview.mockResolvedValue({
      userId: 2,
      username: 'two',
      files: 3,
      aiAvailable: true,
      aiConversations: 2,
      aiMessages: 5,
      aiRuns: 2,
      aiReason: '',
      transferMode: 'same_database',
    });
    mocks.transferResources.mockResolvedValue({
      files: 3,
      aiConversations: 0,
      aiMessages: 0,
      aiRuns: 0,
      transactionMode: 'same_database',
    });
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  async function mount() {
    app = createApp(Users);
    app.mount(element);
    await nextTick();
  }
  function click(id: number, text: string) {
    const row = required(element.querySelector(`[data-user="${id}"]`));
    required(
      [...row.querySelectorAll('button')].find(
        (button) => button.textContent?.trim() === text,
      ),
    ).click();
  }
  it('fills the existing avatar into the real form control and preview', async () => {
    mocks.realForm = true;
    await mount();
    click(1, '编辑');
    await vi.waitFor(() => {
      expect(
        element.querySelector<HTMLInputElement>(
          'input[placeholder="图片 URL（可以清空）"]',
        )?.value,
      ).toBe('/default-avatar.svg');
      expect(element.querySelector('img')?.getAttribute('src')).toBe(
        '/default-avatar.svg',
      );
    });
  });
  it.each(['manual', 'upload', 'clear'])(
    'saves the avatar %s update emitted through the real form',
    async (operation) => {
      mocks.realForm = true;
      const avatarURL =
        operation === 'clear' ? '' : 'https://files.example.test/avatar.png';
      mocks.upload.mockResolvedValue({ fileId: 5, fileImgUrl: avatarURL });
      await mount();
      click(1, '编辑');
      await flush();
      const input = required(
        element.querySelector<HTMLInputElement>(
          'input[placeholder="图片 URL（可以清空）"]',
        ),
      );
      if (operation === 'upload') {
        const file = required(
          element.querySelector<HTMLInputElement>('input[type=file]'),
        );
        Object.defineProperty(file, 'files', {
          value: [new File(['avatar'], 'avatar.png', { type: 'image/png' })],
        });
        file.dispatchEvent(new Event('change'));
      } else {
        input.value = avatarURL;
        input.dispatchEvent(new Event('input'));
      }
      await flush();
      required(
        [...element.querySelectorAll('button')].find(
          (button) => button.textContent === '保存测试',
        ),
      ).click();
      await vi.waitFor(() => {
        expect(mocks.updateUser).toHaveBeenCalledWith(
          expect.objectContaining({ ID: 1, headerImg: avatarURL }),
          'old-token',
        );
      });
    },
  );
  it('does not replace the latest selected user with a late role-list response', async () => {
    const first = deferred<unknown[]>();
    const second = deferred<unknown[]>();
    mocks.authorities
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    await mount();
    click(1, '编辑');
    click(2, '编辑');
    second.resolve([]);
    await flush();
    first.resolve([]);
    await flush();
    expect(mocks.setValues).toHaveBeenCalledTimes(1);
    expect(mocks.setValues).toHaveBeenLastCalledWith(
      expect.objectContaining({ ID: 2 }),
    );
  });
  it('previews retained file and AI ownership before confirming account deletion', async () => {
    await mount();
    click(2, '删除');
    await flush();
    expect(mocks.resourcePreview).toHaveBeenCalledWith(2, 'old-token');
    expect(mocks.confirm).toHaveBeenCalledWith(
      expect.stringContaining('two'),
      expect.any(Function),
      expect.stringContaining('文件 3'),
    );
    expect(required(mocks.confirm.mock.calls[0])[2]).toContain('AI 会话 2');
    expect(mocks.deleteUser).not.toHaveBeenCalled();
  });
  it('does not show a stale resource preview after login changes', async () => {
    const read = deferred<unknown>();
    mocks.resourcePreview.mockReturnValue(read.promise);
    await mount();
    click(2, '删除');
    mocks.access.accessToken = 'new-token';
    read.resolve({ files: 3, aiAvailable: false, aiReason: '不同数据库' });
    await flush();
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(mocks.deleteUser).not.toHaveBeenCalled();
  });
  it('provides an explicit resource transfer action for a selected user', async () => {
    await mount();
    expect(element.querySelector('[data-user="2"]')?.textContent).toContain(
      '资源移交',
    );
  });
  it('does not reuse an open transfer form after switching login', async () => {
    await mount();
    click(2, '资源移交');
    await flush();
    const target = required(
      element.querySelector<HTMLInputElement>('input[aria-label="目标用户ID"]'),
    );
    target.value = '3';
    target.dispatchEvent(new Event('input'));
    await flush();
    mocks.access.accessToken = 'new-token';
    required(
      [...element.querySelectorAll('button')].find(
        (button) => button.textContent === '保存测试',
      ),
    ).click();
    await flush();
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(mocks.transferResources).not.toHaveBeenCalled();
  });
  it('submits only explicitly selected resources after transfer confirmation', async () => {
    await mount();
    click(2, '资源移交');
    await flush();
    const target = required(
      element.querySelector<HTMLInputElement>('input[aria-label="目标用户ID"]'),
    );
    target.value = '3';
    target.dispatchEvent(new Event('input'));
    await flush();
    required(
      [...element.querySelectorAll('button')].find(
        (button) => button.textContent === '保存测试',
      ),
    ).click();
    await flush();
    expect(mocks.transferResources).not.toHaveBeenCalled();
    await required(mocks.confirm.mock.calls[0])[1]();
    expect(mocks.transferResources).toHaveBeenCalledWith(
      { userId: 2, targetUserId: 3, includeFiles: true, includeAI: false },
      'old-token',
    );
    expect(mocks.deleteUser).not.toHaveBeenCalled();
  });
  it('does not submit an old resource confirmation under new login credentials', async () => {
    await mount();
    click(2, '资源移交');
    await flush();
    const target = required(
      element.querySelector<HTMLInputElement>('input[aria-label="目标用户ID"]'),
    );
    target.value = '3';
    target.dispatchEvent(new Event('input'));
    await flush();
    required(
      [...element.querySelectorAll('button')].find(
        (button) => button.textContent === '保存测试',
      ),
    ).click();
    await flush();
    mocks.access.accessToken = 'new-token';
    await required(mocks.confirm.mock.calls[0])[1]();
    expect(mocks.transferResources).not.toHaveBeenCalled();
  });
  it('pins deletion and password reset writes to the confirmed login', async () => {
    await mount();
    click(2, '删除');
    await flush();
    await required(mocks.confirm.mock.calls[0])[1]();
    expect(mocks.deleteUser).toHaveBeenCalledWith(2, 'old-token');
    mocks.confirm.mockClear();
    click(2, '重置密码');
    await required(mocks.confirm.mock.calls[0])[1]();
    expect(mocks.resetPassword).toHaveBeenCalledWith(2, 'old-token');
  });
  it('rejects a late user list from a previous login instead of returning it to the grid', async () => {
    const read = deferred<unknown>();
    mocks.users.mockReturnValue(read.promise);
    await mount();
    const pending = mocks.listQuery(
      { page: { currentPage: 1, pageSize: 10 } },
      {},
    );
    mocks.access.accessToken = 'new-token';
    read.resolve({
      list: [{ ID: 99, userName: 'old-private-user' }],
      total: 1,
    });
    await expect(pending).rejects.toBeInstanceOf(StaleSessionResponseError);
  });
  it('does not open an old user form after the login changes', async () => {
    const read = deferred<unknown[]>();
    mocks.authorities.mockReturnValue(read.promise);
    await mount();
    click(1, '编辑');
    mocks.access.accessToken = 'new-token';
    read.resolve([]);
    await flush();
    expect(mocks.setValues).not.toHaveBeenCalled();
    expect(element.querySelector('[data-modal]')).toBeNull();
  });
  it('does not log out a newer login after an old password reset completes', async () => {
    const write = deferred<null>();
    mocks.resetPassword.mockReturnValue(write.promise);
    await mount();
    click(1, '重置密码');
    const saving = required(mocks.confirm.mock.calls[0])[1]();
    mocks.access.accessToken = 'new-token';
    write.resolve(null);
    await saving;
    expect(mocks.logout).not.toHaveBeenCalled();
  });
  it('does not submit an old confirmation using the new login credentials', async () => {
    await mount();
    click(1, '重置密码');
    mocks.access.accessToken = 'new-token';
    await required(mocks.confirm.mock.calls[0])[1]();
    expect(mocks.resetPassword).not.toHaveBeenCalled();
  });
  it('does not log out a newer login after an old self-role update completes', async () => {
    const write = deferred<null>();
    mocks.updateUser.mockReturnValue(write.promise);
    await mount();
    click(1, '编辑');
    await flush();
    required(
      [...element.querySelectorAll('button')].find(
        (button) => button.textContent === '保存测试',
      ),
    ).click();
    await flush();
    expect(mocks.updateUser).toHaveBeenCalledTimes(1);
    mocks.access.accessToken = 'new-token';
    write.resolve(null);
    await flush();
    expect(mocks.logout).not.toHaveBeenCalled();
    expect(mocks.fetchUserInfo).not.toHaveBeenCalled();
  });
});
