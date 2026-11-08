import { useState } from 'react';
import { Alert, Button, Form, Input, Modal, Select, Space, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import type { AiKbFeedbackInfo, AiKbFeedbackKind } from '../../api/types';
import { ApiError } from '../../utils/response';
import { feedbackLabels, feedbackNoteError, reviewStatusLabels, taskFeedbackMutationOptions, taskFeedbackOptions } from './knowledgeFeedback';
import { FeedbackRevisionNotice } from './FeedbackRevisionNotice';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';

type Guard = ReturnType<typeof useAiWorkspaceGuard>;
interface Props { scope: AiWorkspaceScope; taskId: number; stepIndex: number; guard: Guard }

export function TaskAnalysisFeedback({ scope, taskId, stepIndex, guard }: Props) {
  const [open, setOpen] = useState(false);
  const current = useQuery({ ...taskFeedbackOptions(scope, taskId), enabled: guard.can('sys:ai:task:list') });
  const readFailureVersion = useQueryClient().getQueryState(aiWorkspaceKeys.taskFeedback(scope, taskId))?.errorUpdateCount ?? 0;
  const record = current.data?.list.find(item => item.step_index === stepIndex);
  return <section aria-label={`第 ${stepIndex + 1} 步分析反馈`}>
    <Space wrap>
      {guard.can('sys:ai:task:feedback') && <Button size="small" disabled={guard.writing || !current.data || current.isError} loading={current.isPending}
        onClick={() => { if (guard.isCurrent()) setOpen(true); }}>{record ? '修改评价' : '评价分析'}</Button>}
      {record && <><Typography.Text type="secondary">{feedbackLabels[record.feedback]} · {reviewStatusLabels[record.status]}</Typography.Text>
        {guard.can('sys:ai:kb:list') && <Link to={`/ai/knowledge/${record.kb_id}?tab=feedback`}>查看反馈与复核</Link>}</>}
    </Space>
    {current.isError && <Alert type="warning" title="当前评价未能读取，请重试后再编辑。" action={<Button size="small" onClick={() => void current.refetch()}>重试</Button>} />}
    {open && <AnalysisFeedbackDialog scope={scope} taskId={taskId} stepIndex={stepIndex} guard={guard} record={record}
      refreshing={current.isFetching} readVerified={current.isSuccess && !current.isFetching} readFailureVersion={readFailureVersion}
      onRefresh={() => current.refetch()} onClose={() => setOpen(false)} />}
  </section>;
}

function AnalysisFeedbackDialog({ scope, taskId, stepIndex, guard, record, refreshing, readVerified, readFailureVersion, onRefresh, onClose }: Props & {
  record?: AiKbFeedbackInfo; refreshing: boolean; readVerified: boolean; readFailureVersion: number;
  onRefresh: () => Promise<unknown>; onClose: () => void;
}) {
  const client = useQueryClient();
  const [form] = Form.useForm<{ feedback: AiKbFeedbackKind; note: string }>();
  const [expectedRevision, setExpectedRevision] = useState(record?.revision ?? 0);
  const [conflict, setConflict] = useState(false);
  const [acknowledgedReadFailure, setAcknowledgedReadFailure] = useState(readFailureVersion);
  const [notice, setNotice] = useState('');
  const save = useMutation(taskFeedbackMutationOptions(scope, taskId));
  const needsConfirmation = conflict || (record?.revision ?? 0) !== expectedRevision || readFailureVersion > acknowledgedReadFailure;

  async function refreshRecord() {
    await Promise.allSettled([
      client.invalidateQueries({ queryKey: aiWorkspaceKeys.taskFeedback(scope, taskId), exact: true }),
      ...(record ? [client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbFeedbackList(scope, record.kb_id) }),
        client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbFeedback(scope, record.kb_id, record.id), exact: true })] : []),
    ]);
  }
  async function submit(values: { feedback: AiKbFeedbackKind; note: string }) {
    if (needsConfirmation || !readVerified || !guard.beginWrite('sys:ai:task:feedback')) return;
    try {
      const response = await save.mutateAsync({ ...values, note: values.note ?? '', expected_revision: expectedRevision, step_index: stepIndex });
      if (!guard.isCurrent()) return;
      await client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbFeedbackList(scope, response.feedback.kb_id) });
      await client.invalidateQueries({ queryKey: aiWorkspaceKeys.kbFeedback(scope, response.feedback.kb_id, response.feedback.id), exact: true });
      await refreshRecord();
      if (guard.isCurrent()) onClose();
    } catch (error) {
      if (!guard.isCurrent()) return;
      if (error instanceof ApiError && error.code === 409) setConflict(true);
      else setNotice('提交未完成或结果待核对，草稿已保留。请核对刷新后的记录；不会自动重复提交。');
      await refreshRecord();
    } finally { guard.finishWrite(); }
  }

  return <Modal open title={`评价第 ${stepIndex + 1} 步分析`} okText="保存评价" cancelText="取消" confirmLoading={save.isPending}
    onOk={() => form.submit()} onCancel={() => { if (!guard.writing) onClose(); }}
    okButtonProps={{ disabled: guard.writing || needsConfirmation || !readVerified || !guard.can('sys:ai:task:feedback') }} cancelButtonProps={{ disabled: guard.writing }}
    mask={{ closable: !guard.writing }} closable={!guard.writing} keyboard={!guard.writing}>
    <Typography.Paragraph type="secondary">评价会进入该知识库的待复核列表。修改已复核评价会重新待复核，原分析结果保持不变。</Typography.Paragraph>
    {needsConfirmation && <FeedbackRevisionNotice record={record} expectedRevision={expectedRevision} refreshing={refreshing} readVerified={readVerified} allowUnchangedRevision allowEmptyRecord
      onAcknowledge={revision => { if (!readVerified) return; setExpectedRevision(revision); setAcknowledgedReadFailure(readFailureVersion); setConflict(false); setNotice(''); }} />}
    {!readVerified && <Alert type="warning" title="当前评价尚未核对，草稿已保留。请重新加载后再提交。"
      action={<Button disabled={guard.writing || refreshing} onClick={() => void onRefresh()}>重新加载评价</Button>} />}
    {notice && <Alert type="info" showIcon title={notice} />}
    <Form form={form} layout="vertical" initialValues={{ feedback: record?.feedback ?? 'helpful', note: record?.note ?? '' }} onFinish={values => void submit(values)} disabled={guard.writing}>
      <Form.Item name="feedback" label="评价" rules={[{ required: true, message: '请选择评价' }]}>
        <Select options={Object.entries(feedbackLabels).map(([value, label]) => ({ value, label }))} />
      </Form.Item>
      <Form.Item name="note" label="补充说明（可选）" rules={[{ validator: (_, value: string) => {
        const error = feedbackNoteError(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve();
      } }]}><Input.TextArea rows={3} placeholder="例如：遗漏了回滚条件，或者引用片段不能支持结论" /></Form.Item>
    </Form>
  </Modal>;
}
