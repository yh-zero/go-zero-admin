import { useState } from 'react';
import { Alert, Button, Checkbox, Space } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import type { AiKbFeedbackInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';
import { feedbackDatasetOptions, downloadQualityDataset } from './feedbackDataset';
import { FeedbackDatasetPreview } from './FeedbackDatasetPreview';
import styles from './knowledgeFeedback.module.css';

export function FeedbackDatasetGenerator({ scope, kbId, record, guard, onClose }: { scope: AiWorkspaceScope; kbId: number; record: AiKbFeedbackInfo; guard: ReturnType<typeof useAiWorkspaceGuard>; onClose: () => void }) {
  const [confirmed, setConfirmed] = useState(false);
  const negative = !record.review?.answerable;
  const data = useQuery({ ...feedbackDatasetOptions(scope, kbId, record, confirmed), enabled: guard.can('sys:ai:kb:list') && (!negative || confirmed) });
  const ready = data.isSuccess && !data.isFetching && !data.error && guard.isCurrent();
  return <section className={styles.detail} aria-label="生成验收题集">
    <Space wrap><h3>生成验收题集</h3><Button onClick={onClose}>关闭题集预览</Button></Space>
    {negative && <><Alert type="warning" title="原反馈不能证明整个知识库无答案" description="请先核对当前知识库全部就绪资料。确认后将读取并冻结当前就绪资料；最多100篇、完整题集最多256 KiB，超限或空库不生成。" />
      <Link to={`/ai/knowledge/${kbId}`}>打开当前知识库核对资料</Link>
      <Checkbox checked={confirmed} disabled={data.isFetching} onChange={event => setConfirmed(event.target.checked)}>我已核对当前库全部就绪资料，确认本题资料不足</Checkbox></>}
    {(!negative || confirmed) && <QueryState isPending={data.isPending || data.isFetching} error={data.error} onRetry={() => data.refetch()} />}
    {ready && <><FeedbackDatasetPreview catalogue={data.data} /><Space wrap>
      <Button onClick={() => { if (guard.isCurrent()) downloadQualityDataset(data.data.raw, `kb-${kbId}-feedback-${record.id}-r${record.revision}.json`); }}>下载冻结题集</Button>
      <Link to={`/ai/knowledge/${kbId}/quality`} state={{ datasetRaw: data.data.raw, datasetScope: [...scope] }}><Button>导入质量验收页</Button></Link>
    </Space></>}
  </section>;
}
