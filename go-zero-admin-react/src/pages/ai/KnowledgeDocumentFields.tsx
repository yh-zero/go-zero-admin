import { Form, Input } from 'antd';
import { DOCUMENT_MAX_BYTES, documentContentBytes, validateDocumentContent } from './knowledgeDocuments';

export function KnowledgeDocumentFields({ content }: { content: string }) {
  return <><Form.Item name="title" label="标题" rules={[{ required: true, max: 128, whitespace: true, message: '标题须为1..128字符，不能为空白' }]}>
    <Input maxLength={128} placeholder="如：发布流程规范" />
  </Form.Item><Form.Item name="content" label="正文（纯文本）" extra={`${documentContentBytes(content)} / ${DOCUMENT_MAX_BYTES} 字节（UTF-8）`}
    rules={[{ validator: (_, value: string | undefined) => { const error = validateDocumentContent(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve(); } }]}>
    <Input.TextArea rows={12} placeholder="粘贴资料原文；保存后开始处理，就绪后可用于检索" />
  </Form.Item></>;
}
