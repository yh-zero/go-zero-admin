import { useEffect, useRef, useState } from 'react';
import { Button, Card, Dropdown, Empty, Form, Input, Pagination, Select, Space, Tabs, Tag } from 'antd';
import { DownOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import type { AiListTasksReq, AiTaskStatus } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { AsyncContent } from '../../components/AsyncContent';
import { formatTime } from '../../utils/format';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, taskDetailOptions, taskListOptions, workflowListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { isTerminalTask } from './taskComparisonData';
import { correctTaskPageParams, parseTaskId, taskPollInterval, taskStatuses, taskStepLabel } from './taskState';
import styles from './tasks.module.css';

const primaryStatuses = ['pending_approval', 'failed'] as const;
const moreStatuses = ['pending', 'running', 'succeeded', 'rejected', 'cancelled'] as const;
const loadComparison = () => import('./TaskComparison').then(module => module.TaskComparison);
const loadDetail = () => import('./TaskDetail').then(module => module.TaskDetail);

export default function AITasks() {
  return <AIWorkspaceAccess code="sys:ai:task:list" title="运行记录">{scope => <AITaskSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AITaskSession({ scope }: { scope: AiWorkspaceScope }) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:task:list');
  const client = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [params, setParams] = useState<AiListTasksReq>({ page: 1, page_size: 20 });
  const selectedId = parseTaskId(searchParams.get('task_id'));
  const [comparisonId, setComparisonId] = useState<number | null>(null);
  // 详情组件负责读取；此观察器仅订阅同一缓存，入口不额外发送请求。
  const selectedDetail = useQuery({ ...taskDetailOptions(scope, selectedId ?? 0), enabled: false });
  const [filterForm] = Form.useForm<{ workflow_id?: number; query?: string }>();
  const [workflowSearch, setWorkflowSearch] = useState('');
  const [workflowInput, setWorkflowInput] = useState('');
  const [workflowFilterOpen, setWorkflowFilterOpen] = useState(false);
  const [workflowPage, setWorkflowPage] = useState(1);
  useEffect(() => {
    if (!workflowFilterOpen || workflowInput.trim() === workflowSearch) return;
    const timer = window.setTimeout(() => { setWorkflowSearch(workflowInput.trim()); setWorkflowPage(1); }, 250);
    return () => window.clearTimeout(timer);
  }, [workflowFilterOpen, workflowInput, workflowSearch]);
  const workflows = useQuery({ ...workflowListOptions(scope, { page: workflowPage, page_size: 20, query: workflowSearch }), enabled: workflowFilterOpen && guard.can('sys:ai:workflow:list') });
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
  function selectTask(id: number | null) {
    if (guard.writing || !guard.isCurrent()) return;
    setComparisonId(null);
    if (selectedId !== null && selectedId !== id) {
      void client.cancelQueries({ queryKey: aiWorkspaceKeys.task(scope, selectedId) });
    }
    const next = new URLSearchParams(searchParams);
    if (id === null) next.delete('task_id');
    else next.set('task_id', String(id));
    if (selectedId !== id) setSearchParams(next);
  }

  function refreshTasks() {
    if (!guard.isCurrent() || guard.writing) return;
    pollStartedAt.current = Date.now();
    void client.invalidateQueries({ queryKey: aiWorkspaceKeys.tasks(scope) });
  }

  function changeStatus(status?: AiTaskStatus) {
    if (guard.writing || !guard.isCurrent()) return;
    setParams(current => ({ ...current, page: 1, status }));
  }
  const moreStatusSelected = moreStatuses.some(status => status === params.status);

  return <Card className={styles.workspace} title={<div><span>运行记录</span><p className={styles.subtitle}>查看分析结果，处理需要你确认的事项</p></div>}
    extra={<Button icon={<ReloadOutlined />} disabled={guard.writing} onClick={refreshTasks}>刷新</Button>}>
    <Tabs activeKey={moreStatusSelected ? 'more' : params.status ?? 'all'} onChange={key => changeStatus(key === 'all' ? undefined : key as AiTaskStatus)}
      items={[{ key: 'all', label: '全部', disabled: guard.writing }, ...primaryStatuses.map(key => ({ key, label: taskStatuses[key].label, disabled: guard.writing }))]}
      tabBarExtraContent={<Dropdown trigger={['click']} disabled={guard.writing} menu={{ selectedKeys: params.status ? [params.status] : [], items: moreStatuses.map(key => ({ key, label: taskStatuses[key].label })), onClick: ({ key }) => changeStatus(key as AiTaskStatus) }}>
        <Button type={moreStatusSelected ? 'primary' : 'text'} disabled={guard.writing}>更多状态{moreStatusSelected ? `：${taskStatuses[params.status!].label}` : ''}<DownOutlined /></Button>
      </Dropdown>} />
    <Form form={filterForm} layout="inline" className={styles.filters} onFinish={values => {
      pollStartedAt.current = Date.now();
      setParams(current => ({ ...current, page: 1, workflow_id: values.workflow_id ?? undefined, query: values.query?.trim() || undefined }));
    }}>
      <Form.Item name="query"><Input allowClear aria-label="搜索运行名称或问题" placeholder="按名称、问题查找" disabled={guard.writing} /></Form.Item>
      <Form.Item name="workflow_id"><Select className={styles.workflowSelect} aria-label="按流程名称筛选" allowClear placeholder="全部流程" disabled={guard.writing || !guard.can('sys:ai:workflow:list')} loading={workflows.isFetching}
        onOpenChange={setWorkflowFilterOpen} showSearch={{ filterOption: false, onSearch: setWorkflowInput }} options={workflows.data?.list.map(workflow => ({ value: workflow.id, label: workflow.name }))}
        popupRender={menu => <>{menu}{(workflows.data?.total ?? 0) > 20 && <Pagination simple size="small" current={workflowPage} pageSize={20} total={workflows.data?.total} onChange={setWorkflowPage} />}</>} /></Form.Item>
      <Form.Item><Space><Button htmlType="submit" disabled={guard.writing}>筛选</Button><Button disabled={guard.writing} onClick={() => {
        filterForm.resetFields(); setParams(current => ({ ...current, page: 1, workflow_id: undefined, query: undefined }));
      }}>重置</Button></Space></Form.Item>
    </Form>
    <QueryState isPending={tasks.isPending} error={tasks.error} onRetry={() => tasks.refetch()} />
    {!tasks.isPending && !tasks.error && !tasks.data?.list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无符合条件的运行记录" />}
    <div className={styles.taskLayout}><div className={styles.taskList} aria-label="我的运行记录">
      {!tasks.error && tasks.data?.list.map(task => <button key={task.id} type="button" disabled={guard.writing} aria-pressed={selectedId === task.id} className={`${styles.taskItem} ${selectedId === task.id ? styles.selectedTask : ''}`} onClick={() => selectTask(task.id)}>
        <Tag color={taskStatuses[task.status]?.color}>{taskStatuses[task.status]?.label ?? task.status}</Tag><strong>{task.title || `运行记录 #${task.id}`}</strong>
        {task.input_summary && <span className={styles.taskSummary}>{task.input_summary}</span>}<span className={styles.note}>{task.workflow_name || (task.workflow_id ? `流程 #${task.workflow_id}` : '一次性分析')} · {formatTime(task.created_at)}</span>{!isTerminalTask(task.status) && <span className={styles.note}>{taskStepLabel(task)}</span>}
      </button>)}
    </div><div className={styles.resultWorkspace}>{selectedId !== null ? <AsyncContent key={selectedId} load={loadDetail} componentProps={{ inline: true, scope, taskId: selectedId, guard, metadata: tasks.error ? undefined : tasks.data?.list.find(task => task.id === selectedId), metadataUpdatedAt: tasks.dataUpdatedAt, onClose: () => selectTask(null),
      comparisonDisabled: guard.writing || !guard.can('sys:ai:task:list') || !selectedDetail.data || selectedDetail.isError || selectedDetail.isFetching || !isTerminalTask(selectedDetail.data.task.status), onCompare: () => { if (guard.isCurrent()) setComparisonId(selectedId); } }} /> : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="选择一条运行记录，查看结果与参考来源" />}</div></div>
    <div className={styles.footer}><span>仅显示本人运行记录 · 自动刷新最多持续 2 分钟</span>
      <Pagination current={params.page} pageSize={params.page_size} total={tasks.data?.total ?? 0} showSizeChanger pageSizeOptions={[10, 20, 50, 100]} size="small"
        disabled={guard.writing} showTotal={total => `共 ${total} 条`} onChange={(page, pageSize) => setParams(current => ({ ...current, page, page_size: pageSize }))} />
    </div>
    {comparisonId !== null && comparisonId === selectedId && <AsyncContent key={comparisonId} load={loadComparison} componentProps={{ scope, baseId: comparisonId, guard, onClose: () => setComparisonId(null) }} />}
  </Card>;
}
