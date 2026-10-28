import { beforeEach, describe, expect, it, vi } from 'vitest';

import { initComponentAdapter } from '../index';

const mocks = vi.hoisted(() => ({
  components: {} as Record<string, any>,
  post: vi.fn(),
  legacyUpload: vi.fn(),
}));
vi.mock('#/api/request', () => ({
  requestClient: { post: mocks.post, upload: mocks.legacyUpload },
  baseRequestClient: {},
}));
vi.mock('@vben/common-ui', () => ({
  ApiComponent: {},
  IconPicker: {},
  VCropper: {},
  VbenCollapsibleParams: {},
  globalShareState: {
    setComponents: (components: Record<string, any>) => {
      mocks.components = components;
    },
    defineMessage: vi.fn(),
  },
}));
vi.mock('@vben/hooks', () => ({ useSortable: vi.fn() }));
vi.mock('@vben/icons', () => ({ IconifyIcon: {} }));
vi.mock('@vben/locales', () => ({ $t: (key: string) => key }));
vi.mock('@vben/plugins/tiptap', () => ({ VbenTiptap: {} }));
vi.mock('@vben/utils', () => ({ isEmpty: vi.fn() }));
vi.mock('@vben/stores', () => ({ useAccessStore: vi.fn() }));
vi.mock('antdv-next', () => ({ message: {}, Modal: {}, notification: {} }));
vi.mock('../theme-aware-button', () => ({
  createThemeAwareButton: (component: unknown) => component,
}));

function editorUpload() {
  const render = mocks.components.RichEditor.setup(
    {},
    { attrs: {}, slots: {}, expose: () => {} },
  );
  return render().props.imageUpload.upload as (
    file: File,
    onProgress?: (percent: number) => void,
  ) => Promise<string>;
}

describe('registered rich editor image upload', () => {
  beforeEach(async () => {
    vi.resetAllMocks();
    await initComponentAdapter();
  });
  it('uploads a public image through the business contract and returns fileImgUrl', async () => {
    mocks.post.mockResolvedValue({
      fileId: 17,
      fileImgUrl: 'https://files.example.test/new.png',
    });
    mocks.legacyUpload.mockResolvedValue({
      url: 'https://legacy.example.test/old.png',
    });
    const file = new File(['image'], 'new.png', { type: 'image/png' });
    const progress = vi.fn();
    await expect(editorUpload()(file, progress)).resolves.toBe(
      'https://files.example.test/new.png',
    );
    expect(mocks.legacyUpload).not.toHaveBeenCalled();
    const request = mocks.post.mock.calls[0];
    if (!request) throw new Error('Business image upload request is missing');
    const [path, data, options] = request;
    expect(path).toBe('/v1/sys/base/uploadFileImg');
    expect(data.get('file_img')).toBe(file);
    expect(data.get('visibility')).toBe('public');
    expect(options.headers['Content-Type']).toBeUndefined();
    expect(progress.mock.calls.map((entry) => entry[0])).toEqual([0, 100]);
  });
  it('does not send an empty image to either upload endpoint', async () => {
    const file = new File([], 'empty.png', { type: 'image/png' });
    await expect(editorUpload()(file)).rejects.toThrow('不能为空');
    expect(mocks.post).not.toHaveBeenCalled();
    expect(mocks.legacyUpload).not.toHaveBeenCalled();
  });
});
