import { Alert, Button } from 'antd';
import type { AiKbFeedbackInfo } from '../../api/types';
import { feedbackLabels, reviewStatusLabels } from './knowledgeFeedback';
import styles from './knowledgeFeedback.module.css';

export function FeedbackRevisionNotice({ record, expectedRevision, refreshing, readVerified = true, allowUnchangedRevision = false, allowEmptyRecord = false, onAcknowledge }: {
  record?: AiKbFeedbackInfo; expectedRevision: number; refreshing: boolean; readVerified?: boolean;
  allowUnchangedRevision?: boolean; allowEmptyRecord?: boolean; onAcknowledge: (revision: number) => void;
}) {
  const revision = record?.revision ?? (allowEmptyRecord ? 0 : undefined);
  return <Alert type="warning" showIcon title="记录已变化，草稿已保留"
    description={<>
      <p>请核对最新评价和复核结论，确认后再提交；不会自动覆盖其他更新。</p>
      {record && <><p>最新版本 {record.revision} · {feedbackLabels[record.feedback]} · {reviewStatusLabels[record.status]}</p>
        {record.note && <p className={styles.text}>评价说明：{record.note}</p>}{record.review?.reason && <p className={styles.text}>复核说明：{record.review.reason}</p>}
        {record.review && <details><summary>查看最新完整复核结论</summary><pre className={styles.code}>{JSON.stringify(record.review, null, 2)}</pre></details>}</>}
      {!record && allowEmptyRecord && readVerified && <p>最新记录尚无评价，确认后以新评价提交。</p>}
      {!readVerified && <p>最新详情尚未成功核对，请先重新加载；当前保留的是之前读取的记录。</p>}
      <Button disabled={refreshing || !readVerified || revision === undefined || !allowUnchangedRevision && revision === expectedRevision}
        onClick={() => { if (readVerified && !refreshing && revision !== undefined) onAcknowledge(revision); }}>已核对最新版本，保留草稿继续</Button>
    </>} />;
}
