import { useState } from 'react';
import { Alert, Pagination, Select, Space, Switch, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchAiKb } from '../../api/ai';
import { useAuthStore } from '../../stores/auth';
import { kbListOptions } from './workspaceQueries';
import type { AiChatScope } from './chatQueries';
import styles from './chat.module.css';

export interface ChatKnowledgeSelection { kb_id: number; grounded: boolean }
export function ChatKnowledgePicker({ scope, value, onChange, disabled }: { scope: AiChatScope; value: ChatKnowledgeSelection; onChange: (value: ChatKnowledgeSelection) => void; disabled: boolean }) {
  const canRead = useAuthStore(state => state.permissionsReady && state.permCodes.includes('sys:ai:kb:list'));
  const [page, setPage] = useState(1);
  const query = useQuery({ ...kbListOptions(scope, { page, page_size: 20 }), enabled: canRead });
  const selected = useQuery({ queryKey: ['ai-chat', ...scope, 'selected-kb', value.kb_id], enabled: canRead && value.kb_id > 0 && !query.data?.list.some(kb => kb.id === value.kb_id), retry: false,
    queryFn: async ({ signal }) => { const result = await fetchAiKb(value.kb_id); signal.throwIfAborted(); return result.kb; } });
  if (!canRead) return null;
  const rows = query.data?.list ?? [];
  const options = rows.map(kb => ({ value: kb.id, label: `${kb.name}（${kb.doc_count} 篇）` }));
  if (value.kb_id > 0 && !rows.some(kb => kb.id === value.kb_id)) options.push({ value: value.kb_id, label: selected.data?.name ?? `知识库 #${value.kb_id}` });
  return <div className={styles.knowledgePicker}>
    <Space wrap>
      <Select aria-label="对话知识库" className={styles.knowledgeSelect} value={value.kb_id} disabled={disabled} loading={query.isFetching} options={[{ value: 0, label: '自动检索我的知识库' }, ...options]}
        onChange={kb_id => onChange({ kb_id, grounded: kb_id > 0 && value.grounded })} />
      {value.kb_id > 0 && <Space size="small"><Switch aria-label="仅依据所选资料回答" checked={value.grounded} disabled={disabled} onChange={grounded => onChange({ ...value, grounded })} /><Typography.Text>仅依据资料回答</Typography.Text></Space>}
    </Space>
    {(query.data?.total ?? 0) > 20 && <Pagination size="small" simple current={page} pageSize={20} total={query.data!.total} showSizeChanger={false} disabled={disabled} onChange={setPage} />}
    {value.grounded && <Typography.Text type="secondary" className={styles.recordNote}>本次问题最多 2000 UTF-8 字节；依据不足时说明无法回答。来源为实际检索摘录，相似度不代表答案正确率。</Typography.Text>}
    {(query.isError || selected.isError) && <Alert type="warning" title="知识库读取失败，请刷新后重试" />}
  </div>;
}
