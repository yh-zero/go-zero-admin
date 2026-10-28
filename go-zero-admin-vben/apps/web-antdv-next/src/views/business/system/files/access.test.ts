/* eslint-disable vue/one-component-per-file -- Minimal controls isolate authorization and stale URL responses. */
import type { App } from 'vue';

import type { FileResource } from '#/api/business/system/files';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Access from './access.vue';

const mocks = vi.hoisted(() => ({ url: vi.fn(), success: vi.fn() }));
vi.mock('#/api/business/system/files', () => ({ getFileURL: mocks.url }));
vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  const panel = defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('div', slots.default?.()),
  });
  return {
    Modal: defineComponent({
      emits: ['cancel', 'update:open'],
      setup:
        (_, { slots, emit }) =>
        () =>
          h('div', [
            h(
              'button',
              {
                onClick: () => {
                  emit('update:open', false);
                  emit('cancel');
                },
              },
              '关闭预览',
            ),
            slots.default?.(),
          ]),
    }),
    Space: panel,
    Spin: panel,
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
function resource(id: number): FileResource {
  return {
    id,
    name: `image-${id}.png`,
    mime: 'image/png',
    size: 100,
    ownerId: 10,
    departmentId: 2,
    references: 0,
    visibility: 'private',
    status: 'active',
    createdAt: '2026-10-02T10:00:00Z',
  };
}
async function flush() {
  for (let index = 0; index < 8; index++) await Promise.resolve();
  await nextTick();
}

describe('authorized file access', () => {
  let app: App;
  let element: HTMLDivElement;
  let mounted = false;
  beforeEach(() => {
    vi.resetAllMocks();
    element = document.createElement('div');
    document.body.append(element);
    app = createApp(Access);
  });
  afterEach(() => {
    if (mounted) app.unmount();
    mounted = false;
    element.remove();
    if (vi.isFakeTimers()) vi.clearAllTimers();
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });
  function mount() {
    mounted = true;
    return app.mount(element) as unknown as {
      show: (row: FileResource) => Promise<void>;
    };
  }
  function button(text: string) {
    const entry = [
      ...element.querySelectorAll<HTMLButtonElement>('button'),
    ].find((entry) => entry.textContent?.trim() === text);
    if (!entry) throw new Error(`Button is missing: ${text}`);
    return entry;
  }
  it('keeps a failed authorization retryable and refreshes rather than caches signed URLs', async () => {
    mocks.url
      .mockRejectedValueOnce({
        response: { data: { message: '文件不存在或不在可访问范围' } },
      })
      .mockResolvedValueOnce({
        url: 'https://storage.example.test/image.png?Signature=fresh',
        expiresIn: 300,
      });
    await mount().show(resource(7));
    await nextTick();
    expect(element.textContent).toContain('不在可访问范围');
    expect(element.querySelector('img')).toBeNull();
    const refresh = button('重新获取地址');
    expect(refresh.disabled).toBe(false);
    refresh.click();
    await flush();
    expect(mocks.url.mock.calls.map((call) => call[0])).toEqual([7, 7]);
    expect(element.querySelector('img')?.getAttribute('src')).toContain(
      'Signature=fresh',
    );
    expect(element.textContent).toContain('私有地址有效至');
  });
  it('discards an old response when switching files and rejects unsafe URL schemes', async () => {
    let complete!: (value: { url: string; expiresIn: number }) => void;
    mocks.url
      .mockReturnValueOnce(
        new Promise((resolve) => {
          complete = resolve;
        }),
      )
      .mockResolvedValueOnce({
        url: 'https://storage.example.test/latest.png',
        expiresIn: 300,
      });
    const view = mount();
    const previous = view.show(resource(1));
    await view.show(resource(2));
    complete({ url: 'https://storage.example.test/stale.png', expiresIn: 300 });
    await previous;
    await nextTick();
    expect(element.querySelector('img')?.getAttribute('src')).toContain(
      'latest.png',
    );
    mocks.url.mockResolvedValueOnce({
      url: 'javascript:alert(1)',
      expiresIn: 300,
    });
    await view.show(resource(3));
    await nextTick();
    expect(element.querySelector('img')).toBeNull();
    expect(element.querySelector('a')).toBeNull();
    expect(element.textContent).toContain('文件访问地址无效');
  });
  it('expires a private preview and lets the user request a fresh address', async () => {
    vi.useFakeTimers();
    mocks.url.mockResolvedValue({
      url: 'http://127.0.0.1:9000/private/image.png?Signature=old',
      expiresIn: 2,
    });
    await mount().show(resource(7));
    await nextTick();
    expect(element.querySelector('img')).not.toBeNull();
    await vi.advanceTimersByTimeAsync(2000);
    await nextTick();
    expect(element.querySelector('img')).toBeNull();
    expect(element.querySelector('a')).toBeNull();
    expect(element.textContent).toContain('地址已过期');
    const refresh = button('重新获取地址');
    expect(refresh.disabled).toBe(false);
    mocks.url.mockResolvedValueOnce({
      url: 'http://127.0.0.1:9000/private/image.png?Signature=new',
      expiresIn: 300,
    });
    refresh.click();
    await flush();
    expect(element.querySelector('img')?.getAttribute('src')).toContain(
      'Signature=new',
    );
  });
  it('downloads with a new signed URL and refreshes the visible preview without sending app credentials', async () => {
    mocks.url
      .mockResolvedValueOnce({
        url: 'http://127.0.0.1:9000/private/image.png?Signature=old',
        expiresIn: 300,
      })
      .mockResolvedValueOnce({
        url: 'http://127.0.0.1:9000/private/image.png?Signature=new',
        expiresIn: 300,
      });
    const fetched = vi
      .fn()
      .mockResolvedValue(new Response(new Blob(['image']), { status: 200 }));
    vi.stubGlobal('fetch', fetched);
    const createObjectURL = vi
      .spyOn(URL, 'createObjectURL')
      .mockReturnValue('blob:download');
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    await mount().show(resource(7));
    await nextTick();
    button('下载').click();
    await flush();
    expect(fetched).toHaveBeenCalledWith(
      'http://127.0.0.1:9000/private/image.png?Signature=new',
      expect.objectContaining({
        credentials: 'omit',
        referrerPolicy: 'no-referrer',
      }),
    );
    const request = fetched.mock.calls[0];
    if (!request) throw new Error('File download request is missing');
    expect(request[1].headers).toBeUndefined();
    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(element.querySelector('img')?.getAttribute('src')).toContain(
      'Signature=new',
    );
    expect(mocks.success).toHaveBeenCalledOnce();
  });
  it('removes the previous private preview immediately when download authorization is denied', async () => {
    mocks.url
      .mockResolvedValueOnce({
        url: 'http://127.0.0.1:9000/private/image.png?Signature=old',
        expiresIn: 300,
      })
      .mockRejectedValueOnce({
        response: { data: { message: '文件不存在或不在可访问范围' } },
      });
    const fetched = vi.fn();
    vi.stubGlobal('fetch', fetched);
    await mount().show(resource(7));
    await nextTick();
    expect(element.querySelector('img')).not.toBeNull();
    const download = button('下载');
    download.click();
    await flush();
    expect(element.querySelector('img')).toBeNull();
    expect(element.querySelector('a')).toBeNull();
    expect(element.textContent).not.toContain('私有地址有效至');
    expect(element.textContent).toContain('不在可访问范围');
    expect(download.disabled).toBe(true);
    expect(fetched).not.toHaveBeenCalled();
    expect(mocks.success).not.toHaveBeenCalled();
  });
  it.each(['authorization', 'response', 'blob'] as const)(
    'does not finish a download after leaving during its %s phase',
    async (phase) => {
      vi.useFakeTimers();
      const url = {
        url: 'http://127.0.0.1:9000/private/image.png?Signature=signed',
        expiresIn: 300,
      };
      let finish!: () => void;
      const pending = new Promise<void>((resolve) => {
        finish = resolve;
      });
      mocks.url.mockResolvedValueOnce(url).mockImplementationOnce(async () => {
        if (phase === 'authorization') await pending;
        return url;
      });
      const blob = vi.fn(async () => {
        if (phase === 'blob') await pending;
        return new Blob(['image']);
      });
      const fetched = vi.fn(async () => {
        if (phase === 'response') await pending;
        return { ok: true, blob };
      });
      vi.stubGlobal('fetch', fetched);
      const createURL = vi
        .spyOn(URL, 'createObjectURL')
        .mockReturnValue('blob:download');
      const click = vi
        .spyOn(HTMLAnchorElement.prototype, 'click')
        .mockImplementation(() => {});
      await mount().show(resource(7));
      await nextTick();
      button('下载').click();
      await flush();
      app.unmount();
      mounted = false;
      finish();
      await flush();
      const request = fetched.mock.calls[0] as unknown as
        | [string, RequestInit]
        | undefined;
      expect(request?.[1].signal?.aborted).toBe(
        phase === 'authorization' ? undefined : true,
      );
      expect(fetched).toHaveBeenCalledTimes(phase === 'authorization' ? 0 : 1);
      expect(createURL).not.toHaveBeenCalled();
      expect(click).not.toHaveBeenCalled();
      expect(mocks.success).not.toHaveBeenCalled();
      expect(vi.getTimerCount()).toBe(0);
    },
  );
  it('can reopen another file after canceling a pending download without a stale timeout or busy flag', async () => {
    vi.useFakeTimers();
    let finish!: (value: { url: string; expiresIn: number }) => void;
    mocks.url
      .mockResolvedValueOnce({
        url: 'https://files.example.test/first.png',
        expiresIn: 300,
      })
      .mockReturnValueOnce(
        new Promise((resolve) => {
          finish = resolve;
        }),
      )
      .mockResolvedValueOnce({
        url: 'https://files.example.test/second.png',
        expiresIn: 300,
      });
    const fetched = vi.fn();
    vi.stubGlobal('fetch', fetched);
    const view = mount();
    await view.show(resource(1));
    await nextTick();
    button('下载').click();
    await flush();
    button('关闭预览').click();
    await flush();
    expect(vi.getTimerCount()).toBe(0);
    await view.show(resource(2));
    await nextTick();
    expect(element.querySelector('img')?.getAttribute('src')).toBe(
      'https://files.example.test/second.png',
    );
    finish({ url: 'https://files.example.test/stale.png', expiresIn: 300 });
    await flush();
    expect(fetched).not.toHaveBeenCalled();
    expect(element.querySelector('img')?.getAttribute('src')).toBe(
      'https://files.example.test/second.png',
    );
    const download = button('下载');
    expect(download.disabled).toBe(false);
  });
});
