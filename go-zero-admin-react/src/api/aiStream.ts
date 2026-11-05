import { ApiError } from '../utils/response';
import { prepareStreamHeaders } from '../utils/request';
import { requiredWritePermission } from '../utils/permissions';
import { useAuthStore } from '../stores/auth';
import type { AiChatMessage, AiConversationInfo, AiSendMessageReq, AiSendMessageResult, AiStreamDone, AiStreamMeta } from './types';

export interface AiStreamHandlers {
  onMeta?: (meta: AiStreamMeta) => void;
  onDelta?: (content: string) => void;
  onTool?: (name: string, status: 'start' | 'done') => void;
  onDone?: (done: AiStreamDone) => void;
}

const streamUrl = '/sys/ai/messages/stream';
const protocolError = () => new ApiError(500, '流式响应不完整或标识不一致，请重查本次结果');

// 所有发送与结果重查都走同一SSE入口；只有完整终帧可作为已保存结果。
export async function sendAiMessageStream(data: AiSendMessageReq, handlers: AiStreamHandlers, signal?: AbortSignal): Promise<AiSendMessageResult> {
  const initial = useAuthStore.getState();
  const required = requiredWritePermission('POST', streamUrl);
  if (required && (!initial.permissionsReady || !initial.permCodes.includes(required))) {
    throw new ApiError(403, '操作权限未加载或已变化，请重新加载权限后重试');
  }
  const controller = new AbortController();
  function ensureCurrent() {
    signal?.throwIfAborted();
    const current = useAuthStore.getState();
    if (current.sessionId !== initial.sessionId || current.sessionVersion !== initial.sessionVersion
      || current.departmentVersion !== initial.departmentVersion || current.isLoggingOut
      || !current.permissionsReady || (required && !current.permCodes.includes(required))) {
      throw new DOMException('身份或权限已变化', 'AbortError');
    }
    controller.signal.throwIfAborted();
  }
  const unsubscribe = useAuthStore.subscribe(() => {
    try { ensureCurrent(); } catch (error) { controller.abort(error); }
  });
  const combined = signal ? AbortSignal.any([signal, controller.signal]) : controller.signal;
  const timeout = setTimeout(() => controller.abort(new DOMException('AI 请求超时，请重查本次结果', 'TimeoutError')), 130_000);
  let reader: ReadableStreamDefaultReader<string> | undefined;
  const cancelReader = () => { void reader?.cancel().catch(() => undefined); };
  combined.addEventListener('abort', cancelReader, { once: true });
  try {
    const headers = await prepareStreamHeaders(streamUrl);
    ensureCurrent();
    const base = import.meta.env.VITE_API_BASE_URL || '/v1';
    let response: Response;
    try {
      response = await fetch(base + streamUrl, { method: 'POST', headers, body: JSON.stringify(data), signal: combined });
    } catch (error) {
      if (combined.aborted) throw combined.reason;
      if (error instanceof DOMException && error.name === 'AbortError') throw error;
      throw new ApiError(500, '无法连接服务，请确认服务已启动');
    }
    ensureCurrent();
    if (!response.ok || !response.body) throw await readEnvelopeError(response);
    if (!(response.headers.get('Content-Type') ?? '').includes('text/event-stream')) throw await readEnvelopeError(response);
    let meta: AiStreamMeta | undefined;
    reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
    let buffer = '';
    for (;;) {
      const { done, value } = await reader.read();
      ensureCurrent();
      if (done) break;
      buffer += value;
      let separator = /\r?\n\r?\n/.exec(buffer);
      while (separator) {
        const frame = parseFrame(buffer.slice(0, separator.index));
        buffer = buffer.slice(separator.index + separator[0].length);
        separator = /\r?\n\r?\n/.exec(buffer);
        if (!frame) continue;
        ensureCurrent();
        if (frame.event === 'error') throw toApiError(frame.data);
        const payload = readObject(frame.data);
        if (frame.event === 'meta') {
          const next = readMeta(payload);
          if ((data.conversation_id !== 0 && next.conversation_id !== data.conversation_id)
            || (meta && (meta.conversation_id !== next.conversation_id || meta.user_message_id !== next.user_message_id || meta.ai_message_id !== next.ai_message_id))) throw protocolError();
          meta = next;
          handlers.onMeta?.(meta);
        } else if (frame.event === 'delta') {
          if (!meta || typeof payload.content !== 'string') throw protocolError();
          handlers.onDelta?.(payload.content);
        } else if (frame.event === 'tool') {
          if (!meta || typeof payload.name !== 'string' || (payload.status !== 'start' && payload.status !== 'done')) throw protocolError();
          handlers.onTool?.(payload.name, payload.status);
        } else if (frame.event === 'done') {
          if (!meta || !nonNegativeInteger(payload.tokens) || typeof payload.usage_complete !== 'boolean') throw protocolError();
          const result = readResult(payload.result, meta, data);
          handlers.onDone?.({ tokens: payload.tokens, usage_complete: payload.usage_complete, result });
          return result;
        }
      }
    }
    throw new ApiError(500, '流式响应中断，请重查本次结果');
  } finally {
    clearTimeout(timeout);
    unsubscribe();
    combined.removeEventListener('abort', cancelReader);
    // 终帧已足够确认结果，无需等待代理关闭连接；失败同样释放reader。
    if (reader) { await reader.cancel().catch(() => undefined); reader.releaseLock(); }
  }
}

