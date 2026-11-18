import { Alert, Button, Space, Switch, Typography } from 'antd';
import { useAuthStore } from '../../stores/auth';
import { KnowledgeChoice } from './KnowledgeChoice';
import type { AiChatScope } from './chatQueries';
import type { ChatKnowledgeSelection } from './chatKnowledge';
import styles from './chat.module.css';

interface DocumentScope { title?: string; loading: boolean; problem?: string; retry: () => void }

export function ChatKnowledgePicker({ scope, value, onChange, disabled, documentScope }: { scope: AiChatScope; value: ChatKnowledgeSelection; onChange: (value: ChatKnowledgeSelection) => void; disabled: boolean; documentScope?: DocumentScope }) {
  const canRead = useAuthStore(state => state.permissionsReady && state.permCodes.includes('sys:ai:kb:list'));
  if (value.doc_id) {
    if (!canRead) return <Alert type="warning" showIcon title="当前没有查看这篇资料的权限，已暂停发送并保留原范围。" />;
    return <div className={styles.knowledgePicker} role="group" aria-label="本次提问资料">
      <Space wrap><Typography.Text strong>{documentScope?.title ?? (documentScope?.loading ? '正在读取这篇资料…' : '已选资料')}</Typography.Text>
        <Button size="small" disabled={disabled} onClick={() => onChange({ kb_id: value.kb_id, grounded: true })}>改问整个知识库</Button></Space>
      {documentScope?.problem ? <Alert type="warning" showIcon title={documentScope.problem} action={<Button size="small" loading={documentScope.loading} onClick={documentScope.retry}>重试</Button>} />
        : <Typography.Text type="secondary" className={styles.recordNote}>回答和追问仅使用这篇资料。</Typography.Text>}
    </div>;
  }
  if (!canRead) return value.kb_id > 0 ? <Alert type="warning" title="当前没有查看原知识库的权限，已保留原资料范围。"
    action={<Button size="small" disabled={disabled} onClick={() => onChange({ kb_id: 0, grounded: false })}>改用自动检索</Button>} /> : null;
  return <div className={styles.knowledgePicker}>
    <Space wrap>
      <div className={styles.knowledgeSelect}><KnowledgeChoice label="对话知识库" allLabel="自动检索我的知识库" optional knowledge={{ scope, canRead }} value={value.kb_id} disabled={disabled}
        onChange={next => { const id = typeof next === 'number' ? next : 0; onChange({ kb_id: id, grounded: id > 0 && value.grounded }); }} /></div>
      {value.kb_id > 0 && <Space size="small"><Switch aria-label="仅依据所选资料回答" checked={value.grounded} disabled={disabled} onChange={grounded => onChange({ ...value, grounded })} /><Typography.Text>仅依据资料回答</Typography.Text></Space>}
    </Space>
    {value.grounded && <Typography.Text type="secondary" className={styles.recordNote}>本次问题最多 2000 UTF-8 字节；依据不足时说明无法回答。来源为实际检索摘录，相似度不代表答案正确率。</Typography.Text>}
  </div>;
}
