import type { AiChatMessage, AiGetConversationResp, AiSendMessageReq, AiSendMessageResult, AiStreamMeta } from '../../api/types';
import type { AiStreamHandlers } from '../../api/aiStream';

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
  promise: Promise<AiSendMessageResult>;
  controller: AbortController;
  startedAt: number;
  meta?: AiStreamMeta;
}

// 新旧会话及重查共享同一流式通道；同步锁覆盖React下一次渲染前的双击。
export function createMessageSender(send: (request: AiSendMessageReq, handlers: AiStreamHandlers, signal?: AbortSignal) => Promise<AiSendMessageResult>, newId = () => crypto.randomUUID()) {
  let active: MessageAttempt | undefined;
  let createdConversationId = 0;
  let previous: { request: AiSendMessageReq; startedAt: number; meta?: AiStreamMeta } | undefined;

  function begin(submission: NonNullable<typeof previous>, handlers: AiStreamHandlers): MessageAttempt | null {
    if (active) return null;
    previous = submission;
    const controller = new AbortController();
    const attempt: MessageAttempt = { ...submission, controller, promise: Promise.resolve().then(() => {
      controller.signal.throwIfAborted();
      return send(submission.request, { ...handlers, onMeta: meta => {
        attempt.meta = meta;
        submission.meta = meta;
        createdConversationId = meta.conversation_id;
        handlers.onMeta?.(meta);
      } }, controller.signal);
    }).then(result => {
      controller.signal.throwIfAborted();
      // 首条成功后仍在编辑的草稿属于这个新会话；原请求重查不改写原始0指纹。
      createdConversationId = result.conversation ? result.conversation_id : 0;
      return result;
    }).finally(() => { if (active === attempt) active = undefined; }) };
    active = attempt;
    return attempt;
  }

  return {
    start(conversationId: number, content: string, handlers: AiStreamHandlers = {}, knowledge: Pick<AiSendMessageReq, 'kb_id' | 'grounded'> = {}): MessageAttempt | null {
      if (active || validateMessageContent(content)) return null;
      if (knowledge.grounded && (!(knowledge.kb_id && knowledge.kb_id > 0) || new TextEncoder().encode(content).length > 2000)) return null;
      return begin({ request: { request_id: newId(), conversation_id: conversationId || createdConversationId, content, ...knowledge }, startedAt: Date.now() }, handlers);
    },
    // 重查复用原请求 ID，确认服务端是否已处理，不会新建调用。
    recheck(handlers: AiStreamHandlers = {}) { return previous ? begin(previous, handlers) : null; },
    abort() { active?.controller.abort(); },
  };
}

// 从最新页建立本次导出快照，向前读取到起点；范围超限或身份变化时整体失败，不下载残缺文件。
export async function readEntireConversation(readPage: (beforeId: number) => Promise<AiGetConversationResp>, isCurrent: () => boolean, signal: AbortSignal): Promise<AiGetConversationResp> {
  let beforeId = 0;
  let bytes = 0;
  const pages: AiGetConversationResp[] = [];
  const ensureCurrent = () => { signal.throwIfAborted(); if (!isCurrent()) throw new DOMException('身份已变化，导出已取消', 'AbortError'); };
  for (let count = 0; count < 5000; count++) {
    ensureCurrent();
    const page = await readPage(beforeId);
    ensureCurrent();
    bytes += new TextEncoder().encode(JSON.stringify(page)).length;
    if (bytes > 16 * 1024 * 1024) throw new Error('会话超过 16 MiB 导出限制，请使用已加载消息导出');
    pages.push(page);
    if (!page.has_more) return { ...pages[0]!, messages: mergeMessagePages(pages), has_more: false, next_before_id: 0 };
    const next = nextMessageCursor(page, beforeId);
    if (!next) throw new Error('消息分页游标无效，未导出不完整会话');
    beforeId = next;
  }
  throw new Error('会话超过导出页数限制，请使用已加载消息导出');
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
