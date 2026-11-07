import type { AiCreateKbDocReq, AiKbDocInfo, AiKbDocResp } from '../../api/types';
import { DOCUMENT_MAX_BYTES, documentWriteFailure, validateDocumentContent } from './knowledgeDocuments';

export const KNOWLEDGE_IMPORT_MAX_FILES = 20;
export interface KnowledgeImportItem { id: string; values: AiCreateKbDocReq }
export type KnowledgeImportState = 'waiting' | 'submitting' | 'ready' | 'failed' | 'pending' | 'unknown';
export interface KnowledgeImportChange { id: string; state: KnowledgeImportState; doc?: AiKbDocInfo; error?: string }

export async function readKnowledgeFile(file: File): Promise<AiCreateKbDocReq> {
  if (!/\.(txt|md|markdown)$/i.test(file.name)) throw new Error('仅支持 TXT、MD 和 Markdown 文件');
  if (file.size === 0 || file.size > DOCUMENT_MAX_BYTES) throw new Error('每个文件须为 1..65536 字节');
  const title = file.name.replace(/\.(txt|md|markdown)$/i, '').trim();
  if (!title || Array.from(title).length > 128) throw new Error('文件名去掉扩展名后须为 1..128 字符');
  let content: string;
  try {
    // 严格解码避免静默替换乱码；BOM 和 CRLF 保持原貌，不改变待索引资料。
    content = new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(await file.arrayBuffer());
  } catch { throw new Error('文件必须使用 UTF-8 编码，请转换编码后重新选择'); }
  const error = validateDocumentContent(content);
  if (error) throw new Error(error);
  return { title, content };
}

export async function runKnowledgeImport(items: KnowledgeImportItem[], submit: (values: AiCreateKbDocReq) => Promise<AiKbDocResp>,
  onChange: (change: KnowledgeImportChange) => void, isCurrent: () => boolean): Promise<void> {
  if (items.length > KNOWLEDGE_IMPORT_MAX_FILES) throw new Error('每批最多 20 个文件');
  let cursor = 0;
  async function worker() {
    while (isCurrent() && cursor < items.length) {
      const item = items[cursor++];
      onChange({ id: item.id, state: 'submitting' });
      try {
        const result = await submit(item.values);
        if (isCurrent()) onChange({ id: item.id, state: result.doc.status, doc: result.doc });
      } catch (cause) {
        // 未获得权威文档ID时无法断言回滚，也不能自动重放可能已保存的资料。
        const failure = documentWriteFailure(cause);
        if (isCurrent()) onChange({ id: item.id, state: failure.uncertain ? 'unknown' : 'failed', error: failure.message });
      }
    }
  }
  await Promise.all([worker(), worker()]);
}
