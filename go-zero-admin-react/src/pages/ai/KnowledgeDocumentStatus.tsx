import { Alert } from 'antd';
import type { AiKbDocInfo } from '../../api/types';
import { documentStageLabel } from './knowledgeDocuments';

export function DocumentFeedback({ doc }: { doc: AiKbDocInfo }) {
  return <Alert type={doc.status === 'ready' ? 'success' : doc.status === 'failed' ? 'error' : 'info'} showIcon
    title={doc.status === 'ready' ? `“${doc.title}”已就绪，可以检索` : `“${doc.title}”：${documentStageLabel(doc)}`}
    description={doc.status === 'failed' ? doc.error || '文档已保存，请重试已有文档。'
      : doc.status === 'pending' ? '资料已接收，后台继续处理。可关闭页面，稍后到导入批次查看状态。' : undefined} />;
}
