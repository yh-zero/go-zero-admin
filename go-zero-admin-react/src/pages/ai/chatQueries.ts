import { infiniteQueryOptions, queryOptions } from '@tanstack/react-query';
import { fetchAiConversation, fetchAiConversations } from '../../api/ai';
import { nextMessageCursor } from './chatState';

export type AiChatScope = readonly [sessionId: string, sessionVersion: number, departmentVersion: number];
export const aiChatKeys = {
  lists: (scope: AiChatScope) => ['ai-chat', ...scope, 'conversations'] as const,
  messages: (scope: AiChatScope, id: number) => ['ai-chat', ...scope, 'messages', id] as const,
};

export function conversationListOptions(scope: AiChatScope, page: number) {
  return queryOptions({
    queryKey: [...aiChatKeys.lists(scope), page],
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
