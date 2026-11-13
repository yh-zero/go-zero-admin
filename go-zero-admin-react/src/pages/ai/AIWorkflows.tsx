import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, App, Button, Card, Drawer, Dropdown, Form, Input, Modal, Space, Table, Tag, Typography } from 'antd';
import { ApartmentOutlined, CopyOutlined, DeleteOutlined, EditOutlined, MoreOutlined, PlayCircleOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { deleteAiWorkflow, runAiWorkflow } from '../../api/ai';
import type { AiRunWorkflowReq, AiRunWorkflowResp, AiWorkflowInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { AsyncContent } from '../../components/AsyncContent';
import { correctPageParams } from '../../hooks/usePagedQuery';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { ApiError } from '../../utils/response';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, workflowDetailOptions, workflowListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { WorkflowDefinitionView, WorkflowRunResultNotice } from './WorkflowReadOnly';
import { prepareWorkflowRunInput, validateRunInput, workflowRunFields } from './workflowDefinition';
import { KnowledgeChoice } from './KnowledgeChoice';
import { clearWorkflowRunAttempt, readWorkflowRunAttempt, saveWorkflowRunAttempt, workflowRunAttemptKey } from './workflowRunAttempts';
import type { WorkflowRunAttempt } from './workflowRunAttempts';
import styles from './workflow.module.css';

const readCode = 'sys:ai:workflow:list';
const permission = (operation: string) => `sys:ai:workflow:${operation}`;
const loadHistory = () => import('./WorkflowHistory').then(module => module.WorkflowHistory);
const loadDetail = () => import('./TaskDetail').then(module => module.TaskDetail);
type Panel = { mode: 'view' | 'run'; id: number; key: string };

export default function AIWorkflows() {
  return <AIWorkspaceAccess code={readCode} title="常用流程">{scope => <AIWorkflowsSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AIWorkflowsSession({ scope }: { scope: AiWorkspaceScope }) {
  const client = useQueryClient();
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const guard = useAiWorkspaceGuard(scope, readCode);
  const { can, isCurrent, beginWrite, finishWrite, writing } = guard;
  const [pageParams, setPageParams] = useState<{ page: number; page_size: number; query?: string }>({ page: 1, page_size: 20 });
  const { page, page_size: pageSize } = pageParams;
  const [search, setSearch] = useState('');
  const [selectedTask, setSelectedTask] = useState<number | null>(null);
  const [panel, setPanel] = useState<Panel | null>(null);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<AiWorkflowInfo | null>(null);
  const [uncertain, setUncertain] = useState<'run' | null>(null);
  const [lastRunWithoutAccess, setLastRunWithoutAccess] = useState<AiRunWorkflowResp | null>(null);
  const [runAttempt, setRunAttempt] = useState<(AiRunWorkflowReq & { workflowId: number }) | null>(null);
  const [runError, setRunError] = useState('');
  const runAttempts = useRef(new Map<string, WorkflowRunAttempt>());
  const [runForm] = Form.useForm<{ input: string; question?: string; text?: string; kb_id?: number }>();
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const sessionId = scope[0];
  const attemptKey = (id: number) => workflowRunAttemptKey(sessionId, departmentId, id);
  const incomingTool = searchParams.get('tool');
  const requestedRun = searchParams.get('run');
  const canReadTools = can('sys:ai:tool:list');
  const canCreate = can(permission('create'));
  const canEdit = can(permission('edit'));
  const canReadTasks = can('sys:ai:task:list');
  const canRun = can(permission('run'));
  const consumedRun = useRef<string | null>(null);
  const restoreRunAttempt = useCallback((id: number) => {
    setUncertain(null); setRunAttempt(null); setRunError('');
    try {
      const stored = readWorkflowRunAttempt(sessionStorage, runAttempts.current, workflowRunAttemptKey(sessionId, departmentId, id), id);
      if (stored) { setRunAttempt(stored); setUncertain('run'); }
    } catch { setUncertain('run'); setRunError('浏览器无法恢复运行标识，请先到运行记录核对上次运行。'); }
  }, [sessionId, departmentId]);
  const list = useQuery(workflowListOptions(scope, pageParams));
  const completedTotal = list.isSuccess && !list.isFetching ? list.data.total : undefined;
  const detailId = panel?.id ?? 0;
  const detail = useQuery({ ...workflowDetailOptions(scope, detailId), enabled: detailId > 0, refetchOnMount: 'always' });
  const workflow = detail.data?.workflow;
  const remove = useMutation({ mutationFn: deleteAiWorkflow, retry: false });
  const run = useMutation({ mutationFn: ({ workflowId, ...data }: AiRunWorkflowReq & { workflowId: number }) => runAiWorkflow(workflowId, data), retry: false });
  let runFields = { question: false, text: false, kb_id: false };
  try { if (workflow) runFields = workflowRunFields(workflow.definition); } catch { /* 详情会展示原始定义；运行时服务端再次校验。 */ }

  useEffect(() => {
    if (completedTotal === undefined) return;
    // 远端删除会缩短列表；只修正返回本次总数的参数，不覆盖用户刚切换的新页。
    // oxlint-disable-next-line react/set-state-in-effect
    setPageParams(current => current === pageParams ? correctPageParams(current, completedTotal) : current);
  }, [completedTotal, pageParams]);

  useEffect(() => {
    if (incomingTool && canCreate && canReadTools && isCurrent()) navigate('/ai/workflows/new?tool=' + encodeURIComponent(incomingTool), { replace: true });
  }, [incomingTool, canCreate, canReadTools, isCurrent, navigate]);
  useEffect(() => {
    if (!requestedRun || consumedRun.current === requestedRun || !/^[1-9]\d*$/.test(requestedRun) || !Number.isSafeInteger(Number(requestedRun)) || !canRun || !isCurrent()) return;
    consumedRun.current = requestedRun;
    restoreRunAttempt(Number(requestedRun));
    // 保存后只打开确认表单，真正执行仍需用户确认。
    // oxlint-disable-next-line react/set-state-in-effect
    setPanel({ mode: 'run', id: Number(requestedRun), key: crypto.randomUUID() });
    navigate('/ai/workflows', { replace: true });
  }, [requestedRun, canRun, isCurrent, navigate, restoreRunAttempt]);

  function closePanel() {
    if (writing) return;
    setHistoryOpen(false);
    if (detailId > 0) void client.cancelQueries({ queryKey: aiWorkspaceKeys.workflow(scope, detailId) });
    setPanel(null);
    setUncertain(null);
  }
  function openExisting(id: number, mode: 'view' | 'run') {
    if (writing || !isCurrent()) return;
    setHistoryOpen(false);
    setUncertain(null);
    if (mode === 'run') {
      restoreRunAttempt(id);
    }
    setPanel({ id, mode, key: crypto.randomUUID() });
  }
  async function refreshList() { await client.invalidateQueries({ queryKey: aiWorkspaceKeys.workflows(scope) }); }
  async function deleteWorkflow() {
    if (!deleteTarget || !beginWrite(permission('delete'))) return;
    const id = deleteTarget.id;
    try {
      await remove.mutateAsync(id);
      if (!isCurrent()) return;
      await client.cancelQueries({ queryKey: aiWorkspaceKeys.workflow(scope, id) });
      client.removeQueries({ queryKey: aiWorkspaceKeys.workflow(scope, id) });
      if (!isCurrent()) return;
      setDeleteTarget(null);
      if (detailId === id) setPanel(null);
      if ((list.data?.list.length ?? 0) <= 1 && page > 1) setPageParams(current => ({ ...current, page: Math.max(1, current.page - 1) }));
      void message.success('常用流程已删除，已有运行记录保留');
      await refreshList();
    } catch { /* 失败保留确认上下文，由请求层展示错误。 */ }
    finally { finishWrite(); }
  }
  async function runWorkflow(values: { input?: string; question?: string; text?: string; kb_id?: number }) {
    if (!workflow || panel?.mode !== 'run' || detail.isFetching || detail.error || workflow.enabled !== 1 || !isCurrent()) return;
    let attempt = runAttempt;
    if (!uncertain || !attempt) {
      try { attempt = { workflowId: workflow.id, request_id: crypto.randomUUID(), input: prepareWorkflowRunInput(values) }; }
      catch (error) { setRunError(error instanceof Error ? error.message : '请核对运行输入'); return; }
    }
    if (!beginWrite(permission('run'))) return;
    setRunAttempt(attempt); setRunError('');
    try { saveWorkflowRunAttempt(sessionStorage, runAttempts.current, attemptKey(workflow.id), attempt); } catch { setRunError('运行标识暂时只能保留在本页面，刷新前请先核对运行结果。'); }
    try {
      const result = await run.mutateAsync(attempt);
      if (!isCurrent()) return;
      if (!Number.isSafeInteger(result.task_id) || result.task_id <= 0) { setUncertain('run'); return; }
      try { clearWorkflowRunAttempt(sessionStorage, runAttempts.current, attemptKey(workflow.id)); } catch { /* 同一幂等标识仍只会读回原任务。 */ }
      setRunAttempt(null);
      const mayReadTasks = () => {
        const auth = useAuthStore.getState();
        return auth.permissionsReady && auth.permCodes.includes('sys:ai:task:list');
      };
      if (mayReadTasks()) await client.invalidateQueries({ queryKey: aiWorkspaceKeys.tasks(scope) });
      if (!isCurrent()) return;
      setPanel(null);
      if (mayReadTasks()) setSelectedTask(result.task_id);
      else setLastRunWithoutAccess(result);
    } catch (error) {
      if (isCurrent() && (!(error instanceof ApiError) || error.code >= 500)) setUncertain('run');
      else if (isCurrent()) { setRunAttempt(null); try { clearWorkflowRunAttempt(sessionStorage, runAttempts.current, attemptKey(workflow.id)); } catch { /* 留存的幂等标识不会触发自动执行。 */ } }
    } finally { finishWrite(); }
  }

  const displayed = list.data?.list ?? [];

  return <div className={styles.workspace}>
    <Card>
      <div className={styles.header}><div><Space><ApartmentOutlined /><Typography.Title level={3}>常用流程</Typography.Title><Tag>{list.data?.total ?? 0} 个流程</Tag></Space><p className={styles.headerNote}>保存常用的分析方式，每次单独填写资料与问题</p></div><Space>
        <Button icon={<ReloadOutlined />} aria-label="刷新常用流程列表" disabled={writing} onClick={() => void refreshList()} />
        {canCreate && <Button type="primary" icon={<PlusOutlined />} disabled={writing || !canReadTools} title={!canReadTools ? '需要查看工具权限才能选择注册工具' : undefined} onClick={() => navigate('/ai/workflows/new')}>新建常用流程</Button>}
      </Space></div>
    </Card>
    {lastRunWithoutAccess && <WorkflowRunResultNotice result={lastRunWithoutAccess} />}
    {canCreate && !canReadTools && <Alert type="info" showIcon title="尚无查看工具权限，可查看已有定义；新建和编辑步骤需要查看工具权限。" />}
    <Card>
      <div className={styles.toolbar}><Input.Search className={styles.search} prefix={<SearchOutlined />} allowClear aria-label="搜索流程名称" placeholder="搜索流程名称" value={search} onChange={event => setSearch(event.target.value)} onSearch={query => setPageParams(current => ({ ...current, page: 1, query: query.trim() || undefined }))} /><Typography.Text type="secondary">仅显示我创建的流程</Typography.Text></div>
      <QueryState error={list.error} onRetry={list.refetch} />
      {!list.error && <Table<AiWorkflowInfo> rowKey="id" loading={list.isPending} dataSource={displayed} scroll={{ x: 790 }} pagination={{ current: page, pageSize, total: list.data?.total ?? 0, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showTotal: total => `共 ${total} 个常用流程`, onChange: (next, size) => { if (!writing) setPageParams(current => ({ ...current, page: size !== pageSize ? 1 : next, page_size: size })); }, disabled: writing }} columns={[
        { title: '常用流程', dataIndex: 'name', render: (_: unknown, row) => <Button type="link" className={styles.workflowName} disabled={writing} onClick={() => openExisting(row.id, 'view')}>{row.name}</Button> },
        { title: '步骤', width: 90, render: (_: unknown, row) => `${row.step_count} 步` },
        { title: '状态', width: 100, dataIndex: 'enabled', render: (enabled: number) => <Tag color={enabled === 1 ? 'success' : 'default'}>{enabled === 1 ? '已启用' : '已停用'}</Tag> },
        { title: '更新于', dataIndex: 'updated_at', render: formatTime, responsive: ['lg'] },
        { title: '操作', width: 250, render: (_: unknown, row) => <Space size={4}>
          {can(permission('run')) && <Button type="text" size="small" icon={<PlayCircleOutlined />} disabled={writing || row.enabled !== 1} onClick={() => openExisting(row.id, 'run')}>运行</Button>}
          {canEdit && <Button type="text" size="small" icon={<EditOutlined />} disabled={writing || !canReadTools} onClick={() => navigate(`/ai/workflows/${row.id}/edit`)}>编辑</Button>}
          {(canCreate || can(permission('delete'))) && <Dropdown trigger={['click']} menu={{ items: [
            ...(canCreate ? [{ key: 'copy', label: '复制', icon: <CopyOutlined />, disabled: writing || !canReadTools }] : []),
            ...(can(permission('delete')) ? [{ key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true, disabled: writing }] : []),
          ], onClick: ({ key }) => { if (writing || !isCurrent()) return; if (key === 'copy') navigate(`/ai/workflows/new?copy=${row.id}`); else if (key === 'delete') setDeleteTarget(row); } }}>
            <Button type="text" size="small" icon={<MoreOutlined />} disabled={writing} aria-label={`更多操作：${row.name}`}>更多</Button>
          </Dropdown>}
        </Space> },
      ]} />}
    </Card>
    {selectedTask && <Card title="本次运行结果"><AsyncContent key={selectedTask} load={loadDetail} componentProps={{ inline: true, scope, taskId: selectedTask, guard, onClose: () => setSelectedTask(null) }} /></Card>}
    <Drawer open={panel?.mode === 'view'} title="常用流程详情" size={880} onClose={closePanel} destroyOnHidden>
      <QueryState isPending={detail.isPending || detail.isFetching} error={detail.error} onRetry={detail.refetch} />
      {workflow && !detail.isFetching && !detail.error && <><Button disabled={writing} onClick={() => setHistoryOpen(true)}>版本历史</Button><WorkflowDefinitionView name={workflow.name} definition={workflow.definition} /></>}
      {historyOpen && workflow && <AsyncContent load={loadHistory} componentProps={{ scope, workflow, currentVerified: detail.isSuccess && !detail.isFetching && !detail.error, guard, onRefresh: () => detail.refetch(), onClose: () => setHistoryOpen(false) }} />}
    </Drawer>
    <Modal open={panel?.mode === 'run'} title="确认运行常用流程" onCancel={closePanel} closable={!writing} mask={{ closable: !writing }} keyboard={!writing} confirmLoading={writing} okText={uncertain ? '核对原运行' : '开始后台运行'} okButtonProps={{ disabled: !workflow || workflow.enabled !== 1 || detail.isFetching || !!detail.error || (!!uncertain && !runAttempt) || !can(permission('run')) }} onOk={() => uncertain ? void runWorkflow({}) : runForm.submit()} destroyOnHidden>
      <QueryState isPending={detail.isPending || detail.isFetching} error={detail.error} onRetry={detail.refetch} />
      {workflow && !detail.isFetching && !detail.error && <>
        <Typography.Paragraph>将运行已保存的常用流程「{workflow.name}」，不会使用未保存的编辑内容。</Typography.Paragraph>
        {workflow.enabled !== 1 && <Alert type="warning" showIcon title="此常用流程已停用，请在编辑中启用后再运行。" />}
        <details className={styles.runDetails}><summary>查看本次已保存定义</summary><WorkflowDefinitionView name={workflow.name} definition={workflow.definition} /></details>
        <Form form={runForm} layout="vertical" initialValues={{ input: '', question: undefined, text: undefined }} onFinish={runWorkflow} disabled={writing || !!uncertain} clearOnDestroy>
          {runFields.kb_id && <Form.Item label="本次使用的知识库" name="kb_id" rules={[{ required: true, message: '请选择本次使用的知识库' }]}><KnowledgeChoice knowledge={{ scope, canRead: can('sys:ai:kb:list') }} disabled={writing || !!uncertain} /></Form.Item>}
          {runFields.question && <Form.Item label="本次问题" name="question" rules={[{ required: true, whitespace: true, message: '请输入本次希望分析的问题' }]}><Input.TextArea rows={4} placeholder="希望从资料中了解什么？" /></Form.Item>}
          {runFields.text && <Form.Item label="本次文本" name="text" rules={[{ required: true, message: '请输入本次需要处理的文本' }]}><Input.TextArea rows={5} placeholder="粘贴本次运行需要处理的文本" /></Form.Item>}
          {!runFields.question && !runFields.text && <Typography.Paragraph type="secondary">本常用流程使用已保存的固定参数；提交后可以关闭页面，在运行记录查看进度。</Typography.Paragraph>}
          <details className={styles.runDetails}><summary>高级运行选项（可选）</summary>
          <Form.Item label="运行输入 JSON（可选）" name="input" extra="JSON 对象中的字段可由 input.字段名 引用；上方问题和文本覆盖同名字段，总计最多 16000 UTF-8 字节。" rules={[{ validator: (_: unknown, value: string) => { const error = validateRunInput(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve(); } }]}><Input.TextArea rows={4} spellCheck={false} placeholder="{}" className={styles.jsonInput} /></Form.Item>
          </details>
        </Form>
      </>}
      {runError && <Alert type="warning" showIcon title={runError} />}
      {uncertain === 'run' && <Alert type="warning" showIcon title="上次运行结果待确认" description="点击“核对原运行”会复用原请求标识和输入，已有运行记录会直接返回，不会重复执行；也可到运行记录查看。" action={canReadTasks ? <Button onClick={() => navigate('/ai/tasks')}>前往运行记录</Button> : undefined} />}
    </Modal>
    <Modal open={!!deleteTarget} title="删除常用流程" okText="确认删除" okButtonProps={{ danger: true, disabled: !can(permission('delete')) }} confirmLoading={writing} onOk={() => void deleteWorkflow()} onCancel={() => { if (!writing) setDeleteTarget(null); }} closable={!writing} mask={{ closable: !writing }} keyboard={!writing}>
      <Typography.Paragraph>确定删除「{deleteTarget?.name}」？已有运行记录保留，删除后无法再运行此常用流程。</Typography.Paragraph>
    </Modal>
  </div>;
}
