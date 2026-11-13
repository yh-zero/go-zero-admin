import { useState } from 'react';
import { Alert, Button, Checkbox, Dropdown, Space } from 'antd';
import { DownOutlined } from '@ant-design/icons';
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
  return <section className={styles.detail} aria-label="评测题集预览">
    <Space wrap><h3>核对评测题集</h3><Button onClick={onClose}>关闭题集预览</Button></Space>
    <p className={styles.note}>核对后可前往新建评测，只有点击“开始评测”才执行模型。</p>
    {negative && <><Alert type="warning" title="原反馈不能证明整个知识库无答案" description="请先核对当前知识库全部就绪资料。确认后将读取并冻结当前就绪资料；最多100篇、完整题集最多256 KiB，超限或空库不生成。" />
      <Link to={`/ai/knowledge/${kbId}`}>打开当前知识库核对资料</Link>
      <Checkbox checked={confirmed} disabled={data.isFetching} onChange={event => setConfirmed(event.target.checked)}>我已核对当前库全部就绪资料，确认本题资料不足</Checkbox></>}
    {(!negative || confirmed) && <QueryState isPending={data.isPending || data.isFetching} error={data.error} onRetry={() => data.refetch()} />}
    {ready && <><FeedbackDatasetPreview catalogue={data.data} /><Space wrap>
      <Link to={`/ai/knowledge/${kbId}/quality`} state={{ datasetRaw: data.data.raw, datasetScope: [...scope] }}><Button type="primary">确认题集，前往新建评测</Button></Link>
      <Dropdown trigger={['click']} menu={{ items: [{ key: 'dataset', label: '下载冻结题集', onClick: () => { if (guard.isCurrent()) downloadQualityDataset(data.data.raw, `kb-${kbId}-feedback-${record.id}-r${record.revision}.json`); } }] }}>
        <Button aria-label="更多导出">更多导出 <DownOutlined aria-hidden /></Button>
      </Dropdown>
    </Space></>}
  </section>;
}
