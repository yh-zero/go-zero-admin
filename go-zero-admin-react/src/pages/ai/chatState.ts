import type { AiChatMessage, AiGetConversationResp, AiSendMessageReq, AiSendMessageResp } from '../../api/types';

export function validateMessageContent(content: string): string | undefined {
  if (!content.trim()) return '请输入消息内容';
  if (new TextEncoder().encode(content).length > 16000) return '消息内容最多 16000 UTF-8 字节';
}

export function mergeMessagePages(pages: Pick<AiGetConversationResp, 'messages'>[]): AiChatMessage[] {
  const messages = new Map<number, AiChatMessage>();
  for (const page of pages) {
    for (const message of page.messages) {
      // 最新页先进入；分页边界重叠时保留较新的状态与正文。
      if (!messages.has(message.id)) messages.set(message.id, message);
    }
  }
  return [...messages.values()].sort((a, b) => a.created_at - b.created_at || a.id - b.id);
}

export function nextMessageCursor(page: Pick<AiGetConversationResp, 'has_more' | 'next_before_id'>, beforeId: number): number | undefined {
  const cursor = page.next_before_id;
  if (page.has_more && Number.isSafeInteger(cursor) && cursor > 0 && (beforeId === 0 || cursor < beforeId)) return cursor;
}

export interface MessageAttempt {
  request: AiSendMessageReq;
  promise: Promise<AiSendMessageResp>;
}

export function createMessageSender(send: (request: AiSendMessageReq) => Promise<AiSendMessageResp>, newId = () => crypto.randomUUID()) {
  let busy = false;
  let previous: AiSendMessageReq | undefined;
  function begin(request: AiSendMessageReq): MessageAttempt | null {
    if (busy) return null;
    // React 下一次渲染前也要拒绝第二个事件，避免双击或 Enter 与点击同时触发。
    busy = true;
    previous = request;
    const promise = Promise.resolve().then(() => send(request)).finally(() => { busy = false; });
    return { request, promise };
  }
  return {
    start(conversationId: number, content: string) {
      if (busy || validateMessageContent(content)) return null;
      return begin({ request_id: newId(), conversation_id: conversationId, content });
    },
    recheck() { return previous ? begin(previous) : null; },
    isBusy() { return busy; },
  };
}
