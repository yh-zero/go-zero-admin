import { Alert, Drawer, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import type { AiSearchKbHit } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { knowledgeDocumentOptions } from './knowledgeDocumentQueries';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';
import styles from './chat.module.css';

interface Props { source: AiSearchKbHit; onClose: () => void }

export function ChatSourcePreview({ source, onClose }: Props) {
  const auth = useAuthStore();
  const scope: AiWorkspaceScope = [auth.sessionId, auth.sessionVersion, auth.departmentVersion];
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list');
  const canRead = guard.can('sys:ai:kb:list');
  const detail = useQuery({ ...knowledgeDocumentOptions(scope, source.kb_id, source.doc_id), enabled: canRead });

  return <Drawer open title="当前原文" placement="right" size={720} rootClassName={styles.sourceDrawer} mask={{ closable: false }} onClose={onClose}>
    <Typography.Paragraph type="secondary">当前原文可能已更新，与回答时的版本不同。本次引用片段保留在对话中。</Typography.Paragraph>
    {!canRead ? <Alert type="warning" showIcon title="暂无查看原文权限" description="仍可在对话中查看本次引用片段。" /> : <>
      <QueryState isPending={detail.isPending || detail.isFetching} error={detail.error} onRetry={() => { if (guard.isCurrent()) void detail.refetch(); }} />
      {!detail.isFetching && !detail.error && detail.data && <>
        <Typography.Title level={4}>{detail.data.doc.title}</Typography.Title>
        <Typography.Paragraph type="secondary">第 {detail.data.doc.version} 版</Typography.Paragraph>
        <div className={styles.sourceDocument} aria-label="当前原文全文">{detail.data.content}</div>
      </>}
    </>}
  </Drawer>;
}
