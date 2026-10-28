import type {
  AgentConversation,
  AgentInfo,
  AgentMessage,
  AgentRun,
  CreateAgentRun,
} from '#/api/business/ai-agent';

import { computed, onMounted, onUnmounted, ref, watch } from 'vue';

import {
  cancelAgentRun,
  createAgentRun,
  getAgentConversations,
  getAgentInfo,
  getAgentMessages,
  getAgentRun,
} from '#/api/business/ai-agent';

export const agentPageSize = 20;
const pollMilliseconds = 2000;
export const isAgentActive = (run?: AgentRun) =>
  run?.status === 'queued' || run?.status === 'running';
export function agentError(failure: unknown, fallback: string) {
  if (failure && typeof failure === 'object' && 'response' in failure) {
    const response = failure.response;
    if (response && typeof response === 'object' && 'data' in response) {
      const data = response.data;
      if (
        data &&
        typeof data === 'object' &&
        'message' in data &&
        typeof data.message === 'string' &&
        data.message
      )
        return data.message;
    }
  }
  if (failure instanceof Error && failure.message) return failure.message;
  if (
    failure &&
    typeof failure === 'object' &&
    'message' in failure &&
    typeof failure.message === 'string' &&
    failure.message
  )
    return failure.message;
  return fallback;
}

interface AgentSession {
  getToken: () => string;
  canRun: () => boolean;
  canCancel: () => boolean;
}

