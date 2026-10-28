/* eslint-disable vue/one-component-per-file -- Test a real page with a minimal grid and permission boundary. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Files from './index.vue';

const mocks = vi.hoisted(() => ({
  role: 2,
  permissions: ['files:delete', 'files:reference'],
  row: { id: 7, name: 'example.png', status: 'active', references: 0 },
  remove: vi.fn(),
  query: vi.fn(),
  confirm: vi.fn(),
  success: vi.fn(),
}));
vi.mock('../shared', () => ({
  usePermission: (menu: string) => (button: string) =>
    mocks.permissions.includes(`${menu}:${button}`),
  isCurrentRole: (role: number) => mocks.role === role,
  confirmAction: mocks.confirm,
}));
vi.mock('#/api/business/system/files', () => ({
  getFiles: vi.fn(),
  deleteFile: mocks.remove,
}));
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
              slots.actions?.({ row: mocks.row }),
            ]),
      }),
      { query: mocks.query, reload: mocks.query },
    ],
  };
});
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
    Tag: panel,
    Alert: defineComponent({
      props: { message: { type: String, default: '' } },
      setup: (props) => () => h('p', props.message),
    }),
    Button: defineComponent({
      props: { disabled: Boolean, loading: Boolean },
      setup:
        (props, { attrs, slots }) =>
        () =>
          h(
            'button',
            { ...attrs, disabled: props.disabled || props.loading },
            slots.default?.(),
          ),
    }),
    message: { success: mocks.success },
  };
});
vi.mock('./upload.vue', () => ({ default: { render: () => null } }));
vi.mock('./access.vue', () => ({ default: { render: () => null } }));
vi.mock('./reference.vue', () => ({ default: { render: () => null } }));

describe('file page authorization and delete recovery', () => {
  let app: App;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.role = 2;
    mocks.permissions = ['files:delete', 'files:reference'];
    mocks.row = { id: 7, name: 'example.png', status: 'active', references: 0 };
    element = document.createElement('div');
    document.body.append(element);
    app = createApp(Files);
  });
  afterEach(() => {
    app.unmount();
    element.remove();
  });
  function button(text: string) {
    return [...element.querySelectorAll<HTMLButtonElement>('button')].find(
      (entry) => entry.textContent?.trim() === text,
    );
  }
  function clickButton(text: string) {
    const entry = button(text);
    if (!entry) throw new Error(`Button is missing: ${text}`);
    entry.click();
  }
  function confirmDelete() {
    const callback = mocks.confirm.mock.calls[0]?.[1];
    if (typeof callback !== 'function')
      throw new Error('Delete confirmation callback is missing');
    return callback();
  }
  it('hides reference administration for nonadministrators even when a button code is granted', async () => {
    app.mount(element);
    await nextTick();
    expect(button('引用')).toBeUndefined();
    expect(button('上传图片')).toBeUndefined();
    expect(button('删除')).toBeDefined();
  });
  it('prevents deletion while referenced and only shows manual references to the administrator', async () => {
    mocks.role = 1;
    mocks.row.references = 2;
    app.mount(element);
    await nextTick();
    expect(button('引用')).toBeDefined();
    expect(button('删除')?.disabled).toBe(true);
    button('删除')?.click();
    expect(mocks.confirm).not.toHaveBeenCalled();
    expect(mocks.remove).not.toHaveBeenCalled();
  });
  it('does not claim success after a storage failure and reloads the server state for a retry', async () => {
    mocks.remove.mockRejectedValueOnce(new Error('storage unavailable'));
    mocks.query.mockImplementation(async () => {
      mocks.row.status = 'deleting';
    });
    app.mount(element);
    await nextTick();
    clickButton('删除');
    await expect(confirmDelete()).rejects.toThrow('storage unavailable');
    await nextTick();
    expect(mocks.remove).toHaveBeenCalledExactlyOnceWith(7);
    expect(mocks.success).not.toHaveBeenCalled();
    expect(mocks.query).toHaveBeenCalledOnce();
    expect(button('重试删除')?.disabled).toBe(false);
  });
  it('does not delete after permission is revoked while confirmation is open', async () => {
    app.mount(element);
    await nextTick();
    clickButton('删除');
    mocks.permissions = [];
    await confirmDelete();
    expect(mocks.remove).not.toHaveBeenCalled();
    expect(mocks.success).not.toHaveBeenCalled();
  });
});
