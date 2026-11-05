import { useEffect, useMemo, useState } from 'react';
import { Alert, App, Button, Card, Drawer, Form, Input, Modal, Space, Table, Tag, Typography } from 'antd';
import { ApartmentOutlined, DeleteOutlined, EditOutlined, PlayCircleOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { createAiWorkflow, deleteAiWorkflow, runAiWorkflow, updateAiWorkflow } from '../../api/ai';
import type { AiCreateWorkflowReq, AiRunWorkflowResp, AiUpdateWorkflowReq, AiWorkflowInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { correctPageParams } from '../../hooks/usePagedQuery';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { ApiError } from '../../utils/response';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, toolListOptions, workflowDetailOptions, workflowListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { WorkflowDefinitionView, WorkflowEditor, WorkflowRunResultNotice } from './WorkflowEditor';
import { newWorkflowStep, parseWorkflow, retainWorkflowDraftSeed, validateRunInput } from './workflowDefinition';
import type { WorkflowDraft, WorkflowDraftSeed } from './workflowDefinition';
import styles from './workflow.module.css';

const readCode = 'sys:ai:workflow:list';
const permission = (operation: string) => `sys:ai:workflow:${operation}`;
type Panel = { mode: 'create'; key: string; tool?: string } | { mode: 'view' | 'edit' | 'run'; id: number; key: string };

export default function AIWorkflows() {
  return <AIWorkspaceAccess code={readCode} title="工作流管理">{scope => <AIWorkflowsSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AIWorkflowsSession({ scope }: { scope: AiWorkspaceScope }) {
  const client = useQueryClient();
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { can, isCurrent, beginWrite, finishWrite, writing } = useAiWorkspaceGuard(scope, readCode);
  const [pageParams, setPageParams] = useState({ page: 1, page_size: 20 });
  const { page, page_size: pageSize } = pageParams;
  const [search, setSearch] = useState('');
  const [panel, setPanel] = useState<Panel | null>(null);
  const [draftSeed, setDraftSeed] = useState<WorkflowDraftSeed | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AiWorkflowInfo | null>(null);
  const [uncertain, setUncertain] = useState<'create' | 'run' | null>(null);
  const [lastRunWithoutAccess, setLastRunWithoutAccess] = useState<AiRunWorkflowResp | null>(null);
  const [runForm] = Form.useForm<{ input: string }>();
  const incomingTool = searchParams.get('tool');
  const canReadTools = can('sys:ai:tool:list');
  const canCreate = can(permission('create'));
  const canEdit = can(permission('edit'));
  const canReadTasks = can('sys:ai:task:list');
  const list = useQuery(workflowListOptions(scope, pageParams));
  const completedTotal = list.isSuccess && !list.isFetching ? list.data.total : undefined;
  const needTools = (!!incomingTool && canCreate) || panel?.mode === 'create' || (panel?.mode === 'edit' && canEdit);
  const toolQuery = useQuery({ ...toolListOptions(scope), enabled: canReadTools && needTools });
  const tools = canReadTools ? toolQuery.data?.list ?? [] : [];
  const detailId = panel && panel.mode !== 'create' ? panel.id : 0;
  const detail = useQuery({ ...workflowDetailOptions(scope, detailId), enabled: detailId > 0, refetchOnMount: 'always' });
  const workflow = detail.data?.workflow;
  const create = useMutation({ mutationFn: (request: AiCreateWorkflowReq) => createAiWorkflow(request), retry: false });
  const update = useMutation({ mutationFn: ({ id, request }: { id: number; request: AiUpdateWorkflowReq }) => updateAiWorkflow(id, request), retry: false });
  const remove = useMutation({ mutationFn: deleteAiWorkflow, retry: false });
  const run = useMutation({ mutationFn: ({ id, input }: { id: number; input: string }) => runAiWorkflow(id, input === '' ? undefined : input), retry: false });

  useEffect(() => {
    if (completedTotal === undefined) return;
    // 远端删除会缩短列表；只修正返回本次总数的参数，不覆盖用户刚切换的新页。
    // oxlint-disable-next-line react/set-state-in-effect
    setPageParams(current => current === pageParams ? correctPageParams(current, completedTotal) : current);
  }, [completedTotal, pageParams]);

  useEffect(() => {
    if (!incomingTool || writing || !canCreate || !canReadTools || !toolQuery.data || !isCurrent()) return;
    if (!toolQuery.data.list.some(tool => tool.name === incomingTool)) return;
    // 路由查询参数来自工具目录；注册表与实时权限确认后，将外部导航同步为一次本地编辑草稿。
    // oxlint-disable-next-line react/set-state-in-effect
    setPanel({ mode: 'create', key: crypto.randomUUID(), tool: incomingTool });
    setUncertain(null);
    const next = new URLSearchParams(searchParams);
    next.delete('tool');
    setSearchParams(next, { replace: true });
  }, [incomingTool, writing, canCreate, canReadTools, toolQuery.data, isCurrent, searchParams, setSearchParams]);

  function closePanel() {
    if (writing) return;
    if (detailId > 0) void client.cancelQueries({ queryKey: aiWorkspaceKeys.workflow(scope, detailId) });
    setPanel(null);
    setUncertain(null);
  }
  function openExisting(id: number, mode: 'view' | 'edit' | 'run') {
    if (writing || !isCurrent()) return;
    setUncertain(null);
    runForm.resetFields();
    setPanel({ id, mode, key: crypto.randomUUID() });
  }
  async function refreshList() { await client.invalidateQueries({ queryKey: aiWorkspaceKeys.workflows(scope) }); }
  async function saveWorkflow(draft: WorkflowDraft, definition: string) {
    if (!panel || panel.mode === 'run' || uncertain || !canReadTools || !toolQuery.data || toolQuery.isFetching || toolQuery.error || (panel.mode !== 'create' && (detail.isFetching || detail.error))) return;
    const creating = panel.mode === 'create';
    if (!beginWrite(permission(creating ? 'create' : 'edit'))) return;
    try {
      if (creating) await create.mutateAsync({ name: draft.name.trim(), definition, interval_seconds: 0 });
      else await update.mutateAsync({ id: panel.id, request: { name: draft.name.trim(), definition, enabled: draft.enabled ? 1 : 0 } });
      if (!isCurrent()) return;
      if (!creating) {
        await client.cancelQueries({ queryKey: aiWorkspaceKeys.workflow(scope, panel.id) });
        client.removeQueries({ queryKey: aiWorkspaceKeys.workflow(scope, panel.id) });
      }
      if (!isCurrent()) return;
      setPanel(null);
      setSearch('');
      void message.success(creating ? '工作流已创建，默认启用' : '工作流已保存');
      await refreshList();
    } catch (error) {
      if (isCurrent() && creating && (!(error instanceof ApiError) || error.code >= 500)) setUncertain('create');
    } finally { finishWrite(); }
  }
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
      void message.success('工作流已删除，已有任务记录保留');
      await refreshList();
    } catch { /* 失败保留确认上下文，由请求层展示错误。 */ }
    finally { finishWrite(); }
  }
  async function runWorkflow({ input }: { input: string }) {
    if (!workflow || panel?.mode !== 'run' || detail.isFetching || detail.error || workflow.enabled !== 1 || uncertain || !beginWrite(permission('run'))) return;
    try {
      const result = await run.mutateAsync({ id: workflow.id, input });
      if (!isCurrent()) return;
      if (!Number.isSafeInteger(result.task_id) || result.task_id <= 0) { setUncertain('run'); return; }
      const mayReadTasks = () => {
        const auth = useAuthStore.getState();
        return auth.permissionsReady && auth.permCodes.includes('sys:ai:task:list');
      };
      if (mayReadTasks()) await client.invalidateQueries({ queryKey: aiWorkspaceKeys.tasks(scope) });
      if (!isCurrent()) return;
      setPanel(null);
      if (mayReadTasks()) navigate(`/ai/tasks?task_id=${result.task_id}`);
      else setLastRunWithoutAccess(result);
    } catch (error) {
      if (isCurrent() && (!(error instanceof ApiError) || error.code >= 500)) setUncertain('run');
    } finally { finishWrite(); }
  }

  const { initial, definitionError } = useMemo((): { initial?: WorkflowDraft; definitionError?: string } => {
    const available = canReadTools ? toolQuery.data?.list ?? [] : [];
    if (panel?.mode === 'create') return { initial: { name: panel.tool ? `${panel.tool}工作流` : '', enabled: true, steps: [newWorkflowStep(panel.tool ?? available[0]?.name)] } };
    if (workflow && panel?.mode === 'edit' && canEdit && canReadTools && toolQuery.data) {
      try { return { initial: { name: workflow.name, enabled: workflow.enabled === 1, steps: parseWorkflow(workflow.definition, available) } }; }
      catch (error) { return { definitionError: error instanceof Error ? error.message : '定义暂不可编辑' }; }
    }
    return {};
  }, [panel, workflow, canEdit, canReadTools, toolQuery.data]);
  const incomingError = incomingTool && (!canCreate || !canReadTools ? '需要新建工作流与查看工具权限，才能使用工具创建工作流。'
    : toolQuery.data && !tools.some(tool => tool.name === incomingTool) ? '工具不存在或已从注册表移除，请在工具目录重新选择。' : '');
  const canUseEditor = panel?.mode === 'create' ? canCreate && canReadTools : panel?.mode === 'edit' && canEdit && canReadTools;
  const initialReady = canUseEditor && !toolQuery.isPending && !toolQuery.error && (panel?.mode === 'create' || !detail.isFetching && !detail.error);
  const nextDraftSeed = retainWorkflowDraftSeed(draftSeed, canUseEditor ? panel?.key : undefined, initialReady ? initial : undefined);
  // 首次成功加载后由 Form 持有编辑值。只按抽屉标识切换种子，后台查询状态不会卸载当前表单。
  if (nextDraftSeed !== draftSeed) setDraftSeed(nextDraftSeed);
  const editorSaveDisabled = !!uncertain || toolQuery.isFetching || !!toolQuery.error || !tools.length || (panel?.mode !== 'create' && (detail.isFetching || !!detail.error));
  const displayed = (list.data?.list ?? []).filter(item => item.name.toLocaleLowerCase().includes(search.toLocaleLowerCase()));

  return <div className={styles.workspace}>
    <Card>
      <div className={styles.header}><div><Space><ApartmentOutlined /><Typography.Title level={3}>工作流管理</Typography.Title><Tag>{list.data?.total ?? 0} 个工作流</Tag></Space><p className={styles.headerNote}>将工具组织为顺序步骤，在关键步骤前确认</p></div><Space>
        <Button icon={<ReloadOutlined />} aria-label="刷新工作流列表" disabled={writing} onClick={() => void refreshList()} />
        {canCreate && <Button type="primary" icon={<PlusOutlined />} disabled={writing || !canReadTools} title={!canReadTools ? '需要查看工具权限才能选择注册工具' : undefined} onClick={() => { setUncertain(null); setPanel({ mode: 'create', key: crypto.randomUUID() }); }}>新建工作流</Button>}
      </Space></div>
    </Card>
    {incomingError && <Alert type="warning" showIcon title={incomingError} />}
    {lastRunWithoutAccess && <WorkflowRunResultNotice result={lastRunWithoutAccess} />}
    {canCreate && !canReadTools && <Alert type="info" showIcon title="尚无查看工具权限，可查看已有定义；新建和编辑步骤需要查看工具权限。" />}
    <Card>
      <div className={styles.toolbar}><Input className={styles.search} prefix={<SearchOutlined />} allowClear aria-label="筛选当前页工作流" placeholder="筛选当前页工作流名称" value={search} onChange={event => setSearch(event.target.value)} /><Typography.Text type="secondary">仅显示我创建的工作流 · 名称搜索仅过滤本页</Typography.Text></div>
      <QueryState error={list.error} onRetry={list.refetch} />
      {!list.error && <Table<AiWorkflowInfo> rowKey="id" loading={list.isPending} dataSource={displayed} scroll={{ x: 790 }} pagination={{ current: page, pageSize, total: list.data?.total ?? 0, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showTotal: total => `共 ${total} 个工作流`, onChange: (next, size) => { if (!writing) setPageParams({ page: size !== pageSize ? 1 : next, page_size: size }); }, disabled: writing }} columns={[
        { title: '工作流', dataIndex: 'name', render: (_: unknown, row) => <><Button type="link" className={styles.workflowName} disabled={writing} onClick={() => openExisting(row.id, 'view')}>{row.name}</Button><div className={styles.rowNote}>#{row.id} · 手动运行</div></> },
        { title: '步骤', width: 90, render: (_: unknown, row) => { try { return `${parseWorkflow(row.definition).length} 步`; } catch { return '定义异常'; } } },
        { title: '状态', width: 100, dataIndex: 'enabled', render: (enabled: number) => <Tag color={enabled === 1 ? 'success' : 'default'}>{enabled === 1 ? '已启用' : '已停用'}</Tag> },
        { title: '更新于', dataIndex: 'updated_at', render: formatTime, responsive: ['lg'] },
        { title: '操作', width: 260, render: (_: unknown, row) => <Space size={4}>
          {canEdit && <Button type="text" size="small" icon={<EditOutlined />} disabled={writing || !canReadTools} onClick={() => openExisting(row.id, 'edit')}>编辑</Button>}
          {can(permission('run')) && <Button type="text" size="small" icon={<PlayCircleOutlined />} disabled={writing || row.enabled !== 1} onClick={() => openExisting(row.id, 'run')}>运行</Button>}
          {can(permission('delete')) && <Button type="text" size="small" danger icon={<DeleteOutlined />} disabled={writing} onClick={() => setDeleteTarget(row)}>删除</Button>}
        </Space> },
      ]} />}
    </Card>
    <Drawer open={!!panel && panel.mode !== 'run'} title={panel?.mode === 'create' ? '新建工作流' : panel?.mode === 'edit' && canUseEditor ? '编辑工作流' : '工作流详情'} size={880} onClose={closePanel} closable={!writing} maskClosable={!writing} keyboard={!writing} destroyOnHidden>
      {uncertain === 'create' && <Alert type="warning" showIcon title="创建结果未知，请先刷新列表核对，确认未创建后再新建。" description="本次创建没有幂等标识，不会自动重试。" action={<Button onClick={() => { void refreshList(); closePanel(); }}>刷新列表核对</Button>} />}
      {detailId > 0 && <QueryState isPending={!nextDraftSeed && (detail.isPending || detail.isFetching)} error={detail.error} onRetry={detail.refetch} />}
      {canUseEditor && <QueryState isPending={!nextDraftSeed && toolQuery.isPending} error={toolQuery.error} onRetry={toolQuery.refetch} />}
      {nextDraftSeed && (detail.isFetching || toolQuery.isFetching) && <Alert type="info" showIcon title="正在核对已保存定义与工具目录，未保存草稿已保留；完成后可继续保存。" />}
      {nextDraftSeed && (detail.error || toolQuery.error) && <Alert type="warning" showIcon title="后台数据核对失败，未保存草稿已保留；请重新加载成功后再保存。" />}
      {canUseEditor && toolQuery.data && !tools.length && <Alert type="info" showIcon title="工具注册表为空，暂时无法保存步骤。请刷新工具目录后重试。" />}
      {definitionError && <Alert type="warning" showIcon title="此定义暂不可编辑" description={definitionError} />}
      {canUseEditor && nextDraftSeed ? <WorkflowEditor key={nextDraftSeed.key} initial={nextDraftSeed.draft} tools={tools} existing={panel?.mode !== 'create'} saving={writing} saveDisabled={editorSaveDisabled} onSave={saveWorkflow} onCancel={closePanel} />
        : workflow && detailId > 0 && !detail.isFetching && !detail.error && <WorkflowDefinitionView name={workflow.name} definition={workflow.definition} />}
    </Drawer>
    <Modal open={panel?.mode === 'run'} title="确认运行工作流" onCancel={closePanel} closable={!writing} maskClosable={!writing} keyboard={!writing} confirmLoading={writing} okText="确认运行" okButtonProps={{ disabled: !workflow || workflow.enabled !== 1 || detail.isFetching || !!detail.error || !!uncertain || !can(permission('run')) }} onOk={() => runForm.submit()} destroyOnHidden>
      <QueryState isPending={detail.isPending || detail.isFetching} error={detail.error} onRetry={detail.refetch} />
      {workflow && !detail.isFetching && !detail.error && <>
        <Typography.Paragraph>将运行已保存的工作流「{workflow.name}」，不会使用未保存的编辑内容。</Typography.Paragraph>
        {workflow.enabled !== 1 && <Alert type="warning" showIcon title="此工作流已停用，请在编辑中启用后再运行。" />}
        <details className={styles.runDetails}><summary>查看本次已保存定义</summary><WorkflowDefinitionView name={workflow.name} definition={workflow.definition} /></details>
        <Form form={runForm} layout="vertical" initialValues={{ input: '' }} onFinish={runWorkflow} disabled={writing}>
          <Form.Item label="运行输入 JSON（可选）" name="input" extra="仅保存为本次运行记录，不会注入步骤参数；最多 16000 UTF-8 字节。" rules={[{ validator: (_: unknown, value: string) => { const error = validateRunInput(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve(); } }]}><Input.TextArea rows={4} spellCheck={false} placeholder="{}" className={styles.jsonInput} /></Form.Item>
        </Form>
      </>}
      {uncertain === 'run' && <Alert type="warning" showIcon title={canReadTasks ? '运行结果未知，请先到任务中心刷新核对，确认未创建任务后再发起。' : '运行结果未知，请联系管理员授予任务查看权限后核对，确认未创建任务前请勿重复运行。'} description="本次运行没有幂等标识，不会自动重试。" action={canReadTasks ? <Button onClick={() => navigate('/ai/tasks')}>前往任务中心核对</Button> : undefined} />}
    </Modal>
    <Modal open={!!deleteTarget} title="删除工作流" okText="确认删除" okButtonProps={{ danger: true, disabled: !can(permission('delete')) }} confirmLoading={writing} onOk={() => void deleteWorkflow()} onCancel={() => { if (!writing) setDeleteTarget(null); }} closable={!writing} maskClosable={!writing} keyboard={!writing}>
      <Typography.Paragraph>确定删除「{deleteTarget?.name}」？已有任务记录保留，删除后无法再运行此工作流。</Typography.Paragraph>
    </Modal>
  </div>;
}
