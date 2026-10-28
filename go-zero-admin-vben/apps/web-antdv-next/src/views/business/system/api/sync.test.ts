/* eslint-disable vue/one-component-per-file -- Minimal controls isolate API synchronization behavior. */
import type { App } from 'vue';

import type { ApiSyncItem, ApiSyncPreview } from '#/api/business/system/api';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Sync from './sync.vue';

const mocks = vi.hoisted(() => ({
  preview: vi.fn(),
  apply: vi.fn(),
  saved: vi.fn(),
  success: vi.fn(),
}));
vi.mock('#/api/business/system/api', () => ({
  previewApiSync: mocks.preview,
  applyApiSync: mocks.apply,
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
    Drawer: panel,
    Space: panel,
    Spin: panel,
    Empty: panel,
    Alert: defineComponent({
      props: { message: String },
      setup: (props) => () => h('p', props.message),
    }),
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
    CheckboxGroup: defineComponent({
      props: {
        value: { type: Array, default: () => [] },
        options: { type: Array, default: () => [] },
        disabled: Boolean,
      },
      emits: ['change'],
      setup:
        (props, { emit }) =>
        () =>
          h(
            'div',
            props.options.map((entry) => {
              const option = entry as { label: string; value: string };
              return h('label', [
                h('input', {
                  type: 'checkbox',
                  value: option.value,
                  checked: props.value.includes(option.value),
                  disabled: props.disabled,
                  onChange: (event: Event) =>
                    emit(
                      'change',
                      (event.target as HTMLInputElement).checked
                        ? [...props.value, option.value]
                        : props.value.filter((value) => value !== option.value),
                    ),
                }),
                option.label,
              ]);
            }),
          ),
    }),
    message: { success: mocks.success },
  };
});

function item(key: string, id = 0): ApiSyncItem {
  const [method, path] = key.split(' ');
  return {
    key,
    id,
    path: path!,
    method: method!,
    apiGroup: '系统',
    description: '说明',
    currentApiGroup: '旧分组',
    currentDescription: '旧说明',
  };
}
function preview(version = 'v1'): ApiSyncPreview {
  return {
    version,
    added: [item('GET /new')],
    changed: [item('PUT /existing', 1)],
    obsolete: [item('DELETE /obsolete', 2)],
  };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
async function flush() {
  for (let index = 0; index < 8; index++) await Promise.resolve();
  await nextTick();
}

describe('API sync review and apply', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    mocks.preview.mockResolvedValue(preview());
    mocks.apply.mockResolvedValue({ added: 1, updated: 1 });
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  function mount() {
    app = createApp(Sync, { onSaved: mocks.saved });
    return app.mount(element) as unknown as { show: () => Promise<void> };
  }
  function button(text: string) {
    return [...element.querySelectorAll('button')].find((entry) =>
      entry.textContent?.includes(text),
    )!;
  }
  async function toggle(key: string) {
    [...element.querySelectorAll<HTMLInputElement>('input')]
      .find((entry) => entry.value === key)!
      .click();
    await nextTick();
  }
  it('keeps selections in both sections and sends only selected keys, never obsolete resources', async () => {
    await mount().show();
    await nextTick();
    expect(button('应用所选').disabled).toBe(true);
    await toggle('GET /new');
    await toggle('PUT /existing');
    await toggle('GET /new');
    expect(element.textContent).toContain('已选 1 / 2');
    button('应用所选').click();
    await flush();
    expect(mocks.apply).toHaveBeenCalledExactlyOnceWith('v1', [
      'PUT /existing',
    ]);
    expect(mocks.saved).toHaveBeenCalledOnce();
    expect(mocks.preview).toHaveBeenCalledTimes(2);
    expect(button('应用所选').disabled).toBe(true);
  });
  it('handles initial preview failure and enables explicit retry without stale selections', async () => {
    mocks.preview.mockRejectedValueOnce(new Error('offline'));
    await expect(mount().show()).resolves.toBeUndefined();
    await nextTick();
    expect(element.textContent).toContain('接口预览加载失败');
    expect(button('应用所选').disabled).toBe(true);
    expect(button('全选可同步项').disabled).toBe(true);
    expect(button('重新预览').disabled).toBe(false);
    button('重新预览').click();
    await flush();
    expect(element.textContent).not.toContain('接口预览加载失败');
    expect(element.querySelectorAll('input')).toHaveLength(2);
  });
  it('does not repeat the preview request when refreshing after a successful apply fails', async () => {
    mocks.preview
      .mockResolvedValueOnce(preview())
      .mockRejectedValue(new Error('offline'));
    await mount().show();
    await nextTick();
    button('全选可同步项').click();
    await nextTick();
    button('应用所选').click();
    await flush();
    expect(mocks.apply).toHaveBeenCalledOnce();
    expect(mocks.saved).toHaveBeenCalledOnce();
    expect(mocks.preview).toHaveBeenCalledTimes(2);
    expect(element.textContent).toContain('接口预览加载失败');
    expect(button('应用所选').disabled).toBe(true);
    expect(button('重新预览').disabled).toBe(false);
  });
  it('reloads one fresh preview after an apply conflict and requires a new explicit selection', async () => {
    mocks.apply.mockRejectedValue(new Error('stale preview'));
    mocks.preview
      .mockResolvedValueOnce(preview())
      .mockResolvedValue(preview('v2'));
    await mount().show();
    await nextTick();
    button('全选可同步项').click();
    await nextTick();
    button('应用所选').click();
    await flush();
    expect(mocks.saved).not.toHaveBeenCalled();
    expect(mocks.success).not.toHaveBeenCalled();
    expect(mocks.preview).toHaveBeenCalledTimes(2);
    expect(element.textContent).toContain('已选 0 / 2');
    expect(button('应用所选').disabled).toBe(true);
  });
  it('disables all write and close actions while applying and ignores duplicate clicks', async () => {
    const applying = deferred<{ added: number; updated: number }>();
    mocks.apply.mockReturnValue(applying.promise);
    await mount().show();
    await nextTick();
    button('全选可同步项').click();
    await nextTick();
    button('应用所选').click();
    await nextTick();
    for (const text of [
      '应用所选',
      '关闭',
      '重新预览',
      '全选可同步项',
      '取消选择',
    ])
      expect(button(text).disabled).toBe(true);
    expect(
      [...element.querySelectorAll<HTMLInputElement>('input')].every(
        (entry) => entry.disabled,
      ),
    ).toBe(true);
    button('应用所选').click();
    expect(mocks.apply).toHaveBeenCalledOnce();
    applying.resolve({ added: 1, updated: 1 });
    await flush();
    expect(button('关闭').disabled).toBe(false);
  });
  it('ignores an older preview response after the drawer is reopened', async () => {
    const old = deferred<ApiSyncPreview>();
    mocks.preview
      .mockReturnValueOnce(old.promise)
      .mockResolvedValue({ ...preview('v2'), added: [item('GET /latest')] });
    const view = mount();
    const first = view.show();
    await view.show();
    old.resolve(preview('v1'));
    await first;
    await nextTick();
    expect(element.textContent).toContain('GET /latest');
    expect(element.textContent).not.toContain('GET /new');
  });
});
