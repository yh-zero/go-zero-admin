import { infiniteQueryOptions, queryOptions } from '@tanstack/react-query';
import type { InfiniteData, QueryClient } from '@tanstack/react-query';
import { fetchAiConversation, fetchAiConversations } from '../../api/ai';
import type { AiConversationInfo, AiGetConversationResp, AiSendMessageReq, AiSendMessageResult, PageResult } from '../../api/types';
import { nextMessageCursor } from './chatState';

export type AiChatScope = readonly [sessionId: string, sessionVersion: number, departmentVersion: number];
export const aiChatKeys = {
  lists: (scope: AiChatScope) => ['ai-chat', ...scope, 'conversations'] as const,
  messages: (scope: AiChatScope, id: number) => ['ai-chat', ...scope, 'messages', id] as const,
};

export function conversationListOptions(scope: AiChatScope, page: number) {
  return queryOptions({
    queryKey: [...aiChatKeys.lists(scope), page],
    staleTime: 30_000,
    queryFn: async ({ signal }) => {
      signal.throwIfAborted();
      const data = await fetchAiConversations({ page, page_size: 20 });
      signal.throwIfAborted();
      return data;
    },
  });
}

export function conversationMessagesOptions(scope: AiChatScope, id: number) {
  return infiniteQueryOptions({
    queryKey: aiChatKeys.messages(scope, id),
    staleTime: 30_000,
    initialPageParam: 0,
    queryFn: async ({ signal, pageParam }) => {
      signal.throwIfAborted();
      const data = await fetchAiConversation(id, { before_id: pageParam, page_size: 20 });
      // 统一请求层负责身份取消；Query 取消还要阻止删除后的迟到详情回写缓存。
      signal.throwIfAborted();
      return data;
    },
    getNextPageParam: (page, _pages, beforeId) => nextMessageCursor(page, beforeId),
  });
}

type MessagePages = InfiniteData<AiGetConversationResp, number>;
interface ApplyOptions { request: AiSendMessageReq; isCurrent: () => boolean; startedAt?: number }
interface ListUpdateOptions {
  isCurrent: () => boolean; startedAt?: number; newConversation?: boolean; requireMembershipRead?: boolean;
}

function mergeLatestHead(old: MessagePages | undefined, head: AiGetConversationResp): MessagePages {
  const latest = { pageParams: [0], pages: [head] };
  if (!old?.pages.length) return latest;
  const knownIds = new Set(old.pages.flatMap(page => page.messages.map(message => message.id)));
  const newlyRead = head.messages.filter(message => !knownIds.has(message.id)).length;
  const growth = head.conversation.message_count - old.pages[0]!.conversation.message_count;
  // 最新页无法覆盖所有新增消息时，保留旧游标会跳过中间缺口；重新从最新页向前加载。
  if (growth < 0 || growth > newlyRead) return latest;
  const messages = new Map(old.pages[0]!.messages.map(message => [message.id, message]));
  for (const message of head.messages) messages.set(message.id, message);
  return { ...old, pages: [{ ...old.pages[0]!, conversation: head.conversation,
    messages: [...messages.values()].sort((a, b) => a.created_at - b.created_at || a.id - b.id) },
  ...old.pages.slice(1).map(page => ({ ...page, conversation: head.conversation }))] };
}

// 只将成功终帧落入缓存，分页的历史边界不随新消息移动。
export async function applyAiSendResult(client: QueryClient, scope: AiChatScope, result: AiSendMessageResult, options: ApplyOptions) {
  if (!options.isCurrent()) return;
  const queryKey = aiChatKeys.messages(scope, result.conversation_id);
  await client.cancelQueries({ queryKey, exact: true });
  if (!options.isCurrent()) return;
  let conversation = result.conversation;
  if (!conversation) {
    client.removeQueries({ queryKey, exact: true });
    // 删除可能来自另一标签页，服务端总数须重新读取，不能凭本地存在与否推断。
    await client.invalidateQueries({ queryKey: aiChatKeys.lists(scope), refetchType: 'none' });
    await client.invalidateQueries({ queryKey: [...aiChatKeys.lists(scope), 1], exact: true });
    return;
  }
  let canSeed = true;
  const cached = client.getQueryData<MessagePages>(queryKey);
  const lastKnownId = Math.max(0, ...cached?.pages.flatMap(page => page.messages.map(message => message.id)) ?? []);
  const newPairCount = [result.user_message, result.ai_message].filter(message => message.id > lastKnownId).length;
  const needsHead = cached?.pages.length
    ? conversation.message_count > cached.pages[0]!.conversation.message_count + newPairCount
    : conversation.message_count > 2;
  if (needsHead) {
    // 历史重查或其他标签页可能产生未知消息；只追加当前消息对会遗漏这些消息。
    try {
      const head = await fetchAiConversation(result.conversation_id);
      if (!options.isCurrent()) return;
      client.setQueryData<MessagePages>(queryKey, old => mergeLatestHead(old, head));
      conversation = head.conversation;
    } catch {
      if (!options.isCurrent()) return;
      // 正式详情从最新页恢复；不能将未能确认的旧历史标记成连续完整数据。
      canSeed = false;
      client.removeQueries({ queryKey, exact: true });
    }
  }
  if (!options.isCurrent()) return;
  const summary = conversation;
  client.setQueryData<MessagePages>(queryKey, old => {
    const pair = [result.user_message, result.ai_message];
    if (!old?.pages.length) {
      if (!canSeed) return old;
      return { pageParams: [0], pages: [{ conversation: summary, messages: pair,
        has_more: summary.message_count > pair.length, next_before_id: summary.message_count > pair.length ? pair[0]!.id : 0 }] };
    }
    const replacements = new Map(pair.map(message => [message.id, message]));
    const seen = new Set<number>();
    const pages = old.pages.map((page, index) => {
      const messages = [...page.messages, ...(index === 0 ? pair : [])].flatMap(message => {
        if (seen.has(message.id)) return [];
        seen.add(message.id);
        return [replacements.get(message.id) ?? message];
      }).sort((a, b) => a.created_at - b.created_at || a.id - b.id);
      return { ...page, conversation: summary, messages };
    });
    return { ...old, pages };
  });
  await updateConversationLists(client, scope, summary, { ...options, newConversation: options.request.conversation_id === 0 });
}

