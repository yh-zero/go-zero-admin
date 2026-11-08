import { useEffect, useState } from 'react';
import { Alert, Button, Drawer, Empty, Form, Input, Pagination, Radio, Select, Space, Tag, Typography } from 'antd';
import { MinusCircleOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { fetchAiKbFeedback } from '../../api/ai';
import type { AiKbFeedbackInfo, AiListKbFeedbackReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { ApiError } from '../../utils/response';
import { FeedbackRevisionNotice } from './FeedbackRevisionNotice';
import { FeedbackDatasetGenerator } from './FeedbackDatasetGenerator';
import { feedbackLabels, feedbackRegressionJson, kbFeedbackDetailOptions, kbFeedbackListOptions, kbReviewMutationOptions, parseFeedbackSnapshot, reviewStatusLabels, reviewValuesToRequest } from './knowledgeFeedback';
import type { FeedbackReviewValues, FeedbackSnapshot } from './knowledgeFeedback';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import styles from './knowledgeFeedback.module.css';

type Guard = ReturnType<typeof useAiWorkspaceGuard>;

export function KnowledgeFeedbackPanel({ scope, kbId, guard }: { scope: AiWorkspaceScope; kbId: number; guard: Guard }) {
  const [params, setParams] = useState<AiListKbFeedbackReq>({ page: 1, page_size: 10, status: 'pending' });
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const records = useQuery({ ...kbFeedbackListOptions(scope, kbId, params), enabled: guard.can('sys:ai:kb:list') });
  const total = records.data?.total;
  const current = guard.isCurrent;
  useEffect(() => {
    if (!current() || records.isFetching || !records.isSuccess || total === undefined) return;
    const lastPage = Math.max(1, Math.ceil(total / (params.page_size ?? 10)));
    if ((params.page ?? 1) <= lastPage) return;
    // 仅当前筛选的已完成结果可校正页码，旧筛选不能改写新列表。
    // oxlint-disable-next-line react/set-state-in-effect
    setParams(value => value === params ? { ...value, page: lastPage } : value);
  }, [current, params, records.isFetching, records.isSuccess, total]);
  return <section className={styles.panel} aria-label="知识库反馈与复核">
    <Typography.Paragraph type="secondary">这里保留本人知识库分析的评价。复核后可下载回归样本；采纳不会自动重跑分析或修改模型。</Typography.Paragraph>
    <div className={styles.toolbar}><Select aria-label="反馈状态" className={styles.statusSelect} value={params.status} disabled={guard.writing}
      options={[{ value: 'pending', label: '待复核' }, { value: 'accepted', label: '已采纳' }, { value: 'dismissed', label: '已忽略' }, { value: 'all', label: '全部评价' }]}
      onChange={status => { if (guard.isCurrent() && !guard.writing) setParams(value => ({ ...value, page: 1, status })); }} />
      <Button icon={<ReloadOutlined />} loading={records.isFetching} disabled={guard.writing} onClick={() => void records.refetch()}>刷新评价</Button></div>
    <QueryState isPending={records.isPending} error={records.error} onRetry={() => records.refetch()} />
    {records.data && !records.error && <>
      {records.data.list.length ? <div className={styles.records}>{records.data.list.map(record => <article key={record.id} className={styles.record}>
        <div className={styles.recordHeader}><Tag>{feedbackLabels[record.feedback]}</Tag><Tag color={record.status === 'accepted' ? 'success' : record.status === 'pending' ? 'processing' : 'default'}>{reviewStatusLabels[record.status]}</Tag>
          <span className={styles.note}>任务 #{record.task_id} · 第 {record.step_index + 1} 步</span></div>
        <p>{record.question}</p>{record.note && <p className={styles.note}>{record.note}</p>}
        <Space wrap><Button size="small" disabled={guard.writing} onClick={() => { if (guard.isCurrent()) setSelectedId(record.id); }}>{guard.can('sys:ai:kb:review') ? '查看并复核' : '查看评价'}</Button>
          <span className={styles.note}>{formatTime(record.updated_at)}</span></Space>
      </article>)}</div> : <Empty description={params.status === 'pending' ? '暂无待复核评价，可在分析结果下提交评价。' : '当前筛选下暂无评价。'} />}
      <Pagination current={params.page} pageSize={params.page_size} total={records.data.total} showSizeChanger pageSizeOptions={[10, 20, 50]} disabled={guard.writing}
        showTotal={count => `共 ${count} 条`} onChange={(page, pageSize) => { if (guard.isCurrent()) setParams(value => ({ ...value, page: pageSize === value.page_size ? page : 1, page_size: pageSize })); }} />
    </>}
    {selectedId && <KnowledgeFeedbackDetail key={selectedId} scope={scope} kbId={kbId} id={selectedId} guard={guard} onClose={() => setSelectedId(null)} />}
  </section>;
}

function KnowledgeFeedbackDetail({ scope, kbId, id, guard, onClose }: { scope: AiWorkspaceScope; kbId: number; id: number; guard: Guard; onClose: () => void }) {
  const client = useQueryClient();
  const detail = useQuery({ ...kbFeedbackDetailOptions(scope, kbId, id), enabled: guard.can('sys:ai:kb:list') });
  const [notice, setNotice] = useState('');
  const [generating, setGenerating] = useState(false);
  const record = detail.data?.feedback;
  const snapshot = record ? parseFeedbackSnapshot(record.snapshot) : null;
  const readVerified = detail.isSuccess && !detail.isFetching && !detail.error;
  const readFailureVersion = client.getQueryState(aiWorkspaceKeys.kbFeedback(scope, kbId, id))?.errorUpdateCount ?? 0;

  async function download() {
    if (!readVerified || !guard.beginWrite('sys:ai:kb:list')) return;
    try {
      // 下载前复核知识库存续和当前采纳状态，不能从软删前的缓存继续导出。
      const latest = await fetchAiKbFeedback(kbId, id);
      if (!guard.isCurrent()) return;
      const json = feedbackRegressionJson(latest.feedback);
      client.setQueryData(aiWorkspaceKeys.kbFeedback(scope, kbId, id), latest);
      const url = URL.createObjectURL(new Blob([json], { type: 'application/json;charset=utf-8' }));
      const link = document.createElement('a');
      link.href = url; link.download = `kb-feedback-${id}-r${latest.feedback.revision}.json`;
      document.body.append(link); link.click(); link.remove();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      setNotice('已下载本条已采纳样本。离线转换时仍需核对知识库映射和资料版本。');
    } catch {
      if (guard.isCurrent()) { setNotice('当前记录未能确认可导出，请刷新并核对采纳状态。'); await detail.refetch(); }
    } finally { guard.finishWrite(); }
  }

  return <Drawer open title={`反馈复核 #${id}`} size="large" styles={{ wrapper: { width: 'min(820px, 100vw)' } }} onClose={() => { if (!guard.writing) onClose(); }}
    closable={!guard.writing} maskClosable={!guard.writing} keyboard={!guard.writing}
    extra={<Button aria-label="刷新反馈详情" icon={<ReloadOutlined />} disabled={guard.writing} onClick={() => void detail.refetch()} />}>
    <QueryState isPending={!record && detail.isPending} error={!record ? detail.error : undefined} onRetry={() => detail.refetch()} />
    {record && detail.error && <Alert type="warning" showIcon title="详情刷新失败，草稿已保留"
      description={detail.error instanceof Error ? detail.error.message : '请重新加载后核对最新记录。'}
      action={<Button disabled={guard.writing} onClick={() => void detail.refetch()}>重新加载详情</Button>} />}
    {notice && <Alert type="info" showIcon title={notice} />}
    {record && <div className={styles.detail}>
      <Space wrap><Tag>{feedbackLabels[record.feedback]}</Tag><Tag>{reviewStatusLabels[record.status]}</Tag><Typography.Text type="secondary">版本 {record.revision}</Typography.Text>
        {guard.can('sys:ai:task:list') && <Link to={`/ai/tasks?task_id=${record.task_id}`}>打开原任务 #{record.task_id}</Link>}
        {record.status === 'accepted' && <><Button disabled={guard.writing || !snapshot || !readVerified} onClick={() => void download()}>下载回归样本</Button>
          <Button disabled={guard.writing || !snapshot || !readVerified} onClick={() => { if (guard.isCurrent()) setGenerating(true); }}>生成验收题集</Button></>}</Space>
      {generating && readVerified && record.status === 'accepted' && <FeedbackDatasetGenerator key={`${record.id}:${record.revision}`} scope={scope} kbId={kbId} record={record} guard={guard} onClose={() => setGenerating(false)} />}
      <section><h3>原始问题</h3><p className={styles.text}>{record.question}</p><h3>当时的分析回答</h3><pre className={styles.code}>{record.answer}</pre>
        {record.note && <><h4>评价说明</h4><p className={styles.text}>{record.note}</p></>}</section>
      {snapshot ? <FeedbackSourceSnapshot snapshot={snapshot} /> : <Alert type="warning" title="历史快照无法识别，暂不能采纳或导出，请核对原始记录。" />}
      {record.review && <section><h3>已保存的复核结论</h3><p className={styles.text}>{record.review.reason}</p>
        {record.review.expected_answer && <><h4>预期答案</h4><pre className={styles.code}>{record.review.expected_answer}</pre></>}
        <details><summary>完整复核记录</summary><pre className={styles.code}>{JSON.stringify(record.review, null, 2)}</pre></details></section>}
      {snapshot && guard.can('sys:ai:kb:review') && <FeedbackReviewForm scope={scope} kbId={kbId} record={record} snapshot={snapshot} guard={guard}
        refreshing={detail.isFetching} readVerified={!!readVerified} readFailureVersion={readFailureVersion} />}
      {!guard.can('sys:ai:kb:review') && <Alert type="info" title="需要知识库复核权限才能采纳或忽略评价。" />}
      <details><summary>原始资料快照</summary><pre className={styles.code}>{record.snapshot}</pre></details>
    </div>}
  </Drawer>;
}

export function FeedbackSourceSnapshot({ snapshot }: { snapshot: FeedbackSnapshot }) {
  return <section><h3>本次分析使用的历史片段</h3><p className={styles.note}>以下是当时传给模型的片段；资料版本和正文摘要在评价采集时核对，不能证明模型调用时的完整资料版本。当前原文可能已更新。</p>
    {snapshot.sources.length ? <ol className={styles.sources}>{snapshot.sources.map(source => <li key={source.index}>
      <strong>[{source.index}] {source.doc_title}</strong><p className={styles.text}>{source.snippet}</p>
      <a href={`/ai/knowledge/${source.kb_id}?doc_id=${source.doc_id}`}>打开当前原文</a>
    </li>)}</ol> : <p>本次没有检索到资料片段。</p>}
    {snapshot.documents.some(doc => doc.status !== 'verified') && <Alert type="warning" showIcon title="部分资料在评价时已变化或不可核对，导入回归题集前须重新复核。" />}
    <details><summary>评价采集时核对的资料版本</summary>{snapshot.documents.map(doc => <p key={doc.doc_id} className={styles.text}>{doc.title} · 文档 #{doc.doc_id} · 版本 {doc.version} · {doc.status === 'verified' ? '已核对' : doc.status === 'stale' ? '已变化' : '不可核对'}<br />SHA256：{doc.content_sha256 || '未知'}</p>)}</details>
  </section>;
}

function FeedbackReviewForm({ scope, kbId, record, snapshot, guard, refreshing, readVerified, readFailureVersion }: {
  scope: AiWorkspaceScope; kbId: number; record: AiKbFeedbackInfo; snapshot: FeedbackSnapshot; guard: Guard;
  refreshing: boolean; readVerified: boolean; readFailureVersion: number;
}) {
  const client = useQueryClient();
  const [form] = Form.useForm<FeedbackReviewValues>();
  const status = Form.useWatch('status', form) ?? 'accepted';
  const answerable = Form.useWatch('answerable', form) ?? true;
  const [expectedRevision, setExpectedRevision] = useState(record.revision);
  const [conflict, setConflict] = useState(false);
  const [acknowledgedReadFailure, setAcknowledgedReadFailure] = useState(readFailureVersion);
  const [notice, setNotice] = useState('');
  const save = useMutation(kbReviewMutationOptions(scope, kbId, record.id));
  const needsConfirmation = conflict || record.revision !== expectedRevision || readFailureVersion > acknowledgedReadFailure;
  async function refreshRecord() {
    await Promise.allSettled([
      client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbFeedback(scope, kbId, record.id), exact: true }),
      client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbFeedbackList(scope, kbId) }),
      client.invalidateQueries({ queryKey: aiWorkspaceKeys.taskFeedback(scope, record.task_id), exact: true }),
    ]);
  }
  async function submit(values: FeedbackReviewValues) {
    if (needsConfirmation || !readVerified || !guard.isCurrent()) return;
    let payload;
    try { payload = reviewValuesToRequest(values, snapshot, expectedRevision); }
    catch (error) { setNotice(error instanceof Error ? error.message : '请核对复核表单'); return; }
    if (!guard.beginWrite('sys:ai:kb:review')) return;
    try {
      const response = await save.mutateAsync(payload);
      if (!guard.isCurrent()) return;
      setExpectedRevision(response.feedback.revision);
      client.setQueryData(aiWorkspaceKeys.kbFeedback(scope, kbId, record.id), response);
      setNotice(response.feedback.status === 'accepted' ? '已采纳，可下载回归样本；尚未运行新的模型评测。' : '已忽略此评价。');
      await refreshRecord();
    } catch (error) {
      if (!guard.isCurrent()) return;
      if (error instanceof ApiError && error.code === 409) setConflict(true);
      else setNotice('提交未完成或结果待核对，复核草稿已保留；不会自动重新提交。');
      await refreshRecord();
    } finally { guard.finishWrite(); }
  }
  return <section className={styles.reviewForm}><h3>{record.review ? '更新复核结论' : '复核这条评价'}</h3>
    <Typography.Paragraph type="secondary">请核对历史片段后填写；勾选和采纳本身不代表答案已经通过质量评测。</Typography.Paragraph>
    {needsConfirmation && <FeedbackRevisionNotice record={record} expectedRevision={expectedRevision} refreshing={refreshing} readVerified={readVerified} allowUnchangedRevision
      onAcknowledge={revision => { if (!readVerified) return; setExpectedRevision(revision); setAcknowledgedReadFailure(readFailureVersion); setConflict(false); setNotice(''); }} />}
    {notice && <Alert showIcon type="info" title={notice} />}
    <Form form={form} layout="vertical" disabled={guard.writing} initialValues={{ status: record.status === 'dismissed' ? 'dismissed' : 'accepted',
      answerable: record.review?.answerable ?? true, expected_answer: record.review?.expected_answer ?? '', required_facts: record.review?.required_facts.join('\n') ?? '',
      evidence: record.review?.evidence ?? [], reason: record.review?.reason ?? '' }} onFinish={values => void submit(values)}
      onValuesChange={change => { if (change.answerable === false) form.setFieldsValue({ required_facts: '', evidence: [] }); }}>
      <Form.Item name="status" label="处理方式"><Radio.Group options={[{ value: 'accepted', label: '采纳为回归样本' }, { value: 'dismissed', label: '忽略评价' }]} /></Form.Item>
      {status === 'accepted' && <>
        <Form.Item name="answerable" label="现有资料能否回答？"><Radio.Group options={[{ value: true, label: '有依据，可以回答' }, { value: false, label: '资料不足' }]} /></Form.Item>
        <Form.Item name="expected_answer" label="预期答案" rules={[{ required: true, whitespace: true, message: '请填写预期答案' }]}><Input.TextArea rows={4} placeholder="填写正确答案；资料不足时填写预期的不足说明" /></Form.Item>
        {answerable && <>
          <Form.Item name="required_facts" label="必须包含的事实（每行一项）" extra="填写预期答案和至少一条摘录中都出现的关键词，仅作词法回归检查，不代表语义正确率。"
            rules={[{ required: true, whitespace: true, message: '请填写事实检查项' }]}><Input.TextArea rows={3} placeholder="例如：备份数据库\n切换流量" /></Form.Item>
          <Form.List name="evidence">{(fields, { add, remove }) => <>
            <Typography.Paragraph>依据摘录：从上方历史片段逐字复制，最多5项。</Typography.Paragraph>
            {fields.map(field => <div key={field.key} className={styles.evidence}>
              <Form.Item name={[field.name, 'source_index']} label="历史片段" rules={[{ required: true, message: '请选择历史片段' }]}>
                <Select options={snapshot.sources.map(source => ({ value: source.index, label: `[${source.index}] ${source.doc_title}` }))} /></Form.Item>
              <Form.Item name={[field.name, 'quote']} label="逐字摘录" rules={[{ required: true, whitespace: true, message: '请填写逐字摘录' }]}><Input.TextArea rows={2} /></Form.Item>
              <Button size="small" icon={<MinusCircleOutlined />} onClick={() => remove(field.name)}>移除此摘录</Button>
            </div>)}
            <Button icon={<PlusOutlined />} disabled={fields.length >= 5 || guard.writing || !snapshot.sources.length} onClick={() => add({ source_index: snapshot.sources[0]?.index, quote: '' })}>添加依据摘录</Button>
          </>}</Form.List>
        </>}
      </>}
      <Form.Item name="reason" label="复核说明" rules={[{ required: true, whitespace: true, message: '请填写复核说明' }]}><Input.TextArea rows={3} placeholder="说明采纳、纠正或忽略的原因" /></Form.Item>
      <Button type="primary" htmlType="submit" loading={save.isPending} disabled={guard.writing || needsConfirmation || !readVerified || !guard.can('sys:ai:kb:review')}>{status === 'accepted' ? '保存并采纳' : '保存忽略结论'}</Button>
    </Form>
  </section>;
}
