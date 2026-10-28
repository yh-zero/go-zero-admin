/* eslint-disable vue/one-component-per-file -- Test the real manual-reference guard at the request boundary. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Reference from './reference.vue';

const mocks = vi.hoisted(() => ({
  values: {} as Record<string, unknown>,
  add: vi.fn(),
  remove: vi.fn(),
}));
vi.mock('#/api/business/system/files', () => ({
  addFileReference: mocks.add,
  removeFileReference: mocks.remove,
}));
vi.mock('#/adapter/form', () => {
  const schema = { regex: () => schema };
  return {
    z: { string: () => schema },
    useVbenForm: () => [
      { render: () => null },
      {
        reset: async () => {},
        setValues: async (values: Record<string, unknown>) => {
          mocks.values = values;
        },
        validate: async () => ({ valid: true }),
        getValues: async () => mocks.values,
      },
    ],
  };
});
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    message: { success: vi.fn() },
    Alert: defineComponent({
      props: { message: { type: String, default: '' } },
      setup: (props) => () => h('p', props.message),
    }),
    Modal: defineComponent({
      emits: ['ok'],
      setup:
        (_, { emit, slots }) =>
        () =>
          h('div', [
            slots.default?.(),
            h('button', { onClick: () => emit('ok') }, '保存引用'),
          ]),
    }),
  };
});
describe('manual file references exclude system avatar links', () => {
  let app: App;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    element = document.createElement('div');
    document.body.append(element);
    app = createApp(Reference);
  });
  afterEach(() => {
    app.unmount();
    element.remove();
  });
  function saveReference() {
    const button = element.querySelector('button');
    if (!button) throw new Error('Save reference button is missing');
    button.click();
  }
  it.each(['sys_user_avatar', 'app_user_avatar'])(
    'blocks manual changes to %s without sending a request',
    async (objectType) => {
      const view = app.mount(element) as unknown as {
        show: (file: { id: number; name: string }) => Promise<void>;
      };
      await view.show({ id: 7, name: 'avatar.png' });
      mocks.values = { operation: 'remove', objectType, objectId: '1' };
      saveReference();
      for (let i = 0; i < 8; i++) await Promise.resolve();
      await nextTick();
      expect(mocks.add).not.toHaveBeenCalled();
      expect(mocks.remove).not.toHaveBeenCalled();
      expect(element.textContent).toContain('系统头像引用');
    },
  );
  it('preserves ordinary business reference updates', async () => {
    const view = app.mount(element) as unknown as {
      show: (file: { id: number; name: string }) => Promise<void>;
    };
    await view.show({ id: 7, name: 'photo.png' });
    mocks.values = { operation: 'add', objectType: 'article', objectId: '42' };
    saveReference();
    for (let i = 0; i < 8; i++) await Promise.resolve();
    expect(mocks.add).toHaveBeenCalledWith({
      fileId: 7,
      objectType: 'article',
      objectId: '42',
    });
  });
});
