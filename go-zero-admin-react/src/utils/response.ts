export class ApiError extends Error {
  readonly code: number;
  constructor(code: number, message: string) { super(message); this.name = 'ApiError'; this.code = code; }
}

// 兼容当前裸成功响应，后端统一信封后页面无需再次修改。
export function decodeResponse<T = unknown>(body: unknown): T {
  if (body === '' || body === null || body === undefined) return undefined as T;
  if (typeof body === 'object' && 'code' in body && typeof body.code === 'number') {
    const envelope = body as { code: number; msg?: string; result?: T };
    if (envelope.code !== 200) throw new ApiError(envelope.code, typeof envelope.msg === 'string' && envelope.msg ? envelope.msg : '请求未完成');
    return normalizePayload<T>(envelope.result);
  }
  return normalizePayload<T>(body);
}

function normalizePayload<T>(body: unknown): T {
  if (typeof body === 'string') throw new ApiError(502, '服务返回了非 JSON 数据，请检查 API 地址');
  if (body && typeof body === 'object' && 'list' in body && body.list === null) return { ...body, list: [] } as T;
  return body as T;
}
