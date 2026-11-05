import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Drawer, Form, Input, Modal, Space, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { approveAiTask, rejectAiTask } from '../../api/ai';
import type { AiTaskInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, taskDetailOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { captureTaskDecision, formatTaskValue, isTaskDecisionCurrent, taskPollInterval, taskStatuses, taskStepLabel, validateRejectReason } from './taskState';
import type { TaskDecision } from './taskState';
import styles from './tasks.module.css';

interface TaskDetailProps {
  scope: AiWorkspaceScope;
  taskId: number;
  guard: ReturnType<typeof useAiWorkspaceGuard>;
  onClose: () => void;
}

export function TaskDetail({ scope, taskId, guard, onClose }: TaskDetailProps) {
  const client = useQueryClient();
  const [decision, setDecision] = useState<TaskDecision | null>(null);
  const [feedback, setFeedback] = useState<string>();
  const [form] = Form.useForm<{ reason: string }>();
  const mounted = useRef(true);
  const pollStartedAt = useRef(0);
  const details = useQuery({
    ...taskDetailOptions(scope, taskId),
    enabled: guard.can('sys:ai:task:list'),
    refetchInterval: query => taskPollInterval(query.state.data?.task.status, !guard.writing && !decision && guard.isCurrent(), Date.now() - pollStartedAt.current, query.state.fetchFailureCount),
    refetchIntervalInBackground: false,
  });
  const mutation = useMutation({ mutationFn: async ({ snapshot, reason }: { snapshot: TaskDecision; reason?: string }) => {
    if (snapshot.action === 'approve') return approveAiTask(snapshot.id, { expected_step: snapshot.expected_step });
    return rejectAiTask(snapshot.id, { expected_step: snapshot.expected_step, reason: reason!.trim() });
  }, retry: false });

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
    form.resetFields();
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
  return <>
    <Drawer open title={`任务 #${taskId}`} size={620} className={styles.drawer} onClose={() => { if (!guard.writing) onClose(); }}
      closable={!guard.writing} maskClosable={!guard.writing && !decision} keyboard={!guard.writing && !decision}
      extra={<Button icon={<ReloadOutlined />} aria-label="刷新任务详情" disabled={guard.writing || !!decision} onClick={() => {
        if (!current()) return; pollStartedAt.current = Date.now(); void details.refetch();
      }} />}
      footer={<Space wrap className={styles.drawerActions}><Button disabled={guard.writing} onClick={onClose}>关闭</Button>
        {approvalAvailable && guard.can('sys:ai:task:reject') && <Button danger disabled={guard.writing} onClick={() => openDecision('reject')}>拒绝</Button>}
        {approvalAvailable && guard.can('sys:ai:task:approve') && <Button type="primary" disabled={guard.writing} onClick={() => openDecision('approve')}>批准当前步骤</Button>}
      </Space>}>
      <QueryState isPending={details.isPending} error={details.error} onRetry={() => details.refetch()} />
      {feedback && <Alert className={styles.feedback} title={feedback} type="info" showIcon />}
      {task && !details.error && <TaskDetailContent task={task} />}
    </Drawer>
    <Modal open={!!decision} title={decision?.action === 'approve' ? '批准当前步骤' : '拒绝任务'} onOk={() => void submitDecision()}
      onCancel={() => { if (!guard.writing) setDecision(null); }} confirmLoading={guard.writing} closable={!guard.writing} maskClosable={!guard.writing} keyboard={!guard.writing}
      okText={decision?.action === 'approve' ? '确认批准' : '确认拒绝'} cancelText="取消" cancelButtonProps={{ disabled: guard.writing }} destroyOnHidden>
      <p>任务 #{decision?.id} · 第 {decision ? decision.expected_step + 1 : ''} 步</p>
      <p>{decision?.action === 'approve' ? '仅批准当前等待确认的步骤，批准后将继续执行。' : '拒绝后本次任务结束。'}</p>
      {decision?.action === 'reject' && <Form form={form} layout="vertical" preserve={false}>
        <Form.Item name="reason" label="拒绝原因" rules={[{ validator: (_, value: string | undefined) => {
          const error = validateRejectReason(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve();
        } }]}><Input.TextArea rows={4} disabled={guard.writing} placeholder="请填写拒绝原因（最多 512 个字符）" /></Form.Item>
      </Form>}
    </Modal>
  </>;
}

export function TaskDetailContent({ task }: { task: AiTaskInfo }) {
  return <div className={styles.detailContent}>
    <div className={styles.state}><Tag color={taskStatuses[task.status]?.color}>{taskStatuses[task.status]?.label ?? task.status}</Tag>
      <p>{task.status === 'pending_approval' ? '等待你确认当前步骤，该步骤尚未执行。' : '展示本次任务已记录的状态与结果。'}</p></div>
    <dl className={styles.facts}>
      <div><dt>工作流 ID</dt><dd>#{task.workflow_id}</dd></div><div><dt>当前步骤</dt><dd>{taskStepLabel(task)}</dd></div>
      <div><dt>创建时间</dt><dd>{formatTime(task.created_at)}</dd></div><div><dt>完成时间</dt><dd>{formatTime(task.completed_at)}</dd></div>
      <div><dt>审批范围</dt><dd>本人任务</dd></div><div><dt>审批人 ID</dt><dd>{task.approved_by || '—'}</dd></div>
    </dl>
    <h3>本次输入</h3><TaskRecordedValue value={task.input} empty="未提供输入" />
    <h3>执行结果</h3><TaskRecordedValue value={task.result} empty="尚无执行结果" />
    {task.error && <><h3>错误 / 拒绝原因</h3><pre className={styles.code}>{task.error}</pre></>}
    {task.status === 'pending_approval' && <Alert type="warning" showIcon title="批准后继续执行当前步骤；拒绝后本次任务结束。" />}
  </div>;
}

function TaskRecordedValue({ value, empty }: { value: string; empty: string }) {
  const formatted = value ? formatTaskValue(value) : empty;
  return <><pre className={styles.code}>{formatted}</pre>
    {value && value !== formatted && <details><summary>查看原文</summary><pre className={styles.code}>{value}</pre></details>}
  </>;
}
