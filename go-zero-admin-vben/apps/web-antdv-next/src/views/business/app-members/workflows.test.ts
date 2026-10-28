/* eslint-disable vue/one-component-per-file -- Test the real workflows with UI adapter stubs. */
import type { App } from 'vue';

import type { AppUser } from '#/api/business/app-members';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { StaleSessionResponseError } from '#/adapter/business/session';

import Memberships from './memberships.vue';
import Users from './users.vue';

const mocks = vi.hoisted(() => ({
  access: { accessToken: 'old-token' },
  permissions: [] as string[],
  getAppUsers: vi.fn(),
  updateAppUserStatus: vi.fn(),
  updateAppUserAvatar: vi.fn(),
  finishAvatarUpload: undefined as (() => void) | undefined,
  getAppSessions: vi.fn(),
  revokeAppSession: vi.fn(),
  getAppPlans: vi.fn(),
  getAppGrants: vi.fn(),
  grantAppMembership: vi.fn(),
  revokeAppGrant: vi.fn(),
  confirm: vi.fn(),
  success: vi.fn(),
  query: vi.fn(),
  queries: [] as Array<
    (
      page: { page: { currentPage: number; pageSize: number } },
      values: Record<string, unknown>,
    ) => Promise<unknown>
  >,
  forms: [] as Array<Record<string, unknown>>,
  rows: [] as Array<Record<string, unknown>>,
  sessionRows: [] as Array<Record<string, unknown>>,
}));
vi.mock('#/api/business/app-members', () => mocks);
vi.mock('#/components/business/image-upload.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    default: defineComponent({
      props: {
        value: { type: String, default: '' },
        disabled: Boolean,
        canUpload: Boolean,
      },
      emits: ['update:value', 'uploadingChange'],
      setup:
        (props, { emit }) =>
        () =>
          h('div', [
            h('input', {
              'aria-label': '头像地址',
              value: props.value,
              disabled: props.disabled,
              onInput: (event: Event) =>
                emit('update:value', (event.target as HTMLInputElement).value),
            }),
            props.canUpload
              ? h(
                  'button',
                  {
                    disabled: props.disabled,
                    onClick: () => {
                      emit('uploadingChange', true);
                      mocks.finishAvatarUpload = () => {
                        emit(
                          'update:value',
                          'https://files.example.test/uploaded.png',
                        );
                        emit('uploadingChange', false);
                      };
                    },
                  },
                  '上传头像',
                )
              : null,
          ]),
    }),
  };
});
vi.mock('@vben/stores', async () => {
  const { reactive } = await import('vue');
  mocks.access = reactive(mocks.access);
  return { useAccessStore: () => mocks.access };
});
vi.mock('../system/shared', () => ({
  usePermission: (menu: string) => (button: string) =>
    mocks.permissions.includes(`${menu}:${button}`),
  confirmAction: mocks.confirm,
}));
vi.mock('@vben/common-ui', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    Page: defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('main', slots.default?.()),
    }),
  };
});
vi.mock('#/adapter/form', async () => {
  const { defineComponent, h } = await import('vue');
  const schema = {
    trim: () => schema,
    min: () => schema,
    max: () => schema,
    int: () => schema,
    positive: () => schema,
    optional: () => schema,
  };
  const z = { string: () => schema, number: () => schema };
  return {
    z,
    useVbenForm: () => {
      const index = mocks.forms.length;
      mocks.forms.push({});
      return [
        defineComponent({ setup: () => () => h('div') }),
        {
          reset: async () => {
            mocks.forms[index] = {};
          },
          updateSchema: vi.fn(),
          setValues: async (values: Record<string, unknown>) => {
            mocks.forms[index] = { ...values };
          },
          getValues: async () => ({ ...mocks.forms[index] }),
          validate: async () => ({ valid: true }),
        },
      ];
    },
  };
});
vi.mock('#/adapter/vxe-table', async () => {
  const { defineComponent, h, onMounted } = await import('vue');
  return {
    useVbenVxeGrid: (options: {
      gridOptions: {
        columns: Array<{
          field?: string;
          formatter?: (cell: { cellValue: unknown }) => unknown;
        }>;
        proxyConfig: {
          autoLoad?: boolean;
          ajax: { query: (typeof mocks.queries)[number] };
        };
      };
    }) => {
      const index = mocks.queries.length;
      mocks.queries.push(options.gridOptions.proxyConfig.ajax.query);
      return [
        defineComponent({
          setup: (_, { slots }) => {
            onMounted(() => {
              if (index === 1 && options.gridOptions.proxyConfig.autoLoad) {
                void options.gridOptions.proxyConfig.ajax
                  .query({ page: { currentPage: 1, pageSize: 10 } }, {})
                  .catch(() => {});
              }
            });
            return () =>
              h('div', [
                slots['toolbar-tools']?.(),
                ...(index === 1 ? mocks.sessionRows : mocks.rows).map((row) =>
                  h('section', { 'data-row': row.id }, [
                    ...options.gridOptions.columns
                      .filter((column) => column.field)
                      .map((column) => {
                        const value = row[required(column.field)];
                        return h(
                          'span',
                          { 'data-column': column.field },
                          String(
                            column.formatter
                              ? column.formatter({ cellValue: value })
                              : (value ?? ''),
                          ),
                        );
                      }),
                    slots.identity?.({ row }),
                    slots.avatar?.({ row }),
                    slots.actions?.({ row }),
                  ]),
                ),
              ]);
          },
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
      (_, { attrs, slots }) =>
      () =>
        h('div', [attrs.title as string, slots.default?.()]),
  });
  return {
    Alert: panel,
    Space: panel,
    Tag: panel,
    Button: defineComponent({
      setup:
        (_, { attrs, slots }) =>
        () =>
          h('button', attrs, slots.default?.()),
    }),
    Modal: defineComponent({
      props: {
        open: Boolean,
        title: { type: String, default: '' },
        footer: { type: Object, default: undefined },
      },
      emits: ['ok', 'cancel'],
      setup:
        (props, { slots, emit }) =>
        () =>
          props.open
            ? h('div', { 'data-modal': props.title }, [
                slots.default?.(),
                props.footer === null
                  ? null
                  : h('button', { onClick: () => emit('ok') }, '确认操作'),
                h('button', { onClick: () => emit('cancel') }, '关闭窗口'),
              ])
            : null,
    }),
    message: { warning: vi.fn(), success: mocks.success },
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
  for (let i = 0; i < 16; i++) await Promise.resolve();
  await nextTick();
}
describe('app account and VIP administration workflows', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.access.accessToken = 'old-token';
    mocks.permissions = [
      'app-members-users:review',
      'app-members-users:status',
      'app-members-users:revoke-session',
      'app-members-users:avatar',
      'businessTools:upload',
      'app-memberships:grant',
      'app-memberships:revoke',
    ];
    mocks.forms = [];
    mocks.queries = [];
    mocks.rows = [
      {
        id: 7,
        account: 'user@example.com',
        nickname: '用户',
        avatarUrl: 'https://files.example.test/original.png',
        status: 'pending',
        identityKind: 'email',
        identityVerified: false,
        registrationIP: '2001:db8:85a3:0000:0000:8a2e:0370:7334',
        registrationRegion: '中国 · 广东省 · 深圳市',
        reviewReason: '',
        createdAt: '2026-10-19T00:00:00Z',
        updatedAt: '2026-10-19T00:00:00Z',
      } satisfies AppUser,
    ];
    mocks.getAppUsers.mockResolvedValue({
      list: [
        {
          ...mocks.rows[0],
          status: 'active',
        },
      ],
      total: 1,
      pageNo: 1,
      pageSize: 20,
    });
    mocks.getAppPlans.mockResolvedValue({
      list: [
        {
          id: 1,
          code: 'vip',
          name: 'VIP',
          version: 1,
          entitlements: ['profile.export'],
        },
      ],
    });
    mocks.getAppGrants.mockResolvedValue({
      list: [],
      total: 0,
      pageNo: 1,
      pageSize: 10,
    });
    mocks.sessionRows = [
      {
        id: 'session-7',
        userId: 7,
        ip: '127.0.0.1',
        userAgent: 'App test',
        current: false,
        createdAt: '2026-10-19T00:00:00Z',
        expiresAt: '2026-11-19T00:00:00Z',
      },
    ];
    mocks.getAppSessions.mockResolvedValue({ list: mocks.sessionRows });
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  async function mount(component: typeof Users) {
    app = createApp(component);
    app.mount(element);
    await flush();
  }
  function button(text: string) {
    const found = [...element.querySelectorAll('button')].find(
      (item) => item.textContent?.trim() === text,
    );
    expect(found, `user action ${text} should be available`).toBeDefined();
    return required(found);
  }
  const page = { page: { currentPage: 2, pageSize: 20 } };
  it.each([
    [
      '2001:db8:85a3:0000:0000:8a2e:0370:7334',
      '中国 · 广东省 · 深圳市',
      '2001:db8:85a3:0000:0000:8a2e:0370:7334',
      '中国 · 广东省 · 深圳市',
    ],
    ['', '', '未记录', '未知'],
  ])(
    'shows registration IP and region in the list and selected review for %s',
    async (registrationIP, registrationRegion, expectedIP, expectedRegion) => {
      mocks.rows[0] = { ...mocks.rows[0], registrationIP, registrationRegion };
      await mount(Users);
      const row = required(element.querySelector('[data-row="7"]'));
      expect(
        row.querySelector('[data-column="registrationIP"]')?.textContent,
      ).toBe(expectedIP);
      expect(
        row.querySelector('[data-column="registrationRegion"]')?.textContent,
      ).toBe(expectedRegion);
      button('审核通过').click();
      await flush();
      const review = element.querySelector(
        '[data-modal] [aria-label="注册来源"]',
      );
      expect(review).not.toBeNull();
      expect(required(review).textContent).toContain(expectedIP);
      expect(required(review).textContent).toContain(expectedRegion);
    },
  );
  it('filters account records and rejects a late list from the previous login', async () => {
    await mount(Users);
    expect(mocks.queries[0]).toBeTypeOf('function');
    await required(mocks.queries[0])(page, {
      keyword: 'user',
      status: 'pending',
    });
    expect(mocks.getAppUsers).toHaveBeenCalledWith(
      { pageNo: 2, pageSize: 20, keyword: 'user', status: 'pending' },
      'old-token',
    );
    const old = deferred<unknown>();
    mocks.getAppUsers.mockReturnValue(old.promise);
    const pending = required(mocks.queries[0])(page, {});
    mocks.access.accessToken = 'new-token';
    old.resolve({ list: [], total: 0 });
    await expect(pending).rejects.toBeInstanceOf(StaleSessionResponseError);
  });
  it('shows App avatars and hides avatar edits without the dedicated permission', async () => {
    mocks.permissions = [];
    await mount(Users);
    expect(
      element.querySelector('[data-row="7"] img')?.getAttribute('src'),
    ).toBe('https://files.example.test/original.png');
    expect(element.textContent).not.toContain('设置头像');
  });
  it.each(['https://files.example.test/replacement.png', ''])(
    'saves only the selected App avatar including an explicit clear: %s',
    async (avatarUrl) => {
      await mount(Users);
      button('设置头像').click();
      await flush();
      const input = required(
        element.querySelector<HTMLInputElement>('[aria-label="头像地址"]'),
      );
      input.value = avatarUrl;
      input.dispatchEvent(new Event('input'));
      button('确认操作').click();
      await flush();
      expect(mocks.updateAppUserAvatar).toHaveBeenCalledExactlyOnceWith(
        { userId: 7, avatarUrl },
        'old-token',
      );
      expect(mocks.updateAppUserStatus).not.toHaveBeenCalled();
      expect(mocks.query).toHaveBeenCalledOnce();
    },
  );
  it('blocks saving and closing an App avatar while an image upload is pending', async () => {
    await mount(Users);
    button('设置头像').click();
    await flush();
    button('上传头像').click();
    await flush();
    button('确认操作').click();
    button('关闭窗口').click();
    await flush();
    expect(mocks.updateAppUserAvatar).not.toHaveBeenCalled();
    expect(element.querySelector('[data-modal]')).not.toBeNull();
    required(mocks.finishAvatarUpload)();
    await flush();
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserAvatar).toHaveBeenCalledWith(
      { userId: 7, avatarUrl: 'https://files.example.test/uploaded.png' },
      'old-token',
    );
  });
  it('keeps a failed avatar replacement retryable and blocks duplicate saves', async () => {
    await mount(Users);
    button('设置头像').click();
    await flush();
    mocks.updateAppUserAvatar.mockRejectedValueOnce(new Error('storage error'));
    button('确认操作').click();
    await flush();
    expect(element.querySelector('[aria-label="头像地址"]')).not.toBeNull();
    const pending = deferred<unknown>();
    mocks.updateAppUserAvatar.mockReturnValueOnce(pending.promise);
    button('确认操作').click();
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserAvatar).toHaveBeenCalledTimes(2);
    pending.resolve({ message: 'ok' });
    await flush();
    expect(mocks.query).toHaveBeenCalledOnce();
  });
  it('rejects an avatar save after permission is revoked and ignores completion after changing login', async () => {
    await mount(Users);
    button('设置头像').click();
    await flush();
    mocks.permissions = [];
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserAvatar).not.toHaveBeenCalled();
    mocks.permissions = ['app-members-users:avatar'];
    const pending = deferred<unknown>();
    mocks.updateAppUserAvatar.mockReturnValue(pending.promise);
    button('确认操作').click();
    await flush();
    mocks.access.accessToken = 'new-token';
    pending.resolve({ message: 'ok' });
    await flush();
    expect(mocks.success).not.toHaveBeenCalled();
    expect(mocks.query).not.toHaveBeenCalled();
    expect(element.querySelector('[data-modal]')).toBeNull();
  });
  it('requires a rejection reason and keeps failed review input for retry', async () => {
    await mount(Users);
    button('审核拒绝').click();
    await flush();
    expect(element.textContent).toContain(
      '邮箱验证仅确认归属，账号仍需人工审核；未验证的邮箱或手机号请人工核实',
    );
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserStatus).not.toHaveBeenCalled();
    mocks.forms[0] = { reason: '身份信息不符' };
    mocks.updateAppUserStatus.mockRejectedValueOnce(
      new Error('network failure'),
    );
    button('确认操作').click();
    await flush();
    expect(element.querySelector('[data-modal]')).not.toBeNull();
    expect(mocks.forms[0]?.reason).toBe('身份信息不符');
    expect(mocks.query).not.toHaveBeenCalled();
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserStatus).toHaveBeenLastCalledWith(
      { userId: 7, status: 'rejected', reason: '身份信息不符' },
      'old-token',
    );
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });
  it('blocks duplicate approval and ignores its completion after changing login', async () => {
    const pending = deferred<unknown>();
    mocks.updateAppUserStatus.mockReturnValue(pending.promise);
    await mount(Users);
    button('审核通过').click();
    await flush();
    button('确认操作').click();
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserStatus).toHaveBeenCalledTimes(1);
    mocks.access.accessToken = 'new-token';
    pending.resolve({ message: 'ok' });
    await flush();
    expect(mocks.success).not.toHaveBeenCalled();
    expect(mocks.query).not.toHaveBeenCalled();
  });
  it('does not submit a review form opened by another login', async () => {
    await mount(Users);
    button('审核通过').click();
    await flush();
    mocks.access.accessToken = 'new-token';
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserStatus).not.toHaveBeenCalled();
  });
  it('opens user sessions and pins revocation to the selected login', async () => {
    await mount(Users);
    button('查看会话').click();
    await flush();
    expect(mocks.queries[1]).toBeTypeOf('function');
    await required(mocks.queries[1])(page, {});
    expect(mocks.getAppSessions).toHaveBeenCalledWith(7, 'old-token');
    button('撤销会话').click();
    mocks.access.accessToken = 'new-token';
    await required(mocks.confirm.mock.calls[0])[1]();
    expect(mocks.revokeAppSession).not.toHaveBeenCalled();
  });
  it('loads sessions when the dialog grid is mounted and shows read failures', async () => {
    mocks.getAppSessions.mockRejectedValueOnce(new Error('permission denied'));
    await mount(Users);
    button('查看会话').click();
    await flush();
    expect(mocks.getAppSessions).toHaveBeenCalledWith(7, 'old-token');
    expect(element.textContent).toContain('会话加载失败');
  });
  it('retries an uncertain VIP grant with the same idempotency key', async () => {
    await mount(Memberships);
    button('开通 / 续期').click();
    await flush();
    mocks.forms[0] = { userId: 7, planId: 1, days: 30, reason: '人工开通' };
    mocks.grantAppMembership.mockRejectedValueOnce(new Error('response lost'));
    button('确认操作').click();
    await flush();
    const first = required(mocks.grantAppMembership.mock.calls[0])[0];
    expect(first).toEqual({
      userId: 7,
      planId: 1,
      days: 30,
      reason: '人工开通',
      requestId: expect.any(String),
    });
    expect(first.requestId.length).toBeGreaterThan(10);
    expect(element.querySelector('[data-modal]')).not.toBeNull();
    button('确认操作').click();
    await flush();
    expect(mocks.grantAppMembership).toHaveBeenLastCalledWith(
      first,
      'old-token',
    );
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });
  it('rejects invalid VIP days before dispatch and does not reuse an old login form', async () => {
    await mount(Memberships);
    button('开通 / 续期').click();
    await flush();
    mocks.forms[0] = { userId: 7, planId: 1, days: 3651, reason: '人工开通' };
    button('确认操作').click();
    await flush();
    expect(mocks.grantAppMembership).not.toHaveBeenCalled();
    required(mocks.forms[0]).days = 30;
    mocks.access.accessToken = 'new-token';
    button('确认操作').click();
    await flush();
    expect(mocks.grantAppMembership).not.toHaveBeenCalled();
  });
  it('requires a reason before revoking a specific grant', async () => {
    mocks.rows = [
      {
        id: 17,
        userId: 7,
        account: 'user@example.com',
        planId: 1,
        planCode: 'vip',
        planVersion: 1,
        entitlements: ['profile.export'],
        startsAt: '2026-10-19T00:00:00Z',
        endsAt: '2026-11-18T00:00:00Z',
        revokedAt: '',
        reason: '人工开通',
        revokeReason: '',
        createdAt: '2026-10-19T00:00:00Z',
        status: 'active',
      },
    ];
    await mount(Memberships);
    button('撤销授予').click();
    await flush();
    button('确认操作').click();
    await flush();
    expect(mocks.revokeAppGrant).not.toHaveBeenCalled();
    mocks.forms[1] = { reason: '人工撤销' };
    button('确认操作').click();
    await flush();
    expect(mocks.revokeAppGrant).toHaveBeenCalledWith(
      { id: 17, reason: '人工撤销' },
      'old-token',
    );
  });
  it('hides actions without the matching business permission', async () => {
    mocks.permissions = [];
    await mount(Users);
    expect(element.textContent).not.toContain('审核通过');
    expect(element.textContent).not.toContain('审核拒绝');
  });
  it.each([
    ['email', true, '邮箱已验证'],
    ['email', false, '邮箱未验证'],
    ['phone', false, '手机号待人工核实'],
  ])(
    'shows the identity ownership state for %s verified=%s',
    async (identityKind, identityVerified, label) => {
      mocks.rows[0] = { ...mocks.rows[0], identityKind, identityVerified };
      await mount(Users);
      expect(element.textContent).toContain(label);
    },
  );
  it('requires a disabling reason and sends status permission actions separately from review', async () => {
    mocks.permissions = ['app-members-users:status'];
    mocks.rows[0] = { ...mocks.rows[0], status: 'active' };
    await mount(Users);
    button('禁用').click();
    await flush();
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserStatus).not.toHaveBeenCalled();
    mocks.forms[0] = { reason: '人工禁用' };
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserStatus).toHaveBeenCalledWith(
      { userId: 7, status: 'disabled', reason: '人工禁用' },
      'old-token',
    );
  });
  it('restores disabled accounts with a new-login notice', async () => {
    mocks.rows[0] = { ...mocks.rows[0], status: 'disabled' };
    await mount(Users);
    button('恢复').click();
    await flush();
    expect(element.textContent).toContain('恢复后该用户需要重新登录');
    button('确认操作').click();
    await flush();
    expect(mocks.updateAppUserStatus).toHaveBeenCalledWith(
      { userId: 7, status: 'active', reason: '' },
      'old-token',
    );
  });
  it('revokes only the selected device and refreshes its server list once', async () => {
    await mount(Users);
    button('查看会话').click();
    await flush();
    mocks.query.mockClear();
    button('撤销会话').click();
    const pending = deferred<unknown>();
    mocks.revokeAppSession.mockReturnValue(pending.promise);
    const first = required(mocks.confirm.mock.calls[0])[1]();
    await required(mocks.confirm.mock.calls[0])[1]();
    expect(mocks.revokeAppSession).toHaveBeenCalledExactlyOnceWith(
      'session-7',
      'old-token',
    );
    pending.resolve({ message: 'ok' });
    await first;
    expect(mocks.query).toHaveBeenCalledTimes(1);
  });
  it('rejects a device list after the selected dialog was closed', async () => {
    await mount(Users);
    button('查看会话').click();
    await flush();
    const old = deferred<unknown>();
    mocks.getAppSessions.mockReturnValue(old.promise);
    const pending = required(mocks.queries[1])(page, {});
    button('关闭窗口').click();
    await flush();
    old.resolve({ list: mocks.sessionRows });
    await expect(pending).rejects.toBeInstanceOf(StaleSessionResponseError);
  });
  it('blocks duplicate VIP grants and ignores success after the login changes', async () => {
    await mount(Memberships);
    button('开通 / 续期').click();
    await flush();
    mocks.forms[0] = { userId: 7, planId: 1, days: 30, reason: '人工开通' };
    const pending = deferred<unknown>();
    mocks.grantAppMembership.mockReturnValue(pending.promise);
    button('确认操作').click();
    button('确认操作').click();
    await flush();
    expect(mocks.grantAppMembership).toHaveBeenCalledTimes(1);
    mocks.access.accessToken = 'new-token';
    pending.resolve({ id: 17 });
    await flush();
    expect(mocks.success).not.toHaveBeenCalled();
    expect(mocks.query).not.toHaveBeenCalled();
    expect(element.querySelector('[data-modal]')).toBeNull();
  });
  it('filters grants by user and rejects records from a previous login', async () => {
    await mount(Memberships);
    await required(mocks.queries[0])(page, { userId: 7 });
    expect(mocks.getAppGrants).toHaveBeenCalledWith(
      { pageNo: 2, pageSize: 20, userId: 7 },
      'old-token',
    );
    const old = deferred<unknown>();
    mocks.getAppGrants.mockReturnValue(old.promise);
    const pending = required(mocks.queries[0])(page, {});
    mocks.access.accessToken = 'new-token';
    old.resolve({ list: [], total: 0 });
    await expect(pending).rejects.toBeInstanceOf(StaleSessionResponseError);
  });
  it('does not repopulate a grant form with old plans after login changes', async () => {
    const old = deferred<unknown>();
    mocks.getAppPlans.mockReturnValue(old.promise);
    await mount(Memberships);
    button('开通 / 续期').click();
    await flush();
    mocks.access.accessToken = 'new-token';
    old.resolve({
      list: [
        {
          id: 1,
          code: 'vip',
          name: 'PRIVATE PLAN',
          version: 1,
          entitlements: ['profile.export'],
        },
      ],
    });
    await flush();
    expect(element.querySelector('[data-modal]')).toBeNull();
    expect(element.textContent).not.toContain('PRIVATE PLAN');
  });
});
