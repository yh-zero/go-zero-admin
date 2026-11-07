import { createAiKbDoc, retryAiKbDoc } from '../../api/ai';
import type { AiCreateKbDocReq, AiKbDocInfo } from '../../api/types';
import { ApiError } from '../../utils/response';

export const DOCUMENT_MAX_BYTES = 65_536;

export function documentContentBytes(content: string): number {
  return new TextEncoder().encode(content).byteLength;
}

export function validateDocumentContent(content: string): string | undefined {
  if (!content.trim()) return '正文不能为空白';
  if (documentContentBytes(content) > DOCUMENT_MAX_BYTES) return '正文最多 65536 字节（UTF-8）';
}

export function canRetryDocument(doc: AiKbDocInfo, now = Date.now()): boolean {
  return doc.status === 'failed' || (doc.status === 'pending' && now >= doc.updated_at * 1000 + 120_000);
}

export function documentStageLabel(doc: AiKbDocInfo): string {
  if (doc.status === 'ready') return '已就绪';
  if (doc.status === 'failed') return '处理失败';
  return doc.stage === 'indexing' ? '写入检索索引' : '生成文档向量';
}

export function documentWriteFailure(error: unknown): { uncertain: boolean; message: string } {
  if (error instanceof ApiError && [400, 403, 404].includes(error.code)) {
    return { uncertain: false, message: error.message };
  }
  return { uncertain: true, message: '尚未确认处理结果，请刷新核对。网络错误不代表服务端撤销。' };
}

// 保留已创建文档的身份，恢复操作不会因表单还在而重复创建。
export function documentWriteOptions(kbId: number) {
  return { retry: false as const, mutationFn: async ({ values, doc }: { values: AiCreateKbDocReq; doc?: AiKbDocInfo }) => {
    if (doc) {
      if (!canRetryDocument(doc)) throw new Error('文档仍在处理，请等待保护期结束后刷新状态');
      return retryAiKbDoc(kbId, doc.id);
    }
    const error = validateDocumentContent(values.content);
    if (error) throw new Error(error);
    return createAiKbDoc(kbId, values);
  } };
}