function parseFrame(frame: string): { event: string; data: string } | undefined {
  let event = '';
  const data: string[] = [];
  for (const line of frame.split(/\r?\n/)) {
    if (line.startsWith('event:')) event = line.slice(6).trim();
    else if (line.startsWith('data:')) data.push(line.slice(5).trimStart());
  }
  return event && data.length ? { event, data: data.join('\n') } : undefined;
}

function readObject(raw: string): Record<string, unknown> {
  try {
    const value = JSON.parse(raw) as unknown;
    if (value && typeof value === 'object' && !Array.isArray(value)) return value as Record<string, unknown>;
  } catch { /* 损坏终帧不得被当成成功。 */ }
  throw protocolError();
}
const positiveInteger = (value: unknown): value is number => Number.isSafeInteger(value) && Number(value) > 0;
const nonNegativeInteger = (value: unknown): value is number => Number.isSafeInteger(value) && Number(value) >= 0;
function readMeta(payload: Record<string, unknown>): AiStreamMeta {
  if (!positiveInteger(payload.conversation_id) || !positiveInteger(payload.user_message_id)
    || !positiveInteger(payload.ai_message_id) || payload.user_message_id >= payload.ai_message_id) throw protocolError();
  return { conversation_id: payload.conversation_id, user_message_id: payload.user_message_id, ai_message_id: payload.ai_message_id };
}
function validMessage(value: unknown, id: number, role: string): value is AiChatMessage {
  if (!value || typeof value !== 'object') return false;
  const message = value as AiChatMessage;
  return message.id === id && message.role === role && message.status === 'succeeded' && message.error === ''
    && typeof message.content === 'string' && typeof message.tool_calls === 'string'
    && nonNegativeInteger(message.tokens) && nonNegativeInteger(message.created_at);
}
function validConversation(value: unknown, id: number): value is AiConversationInfo {
  if (!value || typeof value !== 'object') return false;
  const conversation = value as AiConversationInfo;
  return conversation.id === id && typeof conversation.title === 'string' && positiveInteger(conversation.user_id)
    && nonNegativeInteger(conversation.message_count) && nonNegativeInteger(conversation.created_at) && nonNegativeInteger(conversation.updated_at);
}
function readResult(value: unknown, meta: AiStreamMeta, request: AiSendMessageReq): AiSendMessageResult {
  if (!value || typeof value !== 'object') throw protocolError();
  const result = value as AiSendMessageResult;
  if (result.conversation_id !== meta.conversation_id || !validMessage(result.user_message, meta.user_message_id, 'user')
    || !validMessage(result.ai_message, meta.ai_message_id, 'assistant') || result.user_message.content !== request.content
    || (result.conversation !== null && !validConversation(result.conversation, result.conversation_id))) throw protocolError();
  return result;
}
function toApiError(raw: string): ApiError {
  const payload = readObject(raw);
  return new ApiError(typeof payload.code === 'number' ? payload.code : 500,
    typeof payload.msg === 'string' && payload.msg ? payload.msg : 'AI 调用失败');
}
async function readEnvelopeError(response: Response): Promise<ApiError> {
  try {
    const envelope = await response.json() as { code?: unknown; msg?: unknown };
    if (typeof envelope?.code === 'number' && envelope.code !== 200) {
      return new ApiError(envelope.code, typeof envelope.msg === 'string' ? envelope.msg : '请求失败');
    }
  } catch { /* 非信封响应按HTTP状态处理。 */ }
  return new ApiError(response.ok ? 500 : response.status || 500, '请求失败（HTTP ' + response.status + '）');
}
