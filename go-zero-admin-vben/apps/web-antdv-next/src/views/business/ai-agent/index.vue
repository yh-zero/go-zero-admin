<script setup lang="ts">
import { Page } from '@vben/common-ui';
import { useAccessStore } from '@vben/stores';

import {
  Alert,
  Button,
  Pagination,
  Space,
  Spin,
  Tag,
  TextArea,
} from 'antdv-next';

import { usePermission } from '../system/shared';
import AgentAnswer from './agent-answer.vue';
import { agentPageSize, useAgentWorkspace } from './use-agent';

defineOptions({ name: 'AiAgent' });
const access = useAccessStore();
const can = usePermission('ai-agent');
const {
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
} = useAgentWorkspace({
  getToken: () => access.accessToken ?? '',
  canRun: () => can('run'),
  canCancel: () => can('cancel'),
});
const statuses: Record<string, string> = {
  queued: '排队中',
  running: '执行中',
  succeeded: '已完成',
  failed: '失败',
  cancelled: '已停止',
  interrupted: '执行中断',
};
function statusText(status: string) {
  return statuses[status] || status;
}
function timeText(value: string) {
  const time = new Date(value);
  return Number.isNaN(time.getTime()) ? '时间未知' : time.toLocaleString();
}
</script>

<template>
  <Page>
    <div class="space-y-4">
      <Alert
        type="info"
        show-icon
        message="云模型会接收你输入的内容、当前会话历史，以及工具返回的授权查询结果。请确认这些内容适合发送至外部模型服务。模型答案可能有误，请核对引用的实际数据。"
      />
      <section class="bg-card rounded-lg border p-4" aria-label="模型服务状态">
        <Space wrap>
          <strong>AI Agent</strong>
          <Tag :color="info?.enabled && info?.configured ? 'green' : 'orange'">
            {{ info?.enabled && info?.configured ? '可使用' : '暂不可使用' }}
          </Tag>
          <span v-if="info?.provider">{{ info.provider }} · {{ info.model }}</span>
          <Button size="small" :loading="infoLoading" @click="loadInfo">
            刷新服务状态
          </Button>
        </Space>
        <Alert
          v-if="infoError"
          class="mt-3"
          type="error"
          :message="infoError"
        />
        <p v-else-if="info && !info.enabled" class="mt-3 text-sm">
          AI 服务尚未启用。管理员启用并配置云模型后即可发送；历史会话仍可查看。
        </p>
        <p v-else-if="info && !info.configured" class="mt-3 text-sm">
          云模型尚未配置完成，请联系管理员配置后刷新服务状态。
        </p>
        <p v-if="info" class="mt-3 text-sm">
          任务最多 {{ info.maxSteps }} 步，最长
          {{ info.maxRunSeconds }}
          秒。工具只读，始终受你的接口权限和数据范围限制。
        </p>
        <div v-if="info?.tools.length" class="mt-3 flex flex-wrap gap-2">
          <Tag
            v-for="tool in info.tools"
            :key="tool.name"
            :color="tool.available ? 'blue' : 'default'"
            :title="tool.description"
          >
            {{ tool.label }}{{ tool.available ? '' : '（无权限）' }}
          </Tag>
        </div>
      </section>
      <div class="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside class="bg-card rounded-lg border p-4" aria-label="历史会话">
          <Space class="mb-4" wrap>
            <Button :disabled="submitting" @click="newConversation">
              新会话
            </Button>
            <Button :loading="historyLoading" @click="loadConversations()">
              刷新历史
            </Button>
          </Space>
          <Alert
            v-if="historyError"
            class="mb-3"
            type="error"
            :message="historyError"
          />
          <Spin :spinning="historyLoading">
            <p
              v-if="!conversations.length"
              class="text-muted-foreground text-sm"
            >
              暂无会话。
            </p>
            <ul class="space-y-2">
              <li v-for="conversation in conversations" :key="conversation.id">
                <button
                  type="button"
                  class="w-full rounded border p-3 text-left hover:bg-accent"
                  :class="
                    selectedId === conversation.id
                      ? 'border-primary bg-accent'
                      : ''
                  "
                  :disabled="submitting"
                  @click="selectConversation(conversation.id)"
                >
                  <span class="block break-words">{{
                    conversation.title || '未命名会话'
                  }}</span>
                  <span class="text-muted-foreground mt-1 block text-xs">{{
                    timeText(conversation.updatedAt)
                  }}</span>
                </button>
              </li>
            </ul>
          </Spin>
          <Pagination
            v-if="conversationTotal > agentPageSize"
            class="mt-4"
            size="small"
            simple
            :current="conversationPage"
            :page-size="agentPageSize"
            :total="conversationTotal"
            :show-size-changer="false"
            @change="loadConversations"
          />
        </aside>
        <section
          class="bg-card min-w-0 space-y-4 rounded-lg border p-4"
          aria-label="对话"
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-lg font-semibold">{{ selectedTitle }}</h2>
            <Button
              v-if="selectedId"
              :loading="messagesLoading"
              :disabled="submitting || stopping"
              @click="loadMessages(latestRunKnown ? messagePage : 1)"
            >
              重新读取消息与最新任务
            </Button>
          </div>
          <Alert v-if="messagesError" type="error" :message="messagesError" />
          <p
            v-if="selectedId && !latestRunKnown && !messagesLoading"
            class="text-sm"
          >
            最新任务状态尚未确认。请重新读取消息与最新任务，再发送或停止任务。
          </p>
          <Spin :spinning="messagesLoading">
            <div
              class="max-h-[55vh] space-y-4 overflow-y-auto"
              aria-live="polite"
            >
              <p
                v-if="!messages.length"
                class="text-muted-foreground py-4 text-sm"
              >
                输入问题开始新任务，或从左侧选择历史会话。
              </p>
              <article
                v-for="item in messages"
                :key="item.id"
                class="rounded-lg border p-4"
                :class="item.role === 'user' ? 'bg-accent' : ''"
              >
                <div class="text-muted-foreground mb-2 flex gap-3 text-xs">
                  <strong>{{ item.role === 'user' ? '你' : 'AI' }}</strong><span>{{ timeText(item.createdAt) }}</span>
                </div>
                <AgentAnswer
                  v-if="item.role === 'assistant'"
                  :content="item.content"
                />
                <p v-else class="whitespace-pre-wrap break-words">
                  {{ item.content }}
                </p>
              </article>
            </div>
          </Spin>
          <Pagination
            v-if="messageTotal > agentPageSize"
            size="small"
            :current="messagePage"
            :page-size="agentPageSize"
            :total="messageTotal"
            :show-size-changer="false"
            @change="(page) => loadMessages(page)"
          />
          <section
            v-if="run"
            class="space-y-3 rounded-lg border p-3"
            aria-label="当前任务状态"
          >
            <Space wrap>
              <Tag
                :color="
                  run.status === 'succeeded'
                    ? 'green'
                    : run.status === 'failed'
                      ? 'red'
                      : 'blue'
                "
              >
                {{ statusText(run.status) }}
              </Tag>
              <span class="text-xs">{{ run.provider }} · {{ run.model }}</span>
              <span v-if="polling" class="text-xs">每 2 秒查询状态</span>
              <Button
                size="small"
                :disabled="stopping || submitting || !latestRunKnown"
                @click="refreshRun()"
              >
                重新获取状态
              </Button>
              <Button
                v-if="active && can('cancel')"
                size="small"
                danger
                :disabled="!latestRunKnown"
                :loading="stopping"
                @click="cancel"
              >
                停止任务
              </Button>
            </Space>
            <p v-if="run.error" class="whitespace-pre-wrap break-words text-sm">
              {{ run.error }}
            </p>
            <AgentAnswer
              v-if="
                run.status === 'succeeded' &&
                !messages.some(
                  (item) => item.role === 'assistant' && item.runId === run?.id,
                )
              "
              :content="run.answer"
            />
            <p v-if="run.status === 'interrupted'" class="text-sm">
              任务已中断，可重新发送问题创建新任务。
            </p>
            <ul v-if="run.toolCalls?.length" class="space-y-2 text-sm">
              <li
                v-for="(call, index) in run.toolCalls"
                :key="`${index}:${call.name}`"
                class="rounded border p-2"
              >
                <strong>{{ call.name }}</strong> · {{ statusText(call.status) }}
                <p v-if="call.summary" class="whitespace-pre-wrap break-words">
                  {{ call.summary }}
                </p>
                <p v-if="call.error" class="whitespace-pre-wrap break-words">
                  {{ call.error }}
                </p>
              </li>
            </ul>
            <p class="text-muted-foreground text-xs">
              输入 {{ run.inputTokens }} / 输出 {{ run.outputTokens }} tokens ·
              {{ timeText(run.updatedAt) }}
            </p>
          </section>
          <Alert v-if="runError" type="error" :message="runError" />
          <div class="space-y-2">
            <label class="block text-sm" for="agent-message">发送给 AI 的问题</label>
            <TextArea
              id="agent-message"
              v-model:value="draft"
              :rows="5"
              :maxlength="maxInputChars"
              :disabled="submitting || active"
              placeholder="例如：查询今天的审计日志、已知文件ID的状态，或本人登录设备。"
            />
            <div class="flex flex-wrap items-center justify-between gap-2">
              <span class="text-muted-foreground text-xs">{{ inputChars }} /
                {{ maxInputChars }} 字符。每个会话同时执行一个任务。</span>
              <Button
                v-if="can('run')"
                type="primary"
                :loading="submitting"
                :disabled="!canSubmit"
                @click="submit"
              >
                {{ retrying ? '重试提交' : '发送' }}
              </Button>
              <span v-else class="text-muted-foreground text-sm">当前角色没有执行任务权限。</span>
            </div>
          </div>
        </section>
      </div>
    </div>
  </Page>
</template>
