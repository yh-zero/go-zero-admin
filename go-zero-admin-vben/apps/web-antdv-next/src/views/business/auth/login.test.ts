/* eslint-disable vue/one-component-per-file -- Local controls isolate login and CAPTCHA behavior. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import Login from './login.vue';

const mocks = vi.hoisted(() => ({
  getCaptcha: vi.fn(),
  authLogin: vi.fn(),
  setFieldValue: vi.fn(),
}));
vi.mock('#/api/core/auth', () => ({ getCaptchaApi: mocks.getCaptcha }));
vi.mock('#/store', () => ({
  useAuthStore: () => ({ authLogin: mocks.authLogin, loginLoading: false }),
}));
vi.mock('#/components/business/image-captcha.vue', async () => {
  const { defineComponent, h } = await import('vue');
  return { default: defineComponent({ setup: () => () => h('div') }) };
});
vi.mock('@vben/common-ui', async () => {
  const { defineComponent, h } = await import('vue');
  const schema = { min: () => schema, regex: () => schema };
  return {
    z: { string: () => schema },
    AuthenticationLogin: defineComponent({
      props: {
        title: { type: String, default: '' },
        subTitle: { type: String, default: '' },
      },
      emits: ['submit'],
      setup: (props, { emit, expose }) => {
        expose({ getFormApi: () => ({ setFieldValue: mocks.setFieldValue }) });
        return () =>
          h('div', [
            h('h1', props.title),
            h('p', props.subTitle),
            h(
              'button',
              {
                onClick: () =>
                  emit('submit', {
                    username: 'admin',
                    password: '123456',
                    captcha: '654321',
                  }),
              },
              '登录',
            ),
          ]);
      },
    }),
  };
});

describe('backend account login', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.getCaptcha.mockResolvedValue({
      captchaId: 'live-id',
      captchaImg: 'image',
    });
    mocks.authLogin.mockResolvedValue({});
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
  });
  async function mount() {
    app = createApp(Login);
    app.mount(element);
    for (let count = 0; count < 6; count++) await Promise.resolve();
    await nextTick();
  }

  it('shows the backend account login instructions without public credentials', async () => {
    await mount();
    expect(element.textContent).toContain('登录管理后台');
    expect(element.textContent).toContain('使用后端账号登录');
    expect(element.textContent).not.toContain('admin / 123456');
    expect(mocks.getCaptcha).toHaveBeenCalledOnce();
  });

  it('submits account credentials with the current CAPTCHA identifier', async () => {
    await mount();
    const submit = element.querySelector('button');
    if (!submit) throw new Error('Login submit button should be available');
    submit.click();
    await nextTick();
    expect(mocks.authLogin).toHaveBeenCalledExactlyOnceWith({
      username: 'admin',
      password: '123456',
      captcha: '654321',
      captchaId: 'live-id',
    });
  });
});
