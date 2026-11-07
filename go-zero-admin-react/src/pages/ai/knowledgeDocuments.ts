import type { AiKbDocInfo } from '../../api/types';
import { ApiError } from '../../utils/response';

export const DOCUMENT_MAX_BYTES = 65_536;

export function documentContentBytes(content: string): number {
  return new TextEncoder().encode(content).byteLength;
}

export function validateDocumentContent(content: string): string | undefined {
  if (!content.trim()) return '正文不能为空白';
  if (documentContentBytes(content) > DOCUMENT_MAX_BYTES) return '正文最多 65536 字节（UTF-8）';
}

export function canRetryDocument(doc: AiKbDocInfo): boolean {
  return doc.status === 'failed';
}

export function documentStageLabel(doc: AiKbDocInfo): string {
  if (doc.status === 'ready') return '已就绪';
  if (doc.status === 'failed') return '处理失败';
  if (doc.stage === 'queued') return '等待后台处理';
  return doc.stage === 'indexing' ? '写入检索索引' : '生成文档向量';
}

export function documentWriteFailure(error: unknown): { uncertain: boolean; message: string } {
  if (error instanceof ApiError && [400, 403, 404].includes(error.code)) {
    return { uncertain: false, message: error.message };
  }
  return { uncertain: true, message: '尚未确认处理结果，请刷新核对。网络错误不代表服务端撤销。' };
}