async function updateConversationLists(client: QueryClient, scope: AiChatScope, conversation: AiConversationInfo, options: ListUpdateOptions) {
  const prefix = aiChatKeys.lists(scope);
  await client.cancelQueries({ queryKey: prefix });
  if (!options.isCurrent()) return;
  const entries = client.getQueryCache().findAll({ queryKey: prefix }).flatMap(query => {
    const page = query.queryKey[prefix.length];
    const data = query.state.data as PageResult<AiConversationInfo> | undefined;
    return typeof page === 'number' && data ? [{ page, data, queryKey: query.queryKey, updatedAt: query.state.dataUpdatedAt }] : [];
  }).sort((a, b) => a.page - b.page);
  const contiguous = entries.filter((entry, index) => entry.page === index + 1);
  const first = contiguous[0];
  const known = entries.some(entry => entry.data.list.some(row => row.id === conversation.id));
  const complete = first && contiguous.reduce((count, entry) => count + entry.data.list.length, 0) === first.data.total;
  const createdAfterSnapshot = options.startedAt !== undefined && first && first.updatedAt <= options.startedAt;
  if (!first || (!known && (options.requireMembershipRead || (options.newConversation && !complete && !createdAfterSnapshot)))) {
    // 重查前列表可能已包含未确认的新会话；仅刷新第一页确认总数，不猜测 +1。
    await client.invalidateQueries({ queryKey: prefix, refetchType: 'none' });
    await client.invalidateQueries({ queryKey: [...prefix, 1], exact: true });
    return;
  }
  const total = first.data.total + (!known && options.newConversation ? 1 : 0);
  const byId = new Map(contiguous.flatMap(entry => entry.data.list).map(row => [row.id, row]));
  byId.set(conversation.id, conversation);
  const rows = [...byId.values()].sort((a, b) => b.updated_at - a.updated_at || b.id - a.id);
  const pages = complete ? Math.max(contiguous.length, Math.ceil(rows.length / 20)) : contiguous.length;
  for (let index = 0; index < pages; index++) {
    client.setQueryData<PageResult<AiConversationInfo>>([...prefix, index + 1], { list: rows.slice(index * 20, (index + 1) * 20), total });
  }
  // 不连续的页无法推断被挤出的边界，保留旧快照并在下次进入时读取。
  await Promise.all(entries.filter(entry => entry.page > pages).map(entry => client.invalidateQueries({ queryKey: entry.queryKey, exact: true, refetchType: 'none' })));
}

// 未知结果只读最新一页，避免refetch无限查询重读全部已加载历史。
export async function recoverAiSend(client: QueryClient, scope: AiChatScope, id: number, isCurrent: () => boolean) {
  if (!id || !isCurrent()) return;
  try {
    await client.cancelQueries({ queryKey: aiChatKeys.messages(scope, id), exact: true });
    if (!isCurrent()) return;
    const head = await fetchAiConversation(id);
    if (!isCurrent()) return;
    client.setQueryData<MessagePages>(aiChatKeys.messages(scope, id), old => mergeLatestHead(old, head));
    // 失败也已持久化消息对；摘要来自刚读取的head，不再额外重读已知会话列表。
    await updateConversationLists(client, scope, head.conversation, { isCurrent, requireMembershipRead: true });
  } catch { /* 重查入口及原始request_id仍保留，由用户决定下一步。 */ }
}
