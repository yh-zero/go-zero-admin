import { useEffect, useRef, useState } from 'react';
import { Button, Card, Empty, Form, InputNumber, Pagination, Space, Tabs, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import type { AiListTasksReq, AiTaskStatus } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, taskListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { TaskDetail } from './TaskDetail';
import { correctTaskPageParams, parseTaskId, taskPollInterval, taskStatuses, taskStepLabel } from './taskState';
import styles from './tasks.module.css';

export default function AITasks() {
  return <AIWorkspaceAccess code="sys:ai:task:list" title="任务中心">{scope => <AITaskSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AITaskSession({ scope }: { scope: AiWorkspaceScope }) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:task:list');
  const client = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [params, setParams] = useState<AiListTasksReq>({ page: 1, page_size: 20 });
  const incomingTask = searchParams.get('task_id');
  const [lastTaskLink, setLastTaskLink] = useState(incomingTask);
  const [selectedId, setSelectedId] = useState<number | null>(() => parseTaskId(incomingTask));
  const [filterForm] = Form.useForm<{ workflow_id?: number }>();
  const pollStartedAt = useRef(0);
  const tasks = useQuery({
    ...taskListOptions(scope, params),
    enabled: guard.can('sys:ai:task:list'),
    refetchInterval: query => {
      const active = query.state.data?.list.find(task => task.status === 'pending' || task.status === 'running');
      return taskPollInterval(active?.status, !guard.writing && guard.isCurrent(), Date.now() - pollStartedAt.current, query.state.fetchFailureCount);
    },
    refetchIntervalInBackground: false,
  });

  // 路由可在页面保持挂载时带入新任务，立即调整选择以防旧详情闪回。
  if (incomingTask !== lastTaskLink) {
    setLastTaskLink(incomingTask);
    if (incomingTask !== null) setSelectedId(parseTaskId(incomingTask));
  }

  useEffect(() => { pollStartedAt.current = Date.now(); }, [params]);
  const total = tasks.data?.total;
  const { isSuccess, isFetching } = tasks;
  const isCurrent = guard.isCurrent;
  useEffect(() => {
    if (!isCurrent() || !isSuccess || isFetching || total === undefined) return;
    // 只允许这次已完成查询校正自己的页码，旧筛选响应不能改写新筛选。
    // oxlint-disable-next-line react/set-state-in-effect
    setParams(current => correctTaskPageParams(current, params, { isSuccess, isFetching, total }));
  }, [isCurrent, params, isSuccess, isFetching, total]);
  useEffect(() => {
    if (!searchParams.has('task_id')) return;
    const next = new URLSearchParams(searchParams);
    next.delete('task_id');
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams, guard]);

  function selectTask(id: number | null) {
    if (guard.writing || !guard.isCurrent()) return;
    if (selectedId !== null && selectedId !== id) {
      void client.cancelQueries({ queryKey: aiWorkspaceKeys.task(scope, selectedId) });
    }
    if (guard.isCurrent()) setSelectedId(id);
  }

  function refreshTasks() {
    if (!guard.isCurrent() || guard.writing) return;
    pollStartedAt.current = Date.now();
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.tasks(scope) });
  }

  return <Card className={styles.workspace} title={<div><span>任务中心</span><p className={styles.subtitle}>查看本人任务的执行状态、结果与待确认事项</p></div>}
    extra={<Button icon={<ReloadOutlined />} disabled={guard.writing} onClick={refreshTasks}>刷新</Button>}>
    <Tabs activeKey={params.status ?? 'all'} onChange={key => setParams(current => ({ ...current, page: 1, status: key === 'all' ? undefined : key as AiTaskStatus }))}
      items={[{ key: 'all', label: '全部', disabled: guard.writing }, ...Object.entries(taskStatuses).map(([key, value]) => ({ key, label: value.label, disabled: guard.writing }))]} />
    <Form form={filterForm} layout="inline" className={styles.filters} onFinish={values => {
      pollStartedAt.current = Date.now();
      setParams(current => ({ ...current, page: 1, workflow_id: values.workflow_id ?? undefined }));
    }}>
      <Form.Item name="workflow_id" label="工作流 ID" rules={[{ validator: (_, value: number | null | undefined) => value == null || Number.isSafeInteger(value) && value > 0
        ? Promise.resolve() : Promise.reject(new Error('请输入有效的工作流 ID')) }]}><InputNumber min={1} max={Number.MAX_SAFE_INTEGER} precision={0} placeholder="全部工作流" disabled={guard.writing} /></Form.Item>
      <Form.Item><Space><Button htmlType="submit" disabled={guard.writing}>筛选</Button><Button disabled={guard.writing} onClick={() => {
        filterForm.resetFields(); setParams(current => ({ ...current, page: 1, workflow_id: undefined }));
      }}>重置</Button></Space></Form.Item>
    </Form>
    <QueryState isPending={tasks.isPending} error={tasks.error} onRetry={() => tasks.refetch()} />
    {!tasks.isPending && !tasks.error && !tasks.data?.list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无符合条件的任务" />}
    {!tasks.error && !!tasks.data?.list.length && <div className={styles.tableWrap}><table className={styles.table} aria-label="我的任务列表">
      <thead><tr><th>任务</th><th>工作流</th><th>状态</th><th>当前步骤</th><th className={styles.optional}>创建时间</th><th>操作</th></tr></thead>
      <tbody>{tasks.data.list.map(task => <tr key={task.id}>
        <td className={styles.name}><Button type="link" disabled={guard.writing} onClick={() => void selectTask(task.id)}>任务 #{task.id}</Button><div className={styles.note}>由我发起</div></td>
        <td>#{task.workflow_id}</td><td><Tag color={taskStatuses[task.status]?.color}>{taskStatuses[task.status]?.label ?? task.status}</Tag></td>
        <td>{taskStepLabel(task)}</td><td className={styles.optional}>{formatTime(task.created_at)}</td>
        <td className={styles.actions}><Button type="link" disabled={guard.writing} onClick={() => void selectTask(task.id)}>{task.status === 'pending_approval' ? '去确认' : '查看详情'}</Button></td>
      </tr>)}</tbody>
    </table></div>}
    <div className={styles.footer}><span>仅显示本人任务 · 自动刷新最多持续 2 分钟</span>
      <Pagination current={params.page} pageSize={params.page_size} total={tasks.data?.total ?? 0} showSizeChanger pageSizeOptions={[10, 20, 50, 100]} size="small"
        disabled={guard.writing} showTotal={total => `共 ${total} 条`} onChange={(page, pageSize) => setParams(current => ({ ...current, page, page_size: pageSize }))} />
    </div>
    {selectedId !== null && <TaskDetail key={selectedId} scope={scope} taskId={selectedId} guard={guard} onClose={() => void selectTask(null)} />}
  </Card>;
}
