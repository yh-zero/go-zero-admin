/* eslint-disable vue/one-component-per-file -- Controls are stubbed to check the real avatar preview URL. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  upload: vi.fn(),
  success: vi.fn(),
  error: vi.fn(),
}));
vi.mock('#/api/business/base', () => ({ uploadImage: mocks.upload }));
vi.mock('antdv-next', () => ({
  Button: { render: () => null },
  Input: { render: () => null },
  message: { success: mocks.success, error: mocks.error },
}));

describe('avatar preview under a subpath deployment', () => {
  let app: App | undefined;
  let element: HTMLDivElement | undefined;
  afterEach(() => {
    app?.unmount();
    element?.remove();
    vi.unstubAllEnvs();
    vi.resetModules();
    vi.resetAllMocks();
  });
  it.each([
    ['/default-avatar.svg', '/console/default-avatar.svg'],
    [
      'https://files.example.test/avatar.png',
      'https://files.example.test/avatar.png',
    ],
  ])(
    'renders %s at %s without changing its saved value',
    async (value, expected) => {
      vi.stubEnv('BASE_URL', '/console/');
      vi.resetModules();
      const { default: ImageUpload } = await import('./image-upload.vue');
      const update = vi.fn();
      element = document.createElement('div');
      document.body.append(element);
      app = createApp(ImageUpload, { value, 'onUpdate:value': update });
      app.mount(element);
      await nextTick();
      expect(element.querySelector('img')?.getAttribute('src')).toBe(expected);
      expect(update).not.toHaveBeenCalled();
    },
  );
  it.each([
    [new File([], 'empty.png', { type: 'image/png' }), '不能为空'],
    [
      new File(['image'], `${'x'.repeat(256)}.png`, { type: 'image/png' }),
      '文件名',
    ],
    [new File(['svg'], 'unsafe.svg', { type: 'image/svg+xml' }), '图片'],
    [
      Object.defineProperty(
        new File(['image'], 'big.png', { type: 'image/png' }),
        'size',
        { value: 10 * 1024 * 1024 + 1 },
      ),
      '10 MB',
    ],
  ])(
    'rejects an invalid avatar before upload: %s',
    async (file, expectedError) => {
      mocks.upload.mockResolvedValue({
        fileId: 7,
        fileImgUrl: 'https://files.example.test/avatar.png',
      });
      const { default: ImageUpload } = await import('./image-upload.vue');
      element = document.createElement('div');
      document.body.append(element);
      app = createApp(ImageUpload, { canUpload: true });
      app.mount(element);
      const input = element.querySelector<HTMLInputElement>('input[type=file]');
      if (!input) throw new Error('Image file input is missing');
      Object.defineProperty(input, 'files', { value: [file] });
      input.dispatchEvent(new Event('change'));
      await nextTick();
      expect(mocks.error).toHaveBeenCalledWith(
        expect.stringContaining(expectedError),
      );
      expect(mocks.upload).not.toHaveBeenCalled();
    },
  );
});
