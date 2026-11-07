import { useEffect, useRef, useState } from 'react';
import { Button, Card, Empty, Form, Input, Pagination, Select, Space, Tabs, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import type { AiListTasksReq, AiTaskStatus } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, taskListOptions, workflowListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { TaskDetail } from './TaskDetail';
import { correctTaskPageParams, parseTaskId, taskPollInterval, taskStatuses, taskStepLabel } from './taskState';
import styles from './tasks.module.css';

export default function AITasks() {
  return <AIWorkspaceAccess code="sys:ai:task:list" title="运行记录">{scope => <AITaskSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AITaskSession({ scope }: { scope: AiWorkspaceScope }) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:task:list');
  const client = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [params, setParams] = useState<AiListTasksReq>({ page: 1, page_size: 20 });
  const incomingTask = searchParams.get('task_id');
  const [lastTaskLink, setLastTaskLink] = useState(incomingTask);
  const [selectedId, setSelectedId] = useState<number | null>(() => parseTaskId(incomingTask));
  const [filterForm] = Form.useForm<{ workflow_id?: number; query?: string }>();
  const [workflowSearch, setWorkflowSearch] = useState('');
  const [workflowPage, setWorkflowPage] = useState(1);
  const workflows = useQuery({ ...workflowListOptions(scope, { page: workflowPage, page_size: 20, query: workflowSearch }), enabled: guard.can('sys:ai:workflow:list') });
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

  return <Card className={styles.workspace} title={<div><span>运行记录</span><p className={styles.subtitle}>查看分析结果，处理需要你确认的事项</p></div>}
    extra={<Button icon={<ReloadOutlined />} disabled={guard.writing} onClick={refreshTasks}>刷新</Button>}>
    <Tabs activeKey={params.status ?? 'all'} onChange={key => setParams(current => ({ ...current, page: 1, status: key === 'all' ? undefined : key as AiTaskStatus }))}
      items={[{ key: 'all', label: '全部', disabled: guard.writing }, ...Object.entries(taskStatuses).map(([key, value]) => ({ key, label: value.label, disabled: guard.writing }))]} />
    <Form form={filterForm} layout="inline" className={styles.filters} onFinish={values => {
      pollStartedAt.current = Date.now();
      setParams(current => ({ ...current, page: 1, workflow_id: values.workflow_id ?? undefined, query: values.query?.trim() || undefined }));
    }}>
      <Form.Item name="query"><Input allowClear aria-label="搜索运行名称或问题" placeholder="按名称、问题查找" disabled={guard.writing} /></Form.Item>
      <Form.Item name="workflow_id"><Select className={styles.workflowSelect} aria-label="按流程名称筛选" allowClear placeholder="全部流程" disabled={guard.writing || !guard.can('sys:ai:workflow:list')} loading={workflows.isFetching}
        showSearch={{ filterOption: false, onSearch: query => { setWorkflowSearch(query); setWorkflowPage(1); } }} options={workflows.data?.list.map(workflow => ({ value: workflow.id, label: workflow.name }))}
        popupRender={menu => <>{menu}{(workflows.data?.total ?? 0) > 20 && <Pagination simple size="small" current={workflowPage} pageSize={20} total={workflows.data?.total} onChange={setWorkflowPage} />}</>} /></Form.Item>
      <Form.Item><Space><Button htmlType="submit" disabled={guard.writing}>筛选</Button><Button disabled={guard.writing} onClick={() => {
        filterForm.resetFields(); setParams(current => ({ ...current, page: 1, workflow_id: undefined, query: undefined }));
      }}>重置</Button></Space></Form.Item>
    </Form>
    <QueryState isPending={tasks.isPending} error={tasks.error} onRetry={() => tasks.refetch()} />
    {!tasks.isPending && !tasks.error && !tasks.data?.list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无符合条件的任务" />}
    <div className={styles.taskLayout}><div className={styles.taskList} aria-label="我的运行记录">
      {!tasks.error && tasks.data?.list.map(task => <button key={task.id} type="button" disabled={guard.writing} aria-pressed={selectedId === task.id} className={`${styles.taskItem} ${selectedId === task.id ? styles.selectedTask : ''}`} onClick={() => selectTask(task.id)}>
        <Tag color={taskStatuses[task.status]?.color}>{taskStatuses[task.status]?.label ?? task.status}</Tag><strong>{task.title || `运行记录 #${task.id}`}</strong>
        {task.input_summary && <span className={styles.taskSummary}>{task.input_summary}</span>}<span className={styles.note}>{task.workflow_name || (task.workflow_id ? `流程 #${task.workflow_id}` : '一次性分析')} · {formatTime(task.created_at)}</span><span className={styles.note}>{taskStepLabel(task)}</span>
      </button>)}
    </div><div className={styles.resultWorkspace}>{selectedId !== null ? <TaskDetail key={selectedId} inline scope={scope} taskId={selectedId} guard={guard} metadata={tasks.error ? undefined : tasks.data?.list.find(task => task.id === selectedId)} metadataUpdatedAt={tasks.dataUpdatedAt} onClose={() => selectTask(null)} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="选择一条运行记录，查看结果与参考来源" />}</div></div>
    <div className={styles.footer}><span>仅显示本人任务 · 自动刷新最多持续 2 分钟</span>
      <Pagination current={params.page} pageSize={params.page_size} total={tasks.data?.total ?? 0} showSizeChanger pageSizeOptions={[10, 20, 50, 100]} size="small"
        disabled={guard.writing} showTotal={total => `共 ${total} 条`} onChange={(page, pageSize) => setParams(current => ({ ...current, page, page_size: pageSize }))} />
    </div>
  </Card>;
}
