import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { Alert, Button, Drawer, Dropdown, Form, Input, Modal, Space, Tag } from 'antd';
import { MoreOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { approveAiTask, cancelAiTask, rejectAiTask } from '../../api/ai';
import type { AiTaskInfo, AiWorkflowStepInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, taskDetailOptions, taskMetadataOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { captureTaskDecision, formatTaskValue, isTaskDecisionCurrent, taskPollInterval, taskStatuses, taskStepLabel, validateRejectReason } from './taskState';
import type { TaskDecision } from './taskState';
import { parseTaskAnalyses } from './workflowAnalysis';
import { WorkflowAnalysisResult } from './WorkflowAnalysisResult';
import type { TaskAnalysis } from './workflowAnalysis';
import { TaskAnalysisFeedback } from './TaskAnalysisFeedback';
import { CopyTextButton } from './SafeMarkdown';
import styles from './tasks.module.css';

interface TaskDetailProps {
  scope: AiWorkspaceScope;
  taskId: number;
  guard: ReturnType<typeof useAiWorkspaceGuard>;
  onClose: () => void;
  inline?: boolean;
  metadata?: AiTaskInfo;
  metadataUpdatedAt?: number;
  showSaveAction?: boolean;
  onCompare?: () => void;
  comparisonDisabled?: boolean;
}

export function TaskDetail({ scope, taskId, guard, onClose, inline = false, metadata, metadataUpdatedAt = 0, showSaveAction = true, onCompare, comparisonDisabled = true }: TaskDetailProps) {
  const client = useQueryClient();
  const navigate = useNavigate();
  const [decision, setDecision] = useState<TaskDecision | null>(null);
  const [feedback, setFeedback] = useState<string>();
  const [cancelConfirm, setCancelConfirm] = useState(false);
  const [form] = Form.useForm<{ reason: string }>();
  const mounted = useRef(true);
  const pollStartedAt = useRef(0);
  const lastMetadata = useRef('');
  const details = useQuery({
    ...taskDetailOptions(scope, taskId),
    enabled: guard.can('sys:ai:task:list'),
  });
  const metaOptions = taskMetadataOptions(scope, taskId, details.data?.task, metadata);
  const meta = useQuery({ ...metaOptions,
    enabled: metaOptions.enabled && guard.can('sys:ai:task:list'),
    refetchInterval: query => taskPollInterval(query.state.data?.task.status ?? details.data?.task.status, !guard.writing && !decision && !cancelConfirm && guard.isCurrent(), Date.now() - pollStartedAt.current, query.state.fetchFailureCount),
    refetchIntervalInBackground: false });
  const latest = metadata && metadataUpdatedAt > details.dataUpdatedAt ? metadata : meta.dataUpdatedAt > details.dataUpdatedAt ? meta.data?.task : undefined;
  useEffect(() => {
    if (!latest || !details.data || guard.writing || !guard.isCurrent()) return;
    const signature = `${latest.updated_at}:${latest.status}:${latest.next_step}`;
    if (lastMetadata.current === signature) return;
    lastMetadata.current = signature;
    const current = details.data.task;
    if (latest.status !== current.status || latest.updated_at !== current.updated_at || latest.next_step !== current.next_step) void client.invalidateQueries({ queryKey: aiWorkspaceKeys.task(scope, taskId), exact: true });
  }, [latest, details.data, guard, client, scope, taskId]);
  const mutation = useMutation({ mutationFn: async ({ snapshot, reason }: { snapshot: TaskDecision; reason?: string }) => {
    if (snapshot.action === 'approve') return approveAiTask(snapshot.id, { expected_step: snapshot.expected_step });
    return rejectAiTask(snapshot.id, { expected_step: snapshot.expected_step, reason: reason!.trim() });
  }, retry: false });
  const cancellation = useMutation({ mutationFn: () => cancelAiTask(taskId), retry: false });

  useEffect(() => {
    mounted.current = true;
    pollStartedAt.current = Date.now();
    return () => {
      mounted.current = false;
      void client.cancelQueries({ queryKey: aiWorkspaceKeys.task(scope, taskId) });
    };
  }, [client, scope, taskId]);

  function current() { return mounted.current && guard.isCurrent(); }

  function openDecision(action: TaskDecision['action']) {
    if (!current() || guard.writing || !details.data || details.error) return;
    const snapshot = captureTaskDecision(details.data.task, action);
    if (!snapshot) return;
    setFeedback(undefined);
    setDecision(snapshot);
  }

  async function submitDecision() {
    if (!decision || !current()) return;
    const snapshot = decision;
    let reason: string | undefined;
    if (snapshot.action === 'reject') {
      const values = await form.validateFields().catch(() => null);
      if (!values) return;
      reason = values.reason;
    }
    if (!current()) return;
    if (!isTaskDecisionCurrent(snapshot, details.data?.task)) {
      setFeedback('任务状态或当前步骤已变化，请刷新详情后重新确认。');
      setDecision(null);
      return;
    }
    if (!guard.beginWrite(`sys:ai:task:${snapshot.action}`)) return;
    try {
      await client.cancelQueries({ queryKey: aiWorkspaceKeys.tasks(scope) });
      await client.cancelQueries({ queryKey: aiWorkspaceKeys.task(scope, taskId) });
      if (!current() || !guard.can(`sys:ai:task:${snapshot.action}`)) return;
      await mutation.mutateAsync({ snapshot, reason });
      if (!current()) return;
      setFeedback('操作已提交，正在刷新任务状态。');
    } catch {
      if (current()) setFeedback('操作失败或结果尚未确认，请核对刷新后的任务状态；不会自动重新提交。');
    } finally {
      if (current()) {
        setDecision(null);
        pollStartedAt.current = Date.now();
        await Promise.allSettled([
          client.invalidateQueries({ queryKey: aiWorkspaceKeys.task(scope, taskId) }),
          client.invalidateQueries({ queryKey: aiWorkspaceKeys.tasks(scope) }),
        ]);
      }
      guard.finishWrite();
    }
  }

  const task = details.data?.task;
  const approvalAvailable = task?.status === 'pending_approval' && !details.error;
  const cancellable = task && ['pending', 'running', 'pending_approval'].includes(task.status) && !details.error;
  const pendingStep = task?.steps?.find(step => step.index === decision?.expected_step && step.status === 'pending_approval');
  async function submitCancel() {
    if (!current() || !guard.beginWrite('sys:ai:task:cancel')) return;
    try { await cancellation.mutateAsync(); if (current()) setFeedback('已取消后续执行；已发生的外部操作不会撤回。'); }
    catch { if (current()) setFeedback('取消结果待核对，请刷新任务状态；不会自动重新提交。'); }
    finally {
      if (current()) { setCancelConfirm(false); await Promise.allSettled([client.invalidateQueries({ queryKey: aiWorkspaceKeys.task(scope, taskId) }), client.invalidateQueries({ queryKey: aiWorkspaceKeys.tasks(scope) })]); }
      guard.finishWrite();
    }
  }
  const canSave = showSaveAction && task?.definition && guard.can('sys:ai:workflow:create') && guard.can('sys:ai:workflow:list') && guard.can('sys:ai:tool:list');
  const actions = <Space wrap className={styles.drawerActions}><Button disabled={guard.writing} onClick={onClose}>{inline ? '收起结果' : '关闭'}</Button>
    {(canSave || onCompare) && <Dropdown trigger={['click']} menu={{ items: [
      ...(canSave ? [{ key: 'save', label: task?.status === 'failed' ? '带入原问题' : '保存为常用流程', disabled: guard.writing }] : []),
      ...(onCompare ? [{ key: 'compare', label: '对比运行', disabled: comparisonDisabled || guard.writing }] : []),
    ], onClick: ({ key }) => {
      if (guard.writing || !current()) return;
      if (key === 'save' && canSave) navigate(`/ai/workflows/new?from_task=${taskId}`);
      if (key === 'compare' && !comparisonDisabled) onCompare?.();
    } }}><Button icon={<MoreOutlined />} disabled={guard.writing}>更多操作</Button></Dropdown>}
    {cancellable && guard.can('sys:ai:task:cancel') && <Button danger disabled={guard.writing} onClick={() => setCancelConfirm(true)}>取消任务</Button>}
    {approvalAvailable && guard.can('sys:ai:task:reject') && <Button danger disabled={guard.writing} onClick={() => openDecision('reject')}>拒绝</Button>}
    {approvalAvailable && guard.can('sys:ai:task:approve') && <Button type="primary" disabled={guard.writing} onClick={() => openDecision('approve')}>确认并继续</Button>}
  </Space>;
  const content = <>
    <QueryState isPending={details.isPending} error={details.error} onRetry={() => details.refetch()} />
    {feedback && <Alert className={styles.feedback} title={feedback} type="info" showIcon />}
    {task && !details.error && <TaskDetailContent task={task} renderFeedback={analysis => task.steps.some(step => step.index === analysis.step - 1 && step.tool === 'kb_analyze' && step.status === 'succeeded')
      ? <TaskAnalysisFeedback scope={scope} taskId={taskId} stepIndex={analysis.step - 1} guard={guard} /> : null} />}
  </>;
  return <>
    {inline ? <section className={styles.inlineDetail} aria-label="运行结果"><div className={styles.inlineActions}><Button icon={<ReloadOutlined />} disabled={guard.writing} onClick={() => { pollStartedAt.current = Date.now(); void details.refetch(); }}>刷新状态</Button>{actions}</div>{content}</section> : <Drawer open title={task?.title || `运行记录 #${taskId}`} size={820} className={styles.drawer} onClose={() => { if (!guard.writing) onClose(); }}
      closable={!guard.writing} maskClosable={!guard.writing && !decision && !cancelConfirm} keyboard={!guard.writing && !decision && !cancelConfirm}
      extra={<Button icon={<ReloadOutlined />} aria-label="刷新任务详情" disabled={guard.writing || !!decision} onClick={() => {
        if (!current()) return; pollStartedAt.current = Date.now(); void details.refetch();
      }} />}
      footer={actions}>{content}</Drawer>}
    <Modal open={!!decision} title={decision?.action === 'approve' ? '批准当前步骤' : '拒绝任务'} onOk={() => void submitDecision()}
      onCancel={() => { if (!guard.writing) setDecision(null); }} confirmLoading={guard.writing} closable={!guard.writing} mask={{ closable: !guard.writing }} keyboard={!guard.writing}
      okText={decision?.action === 'approve' ? '确认批准' : '确认拒绝'} cancelText="取消" cancelButtonProps={{ disabled: guard.writing }} destroyOnHidden>
      <p>任务 #{decision?.id} · 第 {decision ? decision.expected_step + 1 : ''} 步</p>
      <p>{decision?.action === 'approve' ? '仅批准当前等待确认的步骤，批准后将继续执行。' : '拒绝后本次任务结束。'}</p>
      {pendingStep ? <TaskStepRecord step={pendingStep} preview /> : decision?.action === 'approve' && <Alert type="warning" title="此历史任务没有可展示的步骤输入，批准前请核对本次快照与运行记录。" />}
      {decision?.action === 'reject' && <Form form={form} layout="vertical" preserve={false} clearOnDestroy>
        <Form.Item name="reason" label="拒绝原因" rules={[{ validator: (_, value: string | undefined) => {
          const error = validateRejectReason(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve();
        } }]}><Input.TextArea rows={4} disabled={guard.writing} placeholder="请填写拒绝原因（最多 512 个字符）" /></Form.Item>
      </Form>}
    </Modal>
    <Modal open={cancelConfirm} title="取消任务" okText="确认取消" okButtonProps={{ danger: true }} confirmLoading={guard.writing} onOk={() => void submitCancel()} onCancel={() => { if (!guard.writing) setCancelConfirm(false); }} closable={!guard.writing} mask={{ closable: !guard.writing }} keyboard={!guard.writing}>
      <p>取消后停止后续步骤，当前工具会收到取消信号。已经发生的模型调用或外部操作不会撤回，当前步骤结果可能未知。</p>
    </Modal>
  </>;
}

export function TaskDetailContent({ task, renderFeedback }: { task: AiTaskInfo; renderFeedback?: (analysis: TaskAnalysis) => ReactNode }) {
  const analyses = parseTaskAnalyses(task.result, task.next_step);
  return <div className={styles.detailContent}>
    {task.title && <h2>{task.title}</h2>}
    {task.input_summary && <p className={styles.question}>问题：{task.input_summary}</p>}
    <div className={styles.state}><Tag color={taskStatuses[task.status]?.color}>{taskStatuses[task.status]?.label ?? task.status}</Tag>
      <p>{task.status === 'pending_approval' ? '等待你确认当前步骤，该步骤尚未执行。' : '展示本次任务已记录的状态与结果。'}</p></div>
    <h3>执行结果 <CopyTextButton text={analyses.map(item => item.result.analysis).join('\n\n') || task.result} label="复制结果" /></h3>
    {analyses.map(analysis => <div key={analysis.step}><WorkflowAnalysisResult analysis={analysis} />{renderFeedback?.(analysis)}</div>)}
    {analyses.length ? <details><summary>原始执行记录</summary><pre className={styles.code}>{task.result}</pre></details>
      : <TaskRecordedValue value={task.result} empty={task.status === 'pending_approval' ? '确认后开始当前步骤' : '尚无执行结果'} />}
    {task.error && <Alert type="error" showIcon title="本次运行未完成" description={<pre className={styles.errorText}>{task.error}</pre>} />}
    <details className={styles.technicalDetails}><summary>执行过程与技术详情</summary>
    <dl className={styles.facts}>
      <div><dt>流程</dt><dd>{task.workflow_name || (task.workflow_id ? `流程 #${task.workflow_id}` : '一次性分析')}</dd></div><div><dt>当前步骤</dt><dd>{taskStepLabel(task)}</dd></div>
      <div><dt>创建时间</dt><dd>{formatTime(task.created_at)}</dd></div><div><dt>完成时间</dt><dd>{formatTime(task.completed_at)}</dd></div>
      <div><dt>最近更新</dt><dd>{formatTime(task.updated_at)}</dd></div>
      <div><dt>审批范围</dt><dd>本人任务</dd></div><div><dt>审批人 ID</dt><dd>{task.approved_by || '—'}</dd></div>
    </dl>
    <h3>本次输入</h3><TaskRecordedValue value={task.input} empty="未提供输入" />
    {!!task.steps?.length && <><h3>步骤执行过程</h3><ol className={styles.stepRecords}>{task.steps.map(step => <li key={step.id}><TaskStepRecord step={step} /></li>)}</ol></>}
    </details>
    {task.status === 'pending_approval' && <Alert type="warning" showIcon title="批准后继续执行当前步骤；拒绝后本次任务结束。" />}
  </div>;
}

function TaskStepRecord({ step, preview = false }: { step: AiWorkflowStepInfo; preview?: boolean }) {
  const elapsed = step.started_at > 0 && step.completed_at >= step.started_at ? `${step.completed_at - step.started_at} 毫秒` : '—';
  const usage = step.usage?.scope === 'chat'
    && [step.usage.prompt_tokens, step.usage.completion_tokens, step.usage.total_tokens].every(value => Number.isSafeInteger(value) && value >= 0)
    ? step.usage : undefined;
  const usageComplete = usage?.complete === true && usage.total_tokens === usage.prompt_tokens + usage.completion_tokens;
  return <section className={styles.stepRecord}>
    <strong>第 {step.index + 1} 步 · {step.name}</strong><p><Tag>{step.tool}</Tag><Tag color={taskStatuses[step.status]?.color}>{taskStatuses[step.status]?.label ?? step.status}</Tag>{!preview && <span>耗时：{elapsed}</span>}</p>
    {!preview && usage && <div aria-label="步骤聊天用量"><p>已知聊天用量：输入 {usage.prompt_tokens} · 输出 {usage.completion_tokens} · 合计 {usage.total_tokens} tokens；不含 Embedding。</p>
      {!usageComplete && <Alert type="warning" title="用量不完整（实际费用可能更高）" description="仅包含已知聊天用量，0 不代表免费。" />}
    </div>}
    <details open={preview || step.status === 'pending_approval'}><summary>本次实际输入</summary><pre className={styles.code}>{formatTaskValue(step.input)}</pre></details>
    {!preview && step.output && <details><summary>步骤输出 <CopyTextButton text={step.output} label="复制输出" /></summary><pre className={styles.code}>{formatTaskValue(step.output)}</pre></details>}
    {step.error && <Alert type="warning" title={step.error} />}
  </section>;
}

function TaskRecordedValue({ value, empty }: { value: string; empty: string }) {
  const formatted = value ? formatTaskValue(value) : empty;
  return <><pre className={styles.code}>{formatted}</pre>
    {value && value !== formatted && <details><summary>查看原文</summary><pre className={styles.code}>{value}</pre></details>}
  </>;
}
