import type { App } from 'vue';

import type {
  AgentInfo,
  AgentMessage,
  AgentRun,
  CreateAgentRun,
} from '#/api/business/ai-agent';

import { createApp, h, nextTick, ref } from 'vue';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useAgentWorkspace } from './use-agent';

const mocks = vi.hoisted(() => ({
  getAgentInfo: vi.fn(),
  getAgentConversations: vi.fn(),
  getAgentMessages: vi.fn(),
  getAgentRun: vi.fn(),
  createAgentRun: vi.fn(),
  cancelAgentRun: vi.fn(),
}));
vi.mock('#/api/business/ai-agent', () => mocks);

const info: AgentInfo = {
  enabled: true,
  configured: true,
  provider: 'deepseek',
  model: 'configured-model',
  maxInputChars: 100,
  maxSteps: 8,
  maxRunSeconds: 60,
  tools: [],
};
function task(
  id = 'run-one',
  conversationId = 'one',
  status: AgentRun['status'] = 'running',
  requestId = 'request',
): AgentRun {
  return {
    id,
    conversationId,
    requestId,
    status,
    answer: '',
    error: '',
    provider: 'deepseek',
    model: 'configured-model',
    inputTokens: 0,
    outputTokens: 0,
    createdAt: '2026-10-03T00:00:00Z',
    updatedAt: '2026-10-03T00:00:00Z',
    toolCalls: [],
  };
}
function msg(
  id: string,
  conversationId = 'one',
  runId = 'run-one',
  role: AgentMessage['role'] = 'user',
): AgentMessage {
  return {
    id,
    conversationId,
    runId,
    role,
    content: id,
    createdAt: '2026-10-03T00:00:00Z',
  };
}
function deferred<T>() {
  let resolve!: (result: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((done, fail) => {
    resolve = done;
    reject = fail;
  });
  return { promise, resolve, reject };
}

describe('AI Agent task lifecycle', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  const token = ref('old-login');
  const permissions = ref({ run: true, cancel: true });
  let state: ReturnType<typeof useAgentWorkspace>;
  beforeEach(() => {
    vi.resetAllMocks();
    vi.useFakeTimers();
    token.value = 'old-login';
    permissions.value = { run: true, cancel: true };
    mocks.getAgentInfo.mockResolvedValue(info);
    mocks.getAgentConversations.mockResolvedValue({ items: [], total: 0 });
    mocks.getAgentMessages.mockResolvedValue({ items: [], total: 0 });
    mocks.getAgentRun.mockResolvedValue(task());
    mocks.createAgentRun.mockImplementation(async (data: CreateAgentRun) =>
      task('new-run', data.conversationId || 'new', 'queued', data.requestId),
    );
    mocks.cancelAgentRun.mockResolvedValue(task('run-one', 'one', 'cancelled'));
    element = document.createElement('div');
    document.body.append(element);
  });
  afterEach(() => {
    app?.unmount();
    app = undefined;
    element.remove();
    vi.useRealTimers();
  });
  async function flush() {
    for (let index = 0; index < 12; index++) await Promise.resolve();
    await nextTick();
  }
  async function mount() {
    app = createApp({
      setup() {
        state = useAgentWorkspace({
          getToken: () => token.value,
          canRun: () => permissions.value.run,
          canCancel: () => permissions.value.cancel,
        });
        return () => h('div');
      },
    });
    app.mount(element);
    await flush();
    return state;
  }
  async function openActive() {
    mocks.getAgentMessages.mockResolvedValue({
      items: [msg('question')],
      total: 1,
    });
    const view = await mount();
    await view.selectConversation('one');
    return view;
  }
  it('blocks new runs when the service is unconfigured or run permission is missing', async () => {
    mocks.getAgentInfo.mockResolvedValue({ ...info, configured: false });
    const view = await mount();
    view.draft.value = '问题';
    await view.submit();
    expect(mocks.createAgentRun).not.toHaveBeenCalled();
    mocks.getAgentInfo.mockResolvedValue(info);
    await view.loadInfo();
    permissions.value.run = false;
    await view.submit();
    expect(mocks.createAgentRun).not.toHaveBeenCalled();
  });
  it('coalesces double submit and reuses the same requestId after an ambiguous transport failure', async () => {
    const pending = deferred<AgentRun>();
    mocks.createAgentRun.mockReturnValueOnce(pending.promise);
    const view = await mount();
    view.draft.value = '重复点击同一个问题';
    const first = view.submit();
    await view.submit();
    expect(mocks.createAgentRun).toHaveBeenCalledTimes(1);
    const firstBody = mocks.createAgentRun.mock.calls[0]![0];
    pending.reject(new Error('传输中断'));
    await first;
    expect(view.draft.value).toBe('重复点击同一个问题');
    expect(view.retrying.value).toBe(true);
    await view.submit();
    expect(mocks.createAgentRun.mock.calls[1]![0]).toEqual(firstBody);
    expect(view.selectedId.value).toBe('new');
    expect(view.run.value?.status).toBe('queued');
    expect(view.draft.value).toBe('');
  });
  it('allocates a fresh requestId after editing a failed submission', async () => {
    mocks.createAgentRun.mockRejectedValueOnce(new Error('服务暂不可用'));
    const view = await mount();
    view.draft.value = '第一个问题';
    await view.submit();
    const original = mocks.createAgentRun.mock.calls[0]![0].requestId;
    view.draft.value = '修改后的问题';
    await view.submit();
    expect(mocks.createAgentRun.mock.calls[1]![0].requestId).not.toBe(original);
  });
  it('keeps the actual business rejection message and the same requestId for retry', async () => {
    mocks.createAgentRun.mockRejectedValueOnce({
      response: { data: { code: 100002, message: '已有任务执行中，请等待或停止后重试' } },
    });
    const view = await mount();
    view.draft.value = '一个真实问题';
    await view.submit();
    expect(view.runError.value).toBe('已有任务执行中，请等待或停止后重试');
    expect(view.draft.value).toBe('一个真实问题');
    const requestId = mocks.createAgentRun.mock.calls[0]![0].requestId;
    await view.submit();
    expect(mocks.createAgentRun.mock.calls[1]![0].requestId).toBe(requestId);
  });
  it.each(['pending', 'accepted'] as const)(
    'ignores an old terminal run read when a new submission is %s',
    async (delivery) => {
      mocks.getAgentMessages.mockResolvedValue({
        items: [msg('old-answer', 'one', 'run-one', 'assistant')],
        total: 1,
      });
      mocks.getAgentRun.mockResolvedValue(task('run-one', 'one', 'succeeded'));
      const view = await mount();
      await view.selectConversation('one');
      const oldRead = deferred<AgentRun>();
      mocks.getAgentRun.mockReturnValueOnce(oldRead.promise);
      const refresh = view.refreshRun();
      const pending = deferred<AgentRun>();
      mocks.createAgentRun.mockReturnValueOnce(pending.promise);
      view.draft.value = '同会话新问题';
      const submission = view.submit();
      const requestId = mocks.createAgentRun.mock.calls[0]![0].requestId;
      if (delivery === 'pending') {
        oldRead.resolve(task('run-one', 'one', 'failed'));
        await refresh;
        expect(view.run.value?.status).toBe('succeeded');
        expect(view.submitting.value).toBe(true);
      }
      pending.resolve(task('new-run', 'one', 'queued', requestId));
      await submission;
      if (delivery === 'accepted') {
        oldRead.resolve(task('run-one', 'one', 'failed'));
        await refresh;
      }
      expect(view.run.value?.id).toBe('new-run');
      expect(view.run.value?.status).toBe('queued');
      expect(view.active.value).toBe(true);
      expect(view.polling.value).toBe(true);
      view.draft.value = '不能重复发送';
      expect(view.canSubmit.value).toBe(false);
      mocks.getAgentRun.mockResolvedValue(task('new-run', 'one', 'running'));
      await vi.advanceTimersByTimeAsync(2000);
      expect(mocks.getAgentRun).toHaveBeenLastCalledWith('new-run');
      expect(view.run.value?.id).toBe('new-run');
    },
  );
  it('reverses the current history page, restores the latest task and stops polling at a terminal state', async () => {
    mocks.getAgentMessages.mockResolvedValue({
      items: [msg('answer', 'one', 'run-one', 'assistant'), msg('question')],
      total: 21,
    });
    const view = await mount();
    await view.selectConversation('one');
    expect(view.messages.value.map((item) => item.id)).toEqual([
      'question',
      'answer',
    ]);
    expect(view.active.value).toBe(true);
    view.draft.value = '不能并发发送';
    await view.submit();
    expect(mocks.createAgentRun).not.toHaveBeenCalled();
    mocks.getAgentRun.mockResolvedValue(task('run-one', 'one', 'interrupted'));
    await vi.advanceTimersByTimeAsync(2000);
    expect(view.run.value?.status).toBe('interrupted');
    expect(view.polling.value).toBe(false);
    const reads = mocks.getAgentRun.mock.calls.length;
    await vi.advanceTimersByTimeAsync(10_000);
    expect(mocks.getAgentRun).toHaveBeenCalledTimes(reads);
  });
  it('pauses failed polling without inventing a failed task, then permits an explicit retry', async () => {
    const view = await openActive();
    mocks.getAgentRun.mockRejectedValueOnce(new Error('状态查询失败'));
    await vi.advanceTimersByTimeAsync(2000);
    expect(view.run.value?.status).toBe('running');
    expect(view.runError.value).toBe('状态查询失败');
    expect(view.polling.value).toBe(false);
    const reads = mocks.getAgentRun.mock.calls.length;
    await vi.advanceTimersByTimeAsync(10_000);
    expect(mocks.getAgentRun).toHaveBeenCalledTimes(reads);
    mocks.getAgentRun.mockResolvedValue(task('run-one', 'one', 'failed'));
    await view.refreshRun();
    expect(view.run.value?.status).toBe('failed');
  });
  it('drops a pending old poll after changing conversations', async () => {
    const view = await openActive();
    const old = deferred<AgentRun>();
    mocks.getAgentRun.mockReturnValueOnce(old.promise);
    await vi.advanceTimersByTimeAsync(2000);
    mocks.getAgentMessages.mockResolvedValue({
      items: [msg('other', 'two', 'run-two')],
      total: 1,
    });
    mocks.getAgentRun.mockResolvedValue(task('run-two', 'two', 'succeeded'));
    await view.selectConversation('two');
    old.resolve(task('run-one', 'one', 'failed'));
    await flush();
    expect(view.run.value?.id).toBe('run-two');
    expect(view.messages.value[0]?.conversationId).toBe('two');
    expect(view.polling.value).toBe(false);
  });
  it('discards a submission response from an older login', async () => {
    const pending = deferred<AgentRun>();
    mocks.createAgentRun.mockReturnValueOnce(pending.promise);
    const view = await mount();
    view.draft.value = '旧登录问题';
    const request = view.submit();
    const requestId = mocks.createAgentRun.mock.calls[0]![0].requestId;
    token.value = 'new-login';
    await flush();
    pending.resolve(task('old-run', 'old-conversation', 'queued', requestId));
    await request;
    expect(view.selectedId.value).toBe('');
    expect(view.run.value).toBeUndefined();
    expect(view.draft.value).toBe('');
    expect(mocks.getAgentMessages).not.toHaveBeenCalled();
  });
  it('keeps stop idempotent and ignores a running poll returned after confirmed cancellation', async () => {
    const view = await openActive();
    const poll = deferred<AgentRun>();
    mocks.getAgentRun.mockReturnValueOnce(poll.promise);
    await vi.advanceTimersByTimeAsync(2000);
    const cancellation = deferred<AgentRun>();
    mocks.cancelAgentRun.mockReturnValueOnce(cancellation.promise);
    const stop = view.cancel();
    await view.cancel();
    expect(mocks.cancelAgentRun).toHaveBeenCalledTimes(1);
    mocks.getAgentRun.mockResolvedValue(task('run-one', 'one', 'cancelled'));
    cancellation.resolve(task('run-one', 'one', 'cancelled'));
    await stop;
    poll.resolve(task());
    await flush();
    expect(view.run.value?.status).toBe('cancelled');
    expect(view.stopping.value).toBe(false);
    await view.cancel();
    expect(mocks.cancelAgentRun).toHaveBeenCalledTimes(1);
  });
  it.each(['refresh', 'cancel'] as const)('allows %s after stop is acknowledged but a later confirmation poll is rejected', async (recovery) => {
    const view = await openActive();
    mocks.getAgentRun.mockResolvedValueOnce(task());
    await view.cancel();
    expect(view.stopping.value).toBe(true);
    expect(view.polling.value).toBe(true);
    mocks.getAgentRun.mockRejectedValueOnce({
      response: { data: { code: 100002, message: '任务状态暂时无法读取' } },
    });
    await vi.advanceTimersByTimeAsync(2000);
    expect(view.stopping.value).toBe(false);
    expect(view.run.value?.status).toBe('running');
    expect(view.runError.value).toBe('任务状态暂时无法读取');
    expect(view.polling.value).toBe(false);
    mocks.getAgentRun.mockResolvedValueOnce(task('run-one', 'one', 'cancelled'));
    if (recovery === 'refresh') await view.refreshRun();
    else await view.cancel();
    expect(mocks.cancelAgentRun).toHaveBeenCalledTimes(recovery === 'refresh' ? 1 : 2);
    expect(view.run.value?.status).toBe('cancelled');
    expect(view.stopping.value).toBe(false);
  });
  it('does not let an older history page bypass an unknown latest run', async () => {
    mocks.getAgentMessages.mockResolvedValue({
      items: [msg('question')],
      total: 21,
    });
    mocks.getAgentRun.mockRejectedValue(new Error('任务读取被拒绝'));
    const view = await mount();
    await view.selectConversation('one');
    view.draft.value = '新问题';
    await view.loadMessages(2);
    await view.submit();
    expect(mocks.createAgentRun).not.toHaveBeenCalled();
    expect(mocks.getAgentMessages).toHaveBeenLastCalledWith('one', {
      pageNo: 2,
      pageSize: 20,
    });
  });
  it('invalidates an old status read and requires latest-page recovery after the newer run query is denied', async () => {
    mocks.getAgentMessages.mockResolvedValue({ items: [msg('old-answer', 'one', 'run-one', 'assistant')], total: 21 });
    mocks.getAgentRun.mockResolvedValue(task('run-one', 'one', 'succeeded'));
    const view = await mount();
    await view.selectConversation('one');
    const oldRead = deferred<AgentRun>();
    mocks.getAgentRun.mockReturnValueOnce(oldRead.promise);
    const refresh = view.refreshRun();
    const latestRead = deferred<AgentRun>();
    mocks.getAgentMessages.mockResolvedValue({ items: [msg('new-question', 'one', 'run-two')], total: 22 });
    mocks.getAgentRun.mockReturnValueOnce(latestRead.promise);
    const recovery = view.loadMessages(1);
    await flush();
    oldRead.resolve(task('run-one', 'one', 'succeeded'));
    await refresh;
    expect(view.latestRunKnown.value).toBe(false);
    const calls = mocks.getAgentRun.mock.calls.length;
    await view.refreshRun();
    expect(mocks.getAgentRun).toHaveBeenCalledTimes(calls);
    latestRead.reject({ response: { data: { message: '最新任务暂时无法读取' } } });
    await recovery;
    await view.loadMessages(2);
    view.draft.value = '新问题';
    expect(view.canSubmit.value).toBe(false);
    await view.refreshRun();
    expect(mocks.getAgentRun).toHaveBeenCalledTimes(calls);
    mocks.getAgentRun.mockResolvedValue(task('run-two', 'one', 'queued'));
    await view.loadMessages(1);
    expect(view.run.value?.id).toBe('run-two');
    expect(view.latestRunKnown.value).toBe(true);
    expect(view.polling.value).toBe(true);
    expect(view.canSubmit.value).toBe(false);
  });
  it('drops unfinished polls on unmount and does not schedule further requests', async () => {
    const view = await openActive();
    const pending = deferred<AgentRun>();
    mocks.getAgentRun.mockReturnValueOnce(pending.promise);
    await vi.advanceTimersByTimeAsync(2000);
    app?.unmount();
    app = undefined;
    pending.resolve(task('run-one', 'one', 'succeeded'));
    await flush();
    expect(view.run.value?.status).toBe('running');
    const reads = mocks.getAgentRun.mock.calls.length;
    await vi.advanceTimersByTimeAsync(10_000);
    expect(mocks.getAgentRun).toHaveBeenCalledTimes(reads);
  });
});
