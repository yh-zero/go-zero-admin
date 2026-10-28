/* eslint-disable vue/one-component-per-file -- UI stubs preserve the real page and task workspace. */
import type { App } from 'vue';

import { createApp, nextTick } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import AgentPage from './index.vue';

const mocks = vi.hoisted(() => ({
  getAgentInfo: vi.fn(),
  getAgentConversations: vi.fn(),
  getAgentMessages: vi.fn(),
  getAgentRun: vi.fn(),
  createAgentRun: vi.fn(),
  cancelAgentRun: vi.fn(),
}));
vi.mock('#/api/business/ai-agent', () => mocks);
vi.mock('@vben/stores', () => ({
  useAccessStore: () => ({ accessToken: 'test-login' }),
}));
vi.mock('../system/shared', () => ({ usePermission: () => () => true }));
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
    Spin: panel,
    Tag: panel,
    Pagination: defineComponent({
      props: { current: { type: Number, default: 1 } },
      emits: ['change'],
      setup: (props, { emit }) => () => h('button', { onClick: () => emit('change', props.current + 1) }, '下一页消息'),
    }),
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
            {
              ...attrs,
              disabled: props.disabled || props.loading,
            },
            slots.default?.(),
          ),
    }),
    TextArea: defineComponent({
      props: { value: { type: String, default: '' }, disabled: Boolean },
      emits: ['update:value'],
      setup:
        (props, { emit, attrs }) =>
        () =>
          h('textarea', {
            ...attrs,
            value: props.value,
            disabled: props.disabled,
            onInput: (event: Event) => {
              if (event.target instanceof HTMLTextAreaElement)
                emit('update:value', event.target.value);
            },
          }),
    }),
  };
});

