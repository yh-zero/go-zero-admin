/* eslint-disable vue/one-component-per-file -- Test controls exercise upload status without the UI framework. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Upload from './upload.vue';

const mocks = vi.hoisted(() => ({
  upload: vi.fn(),
  saved: vi.fn(),
  success: vi.fn(),
}));
vi.mock('#/api/business/base', () => ({ uploadImage: mocks.upload }));
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    Space: defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('div', slots.default?.()),
    }),
    Select: defineComponent({
      props: {
        value: { type: String, default: '' },
        options: { type: Array, default: () => [] },
      },
      setup: () => () => h('select'),
    }),
    Alert: defineComponent({
      props: { message: { type: String, default: '' } },
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
    Modal: defineComponent({
      props: {
        open: Boolean,
        confirmLoading: Boolean,
        okButtonProps: { type: Object, default: () => ({}) },
      },
      emits: ['ok'],
      setup:
        (props, { slots, emit }) =>
        () =>
          props.open
            ? h('div', [
                slots.default?.(),
                h(
                  'button',
                  {
                    disabled:
                      props.confirmLoading || props.okButtonProps?.disabled,
                    onClick: () => emit('ok'),
                  },
                  '上传',
                ),
              ])
            : null,
    }),
    message: { success: mocks.success },
  };
});
async function flush() {
  for (let index = 0; index < 8; index++) await Promise.resolve();
  await nextTick();
}

describe('resource upload success and failure states', () => {
  let app: App;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    element = document.createElement('div');
    document.body.append(element);
    app = createApp(Upload, { onSaved: mocks.saved });
  });
  afterEach(() => {
    app.unmount();
    element.remove();
  });
  async function mountAndChoose() {
    const instance = app.mount(element) as unknown as { show: () => void };
    instance.show();
    await nextTick();
    const input = element.querySelector<HTMLInputElement>('input[type=file]');
    if (!input) throw new Error('Image file input is missing');
    Object.defineProperty(input, 'files', {
      value: [new File(['image'], 'test.png', { type: 'image/png' })],
      configurable: true,
    });
    input.dispatchEvent(new Event('change'));
    await nextTick();
    const button = [
      ...element.querySelectorAll<HTMLButtonElement>('button'),
    ].find((button) => button.textContent === '上传');
    if (!button) throw new Error('Upload button is missing');
    return button;
  }
  it('keeps an unsuccessful upload retryable without saved events or fake success', async () => {
    mocks.upload
      .mockRejectedValueOnce({
        response: { data: { message: '文件存储服务未配置' } },
      })
      .mockResolvedValueOnce({
        fileId: 44,
        fileImgUrl: 'https://storage/image.png',
      });
    const button = await mountAndChoose();
    button.click();
    await flush();
    expect(element.textContent).toContain('文件存储服务未配置');
    expect(button.disabled).toBe(false);
    expect(mocks.saved).not.toHaveBeenCalled();
    expect(mocks.success).not.toHaveBeenCalled();
    button.click();
    await flush();
    expect(mocks.upload.mock.calls[0]?.[1]).toBe('private');
    expect(mocks.saved).toHaveBeenCalledExactlyOnceWith({
      fileId: 44,
      fileImgUrl: 'https://storage/image.png',
    });
    expect(mocks.success).toHaveBeenCalledOnce();
  });
  it('prevents duplicate submits while a real upload request is pending', async () => {
    let finish!: (value: { fileId: number; fileImgUrl: string }) => void;
    mocks.upload.mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const button = await mountAndChoose();
    button.click();
    await nextTick();
    button.click();
    expect(button.disabled).toBe(true);
    expect(mocks.upload).toHaveBeenCalledOnce();
    finish({ fileId: 44, fileImgUrl: 'https://storage/image.png' });
    await flush();
    expect(mocks.saved).toHaveBeenCalledOnce();
  });
  it('blocks upload when the current role has no upload permission', async () => {
    app = createApp(Upload, { allowed: false, onSaved: mocks.saved });
    const button = await mountAndChoose();
    button.click();
    await flush();
    expect(button.disabled).toBe(true);
    expect(mocks.upload).not.toHaveBeenCalled();
  });
});
