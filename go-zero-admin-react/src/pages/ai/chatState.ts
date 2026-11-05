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

// 发送通道：流式发送构造请求并持有发送锁；重查复用原请求 ID 走一元调用确认服务端结果。
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
    // 一元发送：构造请求并立即发起调用，返回可等待的 Promise。
    start(conversationId: number, content: string): MessageAttempt | null {
      if (busy || validateMessageContent(content)) return null;
      return begin({ request_id: newId(), conversation_id: conversationId, content });
    },
    // 只构造请求并加锁；帧经回调下发，结束后由调用方 finish() 释放。
    startStream(conversationId: number, content: string): AiSendMessageReq | null {
      if (busy || validateMessageContent(content)) return null;
      busy = true;
      previous = { request_id: newId(), conversation_id: conversationId, content };
      return previous;
    },
    // 重查复用原请求 ID，确认服务端是否已处理，不会新建调用。
    recheck() { return previous ? begin(previous) : null; },
    finish() { busy = false; },
  };
}
// 导出会话为 Markdown 或 JSON，触发浏览器下载。
export function exportConversation(title: string, messages: AiChatMessage[], format: 'md' | 'json'): void {
  const safeTitle = title.replace(/[\\/:*?"<>|]/g, '_') || 'ai-conversation';
  let content: string;
  let mime: string;
  if (format === 'md') {
    const lines = messages.map(m => {
      const role = m.role === 'user' ? '**我**' : m.role === 'assistant' ? '**AI 助手**' : m.role;
      const time = new Date(m.created_at * 1000).toLocaleString('zh-CN');
      const body = m.content || (m.error ? `❌ ${m.error}` : '');
      return `### ${role} · ${time}\n\n${body}\n`;
    });
    content = `# ${title}\n\n> 导出时间: ${new Date().toLocaleString('zh-CN')}\n\n${lines.join('\n')}`;
    mime = 'text/markdown;charset=utf-8';
  } else {
    content = JSON.stringify({ title, exported_at: new Date().toISOString(), messages }, null, 2);
    mime = 'application/json;charset=utf-8';
  }
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${safeTitle}.${format === 'md' ? 'md' : 'json'}`;
  a.click();
  URL.revokeObjectURL(url);
}
