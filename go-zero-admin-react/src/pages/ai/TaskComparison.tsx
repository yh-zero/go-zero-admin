import { useState } from 'react';
import { Alert, Button, Drawer, Empty, Input, Pagination, Select, Space, Table, Tag, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import type { AiListTasksReq, AiTaskInfo, AiWorkflowStepInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { taskDetailOptions, taskListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { TaskDetailContent } from './TaskDetail';
import { formatTaskValue, taskStatuses } from './taskState';
import { isTerminalTask, pairTaskSteps, recordedElapsed } from './taskComparisonData';
import styles from './taskComparison.module.css';

interface Props { scope: AiWorkspaceScope; baseId: number; guard: ReturnType<typeof useAiWorkspaceGuard>; onClose: () => void }

export function TaskComparison({ scope, baseId, guard, onClose }: Props) {
  const [params, setParams] = useState<AiListTasksReq>({ page: 1, page_size: 10, status: 'succeeded' });
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const canRead = guard.can('sys:ai:task:list');
  const list = useQuery({ ...taskListOptions(scope, params), enabled: canRead && selectedId === null });
  const left = useQuery({ ...taskDetailOptions(scope, baseId), enabled: canRead });
  const right = useQuery({ ...taskDetailOptions(scope, selectedId ?? 0), enabled: canRead && selectedId !== null });
  const ready = canRead && left.isSuccess && !left.isFetching && !left.error && right.isSuccess && !right.isFetching && !right.error
    && left.data.task.id === baseId && right.data.task.id === selectedId;
  return <Drawer open title={`对比运行 · 左侧 #${baseId}`} size="large" styles={{ wrapper: { width: 'min(1100px, 100vw)' } }} onClose={onClose} extra={<Button onClick={onClose}>关闭对比</Button>}>
    <Space orientation="vertical" className={styles.workspace} size="middle">
      {!selectedId ? <>
        <Typography.Paragraph>选择另一条本人终态运行。状态分别筛选，页码与总数使用服务端返回值。</Typography.Paragraph>
        <div className={styles.toolbar}><Input.Search className={styles.search} allowClear aria-label="搜索另一条运行" placeholder="搜索名称或问题" maxLength={128}
          onSearch={query => { if (guard.isCurrent()) setParams(current => ({ ...current, page: 1, query: query.trim() || undefined })); }} />
          <Select className={styles.status} aria-label="筛选对照运行状态" value={params.status} options={['succeeded', 'failed', 'rejected', 'cancelled'].map(value => ({ value, label: taskStatuses[value as keyof typeof taskStatuses].label }))}
            onChange={status => { if (guard.isCurrent()) setParams(current => ({ ...current, page: 1, status })); }} />
        </div>
        <QueryState isPending={list.isPending} error={list.error} onRetry={() => list.refetch()} />
        {canRead && list.data && !list.error && <><Table<AiTaskInfo> rowKey="id" size="small" pagination={false} scroll={{ x: 650 }} loading={list.isFetching} dataSource={list.data.list} columns={[
          { title: '运行', render: (_, task) => <span>{task.title || `运行 #${task.id}`}<br /><Typography.Text type="secondary">#{task.id} · {task.workflow_name || '历史流程'}</Typography.Text></span> },
          { title: '状态', render: (_, task) => <Tag>{taskStatuses[task.status]?.label ?? task.status}</Tag> },
          { title: '创建时间', dataIndex: 'created_at', render: value => formatTime(value) },
          { title: '操作', render: (_, task) => <Button size="small" disabled={list.isFetching || task.id === baseId || !Number.isSafeInteger(task.id) || task.id < 1 || !isTerminalTask(task.status)}
            onClick={() => { if (guard.isCurrent()) setSelectedId(task.id); }}>选择运行 #{task.id}</Button> },
        ]} /><Pagination current={params.page} pageSize={params.page_size} total={list.data.total} showSizeChanger={false} onChange={page => { if (guard.isCurrent()) setParams(current => ({ ...current, page })); }} /></>}
      </> : <Space wrap><Typography.Text>左侧 #{baseId} · 右侧 #{selectedId}</Typography.Text><Button onClick={() => setSelectedId(null)}>更换另一条运行</Button>
        <Button disabled={!canRead || left.isFetching || right.isFetching} onClick={() => { if (guard.isCurrent()) { void left.refetch(); void right.refetch(); } }}>重新加载对照记录</Button></Space>}
      {selectedId && <>
        <QueryState isPending={left.isPending || left.isFetching} error={left.error} onRetry={() => left.refetch()} />
        <QueryState isPending={right.isPending || right.isFetching} error={right.error} onRetry={() => right.refetch()} />
        {ready && <TaskComparisonContent left={left.data.task} right={right.data.task} />}
      </>}
    </Space>
  </Drawer>;
}

function StepSide({ label, step }: { label: string; step?: AiWorkflowStepInfo }) {
  if (!step) return <section className={styles.panel}><h4>{label}</h4><p>该次未记录此步骤</p></section>;
  const elapsed = recordedElapsed(step.started_at, step.completed_at);
  return <section className={styles.panel}><h4>{label} · 第 {step.index + 1} 步 · {step.name}</h4><Space wrap><Tag>{step.tool}</Tag><Tag>{taskStatuses[step.status]?.label ?? step.status}</Tag></Space>
    <p>耗时：{elapsed === null ? '未记录' : `${elapsed} 毫秒`}</p>
    {!step.usage && <p>用量：未记录</p>}
    <details><summary>核对本步实际输入和输出</summary><h5>实际输入</h5><pre className={styles.record}>{step.input ? formatTaskValue(step.input) : '未记录'}</pre>
      <h5>步骤输出原文</h5><pre className={styles.record}>{step.output ? formatTaskValue(step.output) : '未记录'}</pre>
      {step.error && <Alert type="warning" title={step.error} />}
    </details>
  </section>;
}

export function TaskComparisonContent({ left, right }: { left: AiTaskInfo; right: AiTaskInfo }) {
  if (left.id === right.id || !isTerminalTask(left.status) || !isTerminalTask(right.status)) return <Alert type="warning" title="仅可对照两条不同的终态运行" />;
  const paired = pairTaskSteps(left, right);
  return <div className={styles.workspace}>
    <Alert type="info" showIcon title="差异不代表质量提升" description="这里只核对两次历史记录。问题、资料、配置和等待时间可能不同，文字变化与耗时差值不能单独证明质量或性能提升。" />
    <Typography.Paragraph className={styles.note}>{paired.explanation}</Typography.Paragraph>
    <div className={styles.columns}>{[{ label: '左侧', task: left }, { label: '右侧', task: right }].map(({ label, task }) => {
      const elapsed = recordedElapsed(task.created_at, task.completed_at);
      return <section className={styles.panel} key={task.id}><h3>{label}运行 #{task.id}</h3><Tag>{taskStatuses[task.status]?.label ?? task.status}</Tag>
        <p>创建至结束（含等待）：{elapsed === null ? '未记录' : `${elapsed} 秒`}</p><h4>本次实际输入</h4><pre className={styles.record}>{task.input ? formatTaskValue(task.input) : '未记录'}</pre>
      </section>;
    })}</div>
    <h3>步骤变化</h3>
    {paired.rows.length === 0 ? <Empty description="未记录步骤；仍可查看下方历史结果。" /> : <div className={styles.steps}>{paired.rows.map(row => <section className={styles.step} key={row.key}>
      <Space wrap>{!row.left || !row.right ? <Tag>{row.left ? '仅左侧有记录' : '仅右侧有记录'}</Tag> : <>
        {row.moved && <Tag color="blue">执行位置变化</Tag>}{row.inputChanged && <Tag color="orange">输入文字有差异</Tag>}{row.outputChanged && <Tag color="orange">输出文字有差异</Tag>}
        {row.statusChanged && <Tag>状态不同</Tag>}{row.left.tool !== row.right.tool && <Tag>工具不同</Tag>}{!row.inputChanged && !row.outputChanged && <Tag>输入输出文字相同</Tag>}
      </>}</Space>
      <div className={styles.columns}><StepSide label="左侧" step={row.left} /><StepSide label="右侧" step={row.right} /></div>
    </section>)}</div>}
    <h3>结果与实际来源</h3><div className={styles.columns}><section className={styles.panel} aria-label="左侧历史结果"><TaskDetailContent task={left} /></section><section className={styles.panel} aria-label="右侧历史结果"><TaskDetailContent task={right} /></section></div>
  </div>;
}