export function useAgentWorkspace(session: AgentSession) {
  const info = ref<AgentInfo>();
  const infoLoading = ref(false);
  const infoError = ref('');
  const conversations = ref<AgentConversation[]>([]);
  const conversationPage = ref(1);
  const conversationTotal = ref(0);
  const historyLoading = ref(false);
  const historyError = ref('');
  const selectedId = ref('');
  const messages = ref<AgentMessage[]>([]);
  const messagePage = ref(1);
  const messageTotal = ref(0);
  const messagesLoading = ref(false);
  const messagesReady = ref(true);
  const latestRunKnown = ref(true);
  const messagesError = ref('');
  const run = ref<AgentRun>();
  const runError = ref('');
  const draft = ref('');
  const submitting = ref(false);
  const stopping = ref(false);
  const polling = ref(false);
  const attempt = ref<CreateAgentRun>();
  let epoch = 0;
  let view = 0;
  let infoVersion = 0;
  let historyVersion = 0;
  let messageVersion = 0;
  let runVersion = 0;
  let disposed = false;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const active = computed(() => isAgentActive(run.value));
  const maxInputChars = computed(() =>
    Math.max(1, info.value?.maxInputChars || 4000),
  );
  const inputChars = computed(() => [...draft.value.trim()].length);
  const canSubmit = computed(
    () =>
      !!session.getToken() &&
      session.canRun() &&
      !!info.value?.enabled &&
      !!info.value.configured &&
      !infoLoading.value &&
      !infoError.value &&
      !submitting.value &&
      !active.value &&
      !messagesLoading.value &&
      messagesReady.value &&
      latestRunKnown.value &&
      inputChars.value > 0 &&
      inputChars.value <= maxInputChars.value,
  );
  const retrying = computed(
    () =>
      !!attempt.value &&
      attempt.value.message === draft.value.trim() &&
      (attempt.value.conversationId ?? '') === selectedId.value,
  );
  const selectedTitle = computed(
    () =>
      conversations.value.find((item) => item.id === selectedId.value)?.title ||
      (selectedId.value ? '历史会话' : '新会话'),
  );

  function current(
    token: string,
    expectedEpoch: number,
    expectedView?: number,
  ) {
    return (
      !disposed &&
      token === session.getToken() &&
      expectedEpoch === epoch &&
      (expectedView === undefined || expectedView === view)
    );
  }
  function stopPolling() {
    if (timer !== undefined) clearTimeout(timer);
    timer = undefined;
    polling.value = false;
  }
  function schedulePoll() {
    stopPolling();
    if (!isAgentActive(run.value) || disposed) return;
    const id = run.value!.id;
    const expectedView = view;
    polling.value = true;
    timer = setTimeout(() => {
      timer = undefined;
      if (!disposed && expectedView === view) void refreshRun(id, expectedView);
    }, pollMilliseconds);
  }
  async function loadInfo() {
    if (!session.getToken() || disposed) return;
    const token = session.getToken();
    const expectedEpoch = epoch;
    const version = ++infoVersion;
    infoLoading.value = true;
    infoError.value = '';
    try {
      const result = await getAgentInfo();
      if (current(token, expectedEpoch) && version === infoVersion)
        info.value = result;
    } catch (failure) {
      if (current(token, expectedEpoch) && version === infoVersion)
        infoError.value = agentError(failure, 'AI 服务状态读取失败，请重试');
    } finally {
      if (current(token, expectedEpoch) && version === infoVersion)
        infoLoading.value = false;
    }
  }
  async function loadConversations(page = conversationPage.value) {
    if (!session.getToken() || disposed) return;
    const token = session.getToken();
    const expectedEpoch = epoch;
    const version = ++historyVersion;
    conversationPage.value = page;
    historyLoading.value = true;
    historyError.value = '';
    try {
      const result = await getAgentConversations({
        pageNo: page,
        pageSize: agentPageSize,
      });
      if (!current(token, expectedEpoch) || version !== historyVersion) return;
      conversations.value = result.items;
      conversationTotal.value = result.total;
    } catch (failure) {
      if (current(token, expectedEpoch) && version === historyVersion)
        historyError.value = agentError(failure, '会话历史读取失败，请重试');
    } finally {
      if (current(token, expectedEpoch) && version === historyVersion)
        historyLoading.value = false;
    }
  }
  async function loadMessages(page = messagePage.value, recoverRun = true) {
    if (
      !selectedId.value ||
      !session.getToken() ||
      disposed ||
      (stopping.value && recoverRun)
    )
      return;
    const id = selectedId.value;
    const token = session.getToken();
    const expectedEpoch = epoch;
    const expectedView = view;
    const version = ++messageVersion;
    messagePage.value = page;
    messagesLoading.value = true;
    messagesError.value = '';
    messagesReady.value = false;
    if (recoverRun && page === 1) {
      latestRunKnown.value = false;
      runVersion++;
      stopPolling();
    }
    try {
      const result = await getAgentMessages(id, {
        pageNo: page,
        pageSize: agentPageSize,
      });
      if (
        !current(token, expectedEpoch, expectedView) ||
        version !== messageVersion
      )
        return;
      if (result.items.some((item) => item.conversationId !== id))
        throw new Error('历史消息响应不属于当前会话，请重新读取');
      messages.value = [...result.items].reverse();
      messageTotal.value = result.total;
      if (recoverRun && page === 1) {
        const latestRunId = result.items.find((item) => item.runId)?.runId;
        if (latestRunId) {
          const latestVersion = ++runVersion;
          const latest = await getAgentRun(latestRunId);
          if (
            !current(token, expectedEpoch, expectedView) ||
            version !== messageVersion ||
            latestVersion !== runVersion
          )
            return;
          if (latest.conversationId !== id || latest.id !== latestRunId)
            throw new Error('任务响应不属于当前会话');
          run.value = latest;
          latestRunKnown.value = true;
          runError.value = '';
          schedulePoll();
        } else {
          run.value = undefined;
          latestRunKnown.value = true;
        }
      }
      messagesReady.value = true;
    } catch (failure) {
      if (
        current(token, expectedEpoch, expectedView) &&
        version === messageVersion
      )
        messagesError.value = agentError(
          failure,
          '历史消息或任务状态读取失败，发送已禁用',
        );
    } finally {
      if (
        current(token, expectedEpoch, expectedView) &&
        version === messageVersion
      )
        messagesLoading.value = false;
    }
  }
  async function refreshRun(id = run.value?.id, expectedView = view) {
    if (!id || !session.getToken() || disposed || !latestRunKnown.value) return;
    const token = session.getToken();
    const expectedEpoch = epoch;
    const version = ++runVersion;
    stopPolling();
    runError.value = '';
    try {
      const result = await getAgentRun(id);
      if (
        !current(token, expectedEpoch, expectedView) ||
        version !== runVersion
      )
        return;
      if (result.conversationId !== selectedId.value || result.id !== id)
        throw new Error('任务响应不属于当前会话');
      run.value = result;
      latestRunKnown.value = true;
      if (isAgentActive(result)) {
        schedulePoll();
      } else {
        stopping.value = false;
        await Promise.allSettled([
          loadConversations(),
          messagePage.value === 1 ? loadMessages(1, false) : Promise.resolve(),
        ]);
      }
    } catch (failure) {
      if (current(token, expectedEpoch, expectedView) && version === runVersion) {
        stopping.value = false;
        runError.value = agentError(
          failure,
          '任务状态读取失败，自动查询已暂停，请重新获取',
        );
      }
    }
  }
  function resetConversation(id: string) {
    view++;
    messageVersion++;
    runVersion++;
    stopPolling();
    selectedId.value = id;
    messages.value = [];
    messageTotal.value = 0;
    messagePage.value = 1;
    messagesLoading.value = false;
    messagesReady.value = !id;
    latestRunKnown.value = !id;
    messagesError.value = '';
    run.value = undefined;
    runError.value = '';
    draft.value = '';
    attempt.value = undefined;
    stopping.value = false;
  }
  async function selectConversation(id: string) {
    if (submitting.value || disposed || id === selectedId.value) return;
    resetConversation(id);
    await loadMessages(1);
  }
  function newConversation() {
    if (!submitting.value && !disposed) resetConversation('');
  }
  async function submit() {
    if (!canSubmit.value || disposed) return;
    const token = session.getToken();
    const expectedEpoch = epoch;
    const expectedView = view;
    const message = draft.value.trim();
    if (
      !attempt.value ||
      attempt.value.message !== message ||
      (attempt.value.conversationId ?? '') !== selectedId.value
    ) {
      attempt.value = {
        ...(selectedId.value ? { conversationId: selectedId.value } : {}),
        requestId: crypto.randomUUID(),
        message,
      };
    }
    const data = { ...attempt.value };
    // Invalidate an older terminal run read before sending a new task.
    // A transport failure must not let that read change this submission view.
    runVersion++;
    submitting.value = true;
    runError.value = '';
    try {
      const result = await createAgentRun(data);
      if (!current(token, expectedEpoch, expectedView)) return;
      if (
        !result.id ||
        !result.conversationId ||
        result.requestId !== data.requestId ||
        (data.conversationId && result.conversationId !== data.conversationId)
      )
        throw new Error('任务提交响应不匹配，请使用原请求重试');
      runVersion++;
      selectedId.value = result.conversationId;
      run.value = result;
      latestRunKnown.value = true;
      draft.value = '';
      attempt.value = undefined;
      messagePage.value = 1;
      await Promise.allSettled([loadConversations(1), loadMessages(1, false)]);
      if (current(token, expectedEpoch, expectedView)) schedulePoll();
    } catch (failure) {
      if (current(token, expectedEpoch, expectedView))
        runError.value = agentError(
          failure,
          '提交失败，保留原输入和请求编号，可重试',
        );
    } finally {
      if (current(token, expectedEpoch, expectedView)) submitting.value = false;
    }
  }
  async function cancel() {
    if (
      !run.value ||
      !active.value ||
      stopping.value ||
      !session.canCancel() ||
      !session.getToken() ||
      !latestRunKnown.value ||
      disposed
    )
      return;
    const id = run.value.id;
    const token = session.getToken();
    const expectedEpoch = epoch;
    const expectedView = view;
    stopping.value = true;
    runVersion++;
    stopPolling();
    runError.value = '';
    let acknowledged = false;
    try {
      await cancelAgentRun(id);
      if (!current(token, expectedEpoch, expectedView)) return;
      acknowledged = true;
      await refreshRun(id, expectedView);
    } catch (failure) {
      if (current(token, expectedEpoch, expectedView)) {
        runError.value = agentError(
          failure,
          '停止请求未完成，可重试或重新获取任务状态',
        );
      }
    } finally {
      if (
        current(token, expectedEpoch, expectedView) &&
        (!acknowledged || runError.value || !active.value)
      )
        stopping.value = false;
    }
  }
  async function initialize() {
    if (!session.getToken() || disposed) return;
    await Promise.allSettled([loadInfo(), loadConversations(1)]);
  }
  watch(
    session.getToken,
    () => {
      epoch++;
      infoVersion++;
      historyVersion++;
      resetConversation('');
      info.value = undefined;
      infoError.value = '';
      infoLoading.value = false;
      conversations.value = [];
      conversationTotal.value = 0;
      historyLoading.value = false;
      historyError.value = '';
      submitting.value = false;
      void initialize();
    },
    { flush: 'sync' },
  );
  onMounted(() => {
    void initialize();
  });
  onUnmounted(() => {
    disposed = true;
    epoch++;
    view++;
    stopPolling();
  });
  return {
    active,
    canSubmit,
    cancel,
    conversationPage,
    conversations,
    conversationTotal,
    draft,
    historyError,
    historyLoading,
    info,
    infoError,
    infoLoading,
    inputChars,
    latestRunKnown,
    loadConversations,
    loadInfo,
    loadMessages,
    maxInputChars,
    messagePage,
    messages,
    messagesError,
    messagesLoading,
    messagesReady,
    messageTotal,
    newConversation,
    polling,
    refreshRun,
    retrying,
    run,
    runError,
    selectedId,
    selectedTitle,
    selectConversation,
    stopping,
    submit,
    submitting,
  };
}