describe('AI Agent page', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  beforeEach(() => {
    vi.resetAllMocks();
    vi.useFakeTimers();
    mocks.getAgentInfo.mockResolvedValue({
      enabled: true,
      configured: true,
      provider: 'deepseek',
      model: 'model',
      maxInputChars: 1000,
      maxSteps: 8,
      maxRunSeconds: 60,
      tools: [],
    });
    mocks.getAgentConversations.mockResolvedValue({
      items: [
        {
          id: 'one',
          title: '安全显示测试',
          createdAt: '2026-10-03T00:00:00Z',
          updatedAt: '2026-10-03T00:00:00Z',
        },
      ],
      total: 1,
    });
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    element.remove();
    vi.useRealTimers();
  });
  async function flush() {
    for (let index = 0; index < 16; index++) await Promise.resolve();
    await nextTick();
  }
  async function mount() {
    app = createApp(AgentPage);
    app.mount(element);
    await flush();
  }
  it('renders external answers and tool summaries as text without creating active HTML', async () => {
    const externalText =
      '<img src=x onerror="window.agentHtmlExecuted=true"><script>window.agentHtmlExecuted=true</script>';
    mocks.getAgentMessages.mockResolvedValue({
      items: [
        {
          id: 'answer',
          conversationId: 'one',
          runId: 'run-one',
          role: 'assistant',
          content: externalText,
          createdAt: '2026-10-03T00:00:00Z',
        },
      ],
      total: 1,
    });
    mocks.getAgentRun.mockResolvedValue({
      id: 'run-one',
      conversationId: 'one',
      requestId: 'request',
      status: 'succeeded',
      answer: externalText,
      error: '',
      provider: 'deepseek',
      model: 'model',
      inputTokens: 12,
      outputTokens: 15,
      createdAt: '2026-10-03T00:00:00Z',
      updatedAt: '2026-10-03T00:00:00Z',
      toolCalls: [
        {
          name: 'read-only',
          status: 'succeeded',
          error: '',
          summary: '<strong>外部摘要</strong>',
        },
      ],
    });
    await mount();
    [...element.querySelectorAll('button')]
      .find((button) => button.textContent?.includes('安全显示测试'))!
      .click();
    await flush();
    expect(element.textContent).toContain(externalText);
    expect(element.textContent).toContain('<strong>外部摘要</strong>');
    expect(element.querySelector('img')).toBeNull();
    expect(element.querySelector('script')).toBeNull();
    expect(element.textContent).toContain('云模型会接收你输入的内容');
  });
  it('explains missing model configuration and disables sending after the user enters input', async () => {
    mocks.getAgentInfo.mockResolvedValue({
      enabled: true,
      configured: false,
      provider: 'qwen',
      model: '',
      maxInputChars: 100,
      maxSteps: 8,
      maxRunSeconds: 60,
      tools: [],
    });
    await mount();
    const input = element.querySelector<HTMLTextAreaElement>('textarea')!;
    input.value = '一个真实问题';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();
    const send = [...element.querySelectorAll('button')].find(
      (button) => button.textContent === '发送',
    )!;
    expect(send.disabled).toBe(true);
    send.click();
    expect(mocks.createAgentRun).not.toHaveBeenCalled();
    expect(element.textContent).toContain('云模型尚未配置完成');
  });
  it('re-enables the actual status button after an acknowledged stop loses confirmation polling', async () => {
    mocks.getAgentMessages.mockResolvedValue({
      items: [{ id: 'question', conversationId: 'one', runId: 'run-one', role: 'user', content: '问题', createdAt: '2026-10-03T00:00:00Z' }],
      total: 1,
    });
    const activeRun = {
      id: 'run-one', conversationId: 'one', requestId: 'request', status: 'running',
      answer: '', error: '', provider: 'deepseek', model: 'model', inputTokens: 0, outputTokens: 0,
      createdAt: '2026-10-03T00:00:00Z', updatedAt: '2026-10-03T00:00:00Z', toolCalls: [],
    };
    mocks.getAgentRun.mockResolvedValue(activeRun);
    mocks.cancelAgentRun.mockResolvedValue(activeRun);
    await mount();
    const button = (label: string) => [...element.querySelectorAll('button')].find((entry) => entry.textContent?.includes(label))!;
    button('安全显示测试').click();
    await flush();
    button('停止任务').click();
    await flush();
    expect(button('重新获取状态').disabled).toBe(true);
    mocks.getAgentRun.mockRejectedValueOnce({ response: { data: { message: '确认状态暂时无法读取' } } });
    await vi.advanceTimersByTimeAsync(2000);
    await flush();
    expect(element.textContent).toContain('确认状态暂时无法读取');
    expect(button('重新获取状态').disabled).toBe(false);
    mocks.getAgentRun.mockResolvedValueOnce({ ...activeRun, status: 'cancelled' });
    button('重新获取状态').click();
    await flush();
    expect(element.textContent).toContain('已停止');
    expect(element.textContent).not.toContain('确认状态暂时无法读取');
  });
  it('keeps latest-message recovery clickable and returns to page one after a newer task read is denied', async () => {
    const oldRun = {
      id: 'run-one', conversationId: 'one', requestId: 'request', status: 'succeeded',
      answer: '旧答案', error: '', provider: 'deepseek', model: 'model', inputTokens: 0, outputTokens: 0,
      createdAt: '2026-10-03T00:00:00Z', updatedAt: '2026-10-03T00:00:00Z', toolCalls: [],
    };
    mocks.getAgentMessages.mockResolvedValue({ items: [{ id: 'old-answer', conversationId: 'one', runId: 'run-one', role: 'assistant', content: '旧答案', createdAt: oldRun.createdAt }], total: 21 });
    mocks.getAgentRun.mockResolvedValue(oldRun);
    await mount();
    const button = (label: string) => [...element.querySelectorAll('button')].find((entry) => entry.textContent?.includes(label))!;
    button('安全显示测试').click();
    await flush();
    mocks.getAgentMessages.mockResolvedValue({ items: [{ id: 'new-question', conversationId: 'one', runId: 'run-two', role: 'user', content: '新问题', createdAt: oldRun.createdAt }], total: 22 });
    mocks.getAgentRun.mockRejectedValueOnce({ response: { data: { message: '最新任务暂时无法读取' } } });
    button('重新读取消息').click();
    await flush();
    expect(button('重新获取状态').disabled).toBe(true);
    expect(button('重新读取消息').disabled).toBe(false);
    button('下一页消息').click();
    await flush();
    expect(mocks.getAgentMessages).toHaveBeenLastCalledWith('one', { pageNo: 2, pageSize: 20 });
    mocks.getAgentRun.mockResolvedValueOnce({ ...oldRun, id: 'run-two', status: 'queued', answer: '' });
    button('重新读取消息').click();
    await flush();
    expect(mocks.getAgentMessages).toHaveBeenLastCalledWith('one', { pageNo: 1, pageSize: 20 });
    expect(mocks.getAgentRun).toHaveBeenLastCalledWith('run-two');
    expect(element.textContent).toContain('排队中');
    expect(button('重新获取状态').disabled).toBe(false);
    expect(button('停止任务').disabled).toBe(false);
  });
});
