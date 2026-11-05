import { ApiError } from '../utils/response';
import { prepareStreamHeaders } from '../utils/request';
import { requiredWritePermission } from '../utils/permissions';
import { useAuthStore } from '../stores/auth';
import type { AiSendMessageReq } from './types';

export interface AiStreamMeta { conversation_id: number; user_message_id: number; ai_message_id: number }
export interface AiStreamDone { tokens: number; usage_complete: boolean }
export interface AiStreamHandlers {
  onMeta?: (meta: AiStreamMeta) => void;
  onDelta?: (content: string) => void;
  onTool?: (name: string, status: 'start' | 'done') => void;
  onDone?: (done: AiStreamDone) => void;
}

const streamUrl = '/sys/ai/messages/stream';

// 流式发送（SSE）。契约见 api-contract.md：request_id 幂等语义同 /messages；
// error 帧为业务失败终帧；流中断不续传，调用方以服务端会话状态为准。
export async function sendAiMessageStream(data: AiSendMessageReq, handlers: AiStreamHandlers, signal?: AbortSignal): Promise<AiStreamMeta> {
  const auth = useAuthStore.getState();
  const required = requiredWritePermission('POST', streamUrl);
  if (required && (!auth.permissionsReady || !auth.permCodes.includes(required))) {
    throw new ApiError(403, '操作权限未加载或已变化，请重新加载权限后重试');
  }
  const headers = await prepareStreamHeaders(streamUrl);
  const base = import.meta.env.VITE_API_BASE_URL || '/v1';

  let response: Response;
  try {
    response = await fetch(base + streamUrl, { method: 'POST', headers, body: JSON.stringify(data), signal });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error;
    throw new ApiError(500, '无法连接服务，请确认服务已启动');
  }
  if (!response.ok || !response.body) {
    // 开流前失败仍是统一 JSON 信封
    throw await readEnvelopeError(response);
  }

  let meta: AiStreamMeta | undefined;
  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
  let buffer = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += value;
    let sep = buffer.indexOf('\n\n');
    while (sep >= 0) {
      const frame = parseFrame(buffer.slice(0, sep));
      buffer = buffer.slice(sep + 2);
      sep = buffer.indexOf('\n\n');
      if (!frame) continue;
      if (frame.event === 'error') throw toApiError(frame.data);
      if (frame.event === 'meta') {
        meta = readMeta(frame.data);
        handlers.onMeta?.(meta);
        continue;
      }
      if (frame.event === 'delta') {
        const payload = readObject(frame.data);
        handlers.onDelta?.(String(payload.content ?? ''));
        continue;
      }
      if (frame.event === 'tool') {
        const payload = readObject(frame.data);
        handlers.onTool?.(String(payload.name ?? ''), payload.status === 'done' ? 'done' : 'start');
        continue;
      }
      if (frame.event === 'done') {
        const payload = readObject(frame.data);
        handlers.onDone?.({ tokens: Number(payload.tokens ?? 0), usage_complete: Boolean(payload.usage_complete) });
        if (!meta) throw new ApiError(500, '流式响应缺少会话标识');
        return meta;
      }
    }
  }
  throw new ApiError(500, '流式响应中断，请查看会话消息状态后重试');
}

// 解析一帧 SSE：event + data 行；心跳等注释行返回 undefined。
function parseFrame(frame: string): { event: string; data: string } | undefined {
  let event = '';
  let data = '';
  for (const line of frame.split('\n')) {
    if (line.startsWith(':')) return undefined;
    if (line.startsWith('event:')) event = line.slice(6).trim();
    else if (line.startsWith('data:')) data += (data ? '\n' : '') + line.slice(5).trim();
  }
  return event && data ? { event, data } : undefined;
}

function readObject(raw: string): Record<string, unknown> {
  try {
    const value = JSON.parse(raw) as unknown;
    return value && typeof value === 'object' ? value as Record<string, unknown> : {};
  } catch {
    return {}; // 单帧损坏不中断整流；终帧缺失按中断处理
  }
}

function readMeta(raw: string): AiStreamMeta {
  const payload = readObject(raw);
  return {
    conversation_id: Number(payload.conversation_id ?? 0),
    user_message_id: Number(payload.user_message_id ?? 0),
    ai_message_id: Number(payload.ai_message_id ?? 0),
  };
}

function toApiError(raw: string): ApiError {
  const payload = readObject(raw);
  const code = typeof payload.code === 'number' ? payload.code : 500;
  const msg = typeof payload.msg === 'string' && payload.msg ? payload.msg : 'AI 调用失败';
  return new ApiError(code, msg);
}

async function readEnvelopeError(response: Response): Promise<ApiError> {
  try {
    const envelope = await response.json() as { code?: unknown; msg?: unknown };
    if (typeof envelope?.code === 'number') {
      return new ApiError(envelope.code, typeof envelope.msg === 'string' ? envelope.msg : '请求失败');
    }
  } catch { /* 非信封响应按 HTTP 状态处理 */ }
  return new ApiError(response.status || 500, '请求失败（HTTP ' + response.status + '）');
}
