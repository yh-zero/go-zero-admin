import { useEffect, useRef, useState } from 'react';
import { Alert, App, Button, Card, Result, Space, Spin, Typography } from 'antd';
import { ArrowLeftOutlined, DatabaseOutlined, FileTextOutlined, PlusOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useBlocker, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { createAiWorkflow, updateAiWorkflow } from '../../api/ai';
import type { AiCreateWorkflowReq, AiUpdateWorkflowReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { ApiError } from '../../utils/response';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, toolListOptions, workflowDetailOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { WorkflowEditor } from './WorkflowEditor';
import { newWorkflowStep, parseWorkflow, workflowTemplate } from './workflowDefinition';
import type { WorkflowDraft } from './workflowDefinition';
import { clearWorkflowDraft, draftContent, makeWorkflowDraft, readWorkflowDraft, resolveWorkflowEditorDraft, workflowDraftKey, workflowSnapshot, writeWorkflowDraft } from './workflowDraft';
import type { StoredWorkflowDraft, WorkflowSubmissionState } from './workflowDraft';
import styles from './workflow.module.css';

const readCode = 'sys:ai:workflow:list';
interface EditorSeed { key: string; draft: WorkflowDraft; baseSnapshot: string | null; initialContent: string }

export default function WorkflowEditorPage() {
  const { id } = useParams();
  const workflowId = id === undefined ? 'new' : Number(id);
  if (workflowId !== 'new' && (!Number.isSafeInteger(workflowId) || workflowId <= 0)) return <Result status="404" title="工作流不存在" />;
  return <AIWorkspaceAccess code={readCode} title="工作流编辑">{scope => <WorkflowEditorPageSession key={`${scope.join(':')}:${workflowId}`} scope={scope} workflowId={workflowId} />}</AIWorkspaceAccess>;
}

function WorkflowEditorPageSession({ scope, workflowId }: { scope: AiWorkspaceScope; workflowId: number | 'new' }) {
  const existing = workflowId !== 'new';
  const operationCode = `sys:ai:workflow:${existing ? 'edit' : 'create'}`;
  const { can, isCurrent, beginWrite, finishWrite, writing } = useAiWorkspaceGuard(scope, readCode);
  const canEdit = can(operationCode);
  const canReadTools = can('sys:ai:tool:list');
  const canReadKnowledge = can('sys:ai:kb:list');
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const storageKey = workflowDraftKey(scope[0], departmentId, workflowId);
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const requestedCopy = params.get('copy');
  const copyId = !existing && requestedCopy && /^[1-9]\d*$/.test(requestedCopy) && Number.isSafeInteger(Number(requestedCopy)) ? Number(requestedCopy) : 0;
  const { modal, message } = App.useApp();
  const client = useQueryClient();
  const [storageError, setStorageError] = useState('');
  const [restored] = useState<{ record: StoredWorkflowDraft | null; error: boolean }>(() => {
    try { return { record: readWorkflowDraft(sessionStorage, storageKey), error: false }; }
    catch { return { record: null, error: true }; }
  });
  const [seed, setSeed] = useState<EditorSeed | null>(null);
  const [dirty, setDirty] = useState(!!restored.record);
  const [submission, setSubmission] = useState<WorkflowSubmissionState>(restored.record?.submissionState ?? 'idle');
  const [restoredNotice, setRestoredNotice] = useState(!!restored.record);
  const [definitionError, setDefinitionError] = useState('');
  const [requiresKnowledge, setRequiresKnowledge] = useState(() => !!restored.record?.draft.steps.some(step => step.tool === 'kb_analyze' || step.tool === 'kb_search'));
  const liveDraft = useRef<WorkflowDraft | null>(restored.record?.draft ?? null);
  const baseSnapshot = useRef<string | null>(restored.record?.baseSnapshot ?? null);
  const submissionRef = useRef<WorkflowSubmissionState>(submission);
  const allowLeave = useRef(false);
  const pendingChanges = useRef(dirty);
  const toolsQuery = useQuery({ ...toolListOptions(scope), enabled: canEdit && canReadTools, refetchOnMount: 'always' });
  const detail = useQuery({ ...workflowDetailOptions(scope, existing ? workflowId : copyId), enabled: (existing || copyId > 0) && canEdit && canReadTools, refetchOnMount: 'always' });
  const create = useMutation({ mutationFn: (request: AiCreateWorkflowReq) => createAiWorkflow(request), retry: false });
  const update = useMutation({ mutationFn: (request: AiUpdateWorkflowReq) => updateAiWorkflow(existing ? workflowId : 0, request), retry: false });
  const tools = toolsQuery.data?.list ?? [];
  const currentSnapshot = detail.data ? workflowSnapshot(detail.data.workflow) : null;
  const conflict = !!seed && existing && !!currentSnapshot && seed.baseSnapshot !== currentSnapshot;

  function persist(draft: WorkflowDraft, state = submissionRef.current) {
    liveDraft.current = draft;
    try { writeWorkflowDraft(sessionStorage, storageKey, makeWorkflowDraft(draft, baseSnapshot.current, state)); setStorageError(''); }
    catch { setStorageError('浏览器暂时无法保存草稿，当前内容仍在页面中。关闭或刷新前请复制高级参数并记录名称。'); }
  }
  function changeDraft(draft: WorkflowDraft) {
    const changed = !seed || draftContent(draft) !== seed.initialContent;
    pendingChanges.current = changed;
    setDirty(changed);
    setRequiresKnowledge(draft.steps.some(step => step.tool === 'kb_analyze' || step.tool === 'kb_search'));
    persist(draft);
  }
  function setSubmissionState(state: WorkflowSubmissionState) {
    submissionRef.current = state;
    setSubmission(state);
    if (liveDraft.current) persist(liveDraft.current, state);
  }

  useEffect(() => {
    if (seed || !toolsQuery.isSuccess || toolsQuery.isFetching || (existing || copyId > 0) && (!detail.isSuccess || detail.isFetching)) return;
    try {
      let initial: WorkflowDraft = { name: '我的工作流', enabled: true, steps: [newWorkflowStep()] };
      if (!existing) {
        const incoming = params.get('tool');
        const preferred = incoming && toolsQuery.data.list.some(tool => tool.name === incoming) ? incoming : canReadKnowledge && toolsQuery.data.list.some(tool => tool.name === 'kb_analyze') ? 'kb_analyze' : toolsQuery.data.list.some(tool => tool.name === 'text_stats') ? 'text_stats' : '';
        initial = workflowTemplate(preferred, toolsQuery.data.list);
      }
      if (copyId && detail.data) initial = { name: Array.from(`${detail.data.workflow.name} 副本`).slice(0, 128).join(''), enabled: true, steps: parseWorkflow(detail.data.workflow.definition, toolsQuery.data.list) };
      const resolved = resolveWorkflowEditorDraft(existing ? detail.data?.workflow : undefined, restored.record, toolsQuery.data.list, initial);
      const { draft, baseSnapshot: snapshot } = resolved;
      baseSnapshot.current = snapshot;
      liveDraft.current = draft;
      // 首次成功加载是编辑基线；之后服务端重新查询不会替换 Form 当前值。
      // oxlint-disable-next-line react/set-state-in-effect
      setSeed({ key: crypto.randomUUID(), ...resolved });
      setDefinitionError('');
      setRequiresKnowledge(draft.steps.some(step => step.tool === 'kb_analyze' || step.tool === 'kb_search'));
      if (restored.record) { pendingChanges.current = true; setDirty(true); }
    } catch (error) { setDefinitionError(error instanceof Error ? error.message : '已保存定义暂不可编辑'); }
  }, [seed, toolsQuery.isSuccess, toolsQuery.isFetching, toolsQuery.data, existing, copyId, detail.isSuccess, detail.isFetching, detail.data, restored, currentSnapshot, params, canReadKnowledge]);

  useEffect(() => {
    function beforeUnload(event: BeforeUnloadEvent) {
      if (!pendingChanges.current && submissionRef.current === 'idle') return;
      if (liveDraft.current) {
        try { writeWorkflowDraft(sessionStorage, storageKey, makeWorkflowDraft(liveDraft.current, baseSnapshot.current, submissionRef.current)); } catch { /* 页面内已提示草稿存储限制，浏览器继续确认离开。 */ }
      }
      event.preventDefault();
      event.returnValue = '';
    }
    window.addEventListener('beforeunload', beforeUnload);
    return () => window.removeEventListener('beforeunload', beforeUnload);
  }, [storageKey]);
  const blocker = useBlocker(() => !allowLeave.current && (pendingChanges.current || submissionRef.current !== 'idle'));
  useEffect(() => {
    if (blocker.state !== 'blocked') return;
    const confirmation = modal.confirm({ title: submissionRef.current === 'submitting' ? '正在保存，请留在当前页面' : '离开工作流编辑？', content: submissionRef.current === 'submitting' ? '请求已经发出。请等待结果，避免重复创建。' : '未保存的内容会保留在当前标签页草稿中，返回后可继续编辑。', okText: '保留草稿并离开', cancelText: '继续编辑', okButtonProps: { disabled: submissionRef.current === 'submitting' }, onOk: () => blocker.proceed(), onCancel: () => blocker.reset() });
    return () => confirmation.destroy();
  }, [blocker, modal]);

  function selectTemplate(tool: string) {
    const apply = () => {
      const draft = workflowTemplate(tool, tools);
      setSeed({ key: crypto.randomUUID(), draft, baseSnapshot: null, initialContent: seed?.initialContent ?? '' });
      pendingChanges.current = true;
      setDirty(true);
      setRequiresKnowledge(draft.steps.some(step => step.tool === 'kb_analyze' || step.tool === 'kb_search'));
      persist(draft);
    };
    if (dirty) modal.confirm({ title: '使用这个起步模板？', content: '当前步骤会被替换。可取消并继续编辑现有内容。', okText: '替换步骤', cancelText: '继续当前编辑', onOk: apply }); else apply();
  }
  function discardDraft() {
    modal.confirm({ title: '丢弃当前草稿？', content: submission !== 'idle' ? '保存结果仍需到列表核对。丢弃草稿不会撤销已经发出的请求。' : '当前尚未保存的修改将被清除。', okText: '丢弃并返回列表', okButtonProps: { danger: true }, cancelText: '继续编辑', onOk: () => {
      try { clearWorkflowDraft(sessionStorage, storageKey); }
      catch { setStorageError('草稿清除失败，请检查浏览器存储设置后重试。'); return; }
      allowLeave.current = true;
      pendingChanges.current = false;
      submissionRef.current = 'idle';
      liveDraft.current = null;
      navigate('/ai/workflows');
    } });
  }
  async function save(draft: WorkflowDraft, definition: string) {
    if (submissionRef.current !== 'idle' || conflict || toolsQuery.isFetching || toolsQuery.error || existing && (detail.isFetching || detail.error) || !beginWrite(operationCode)) return;
    liveDraft.current = draft;
    setSubmissionState('submitting');
    try {
      let savedId = existing ? workflowId : 0;
      if (existing) await update.mutateAsync({ name: draft.name.trim(), enabled: draft.enabled ? 1 : 0, definition });
      else { const result = await create.mutateAsync({ name: draft.name.trim(), definition, interval_seconds: 0 }); savedId = result.workflow.id; }
      if (!isCurrent()) return;
      try { clearWorkflowDraft(sessionStorage, storageKey); } catch { setStorageError('保存成功，但草稿清除失败。请勿重复创建，返回列表核对。'); setSubmissionState('unknown'); return; }
      allowLeave.current = true;
      pendingChanges.current = false;
      submissionRef.current = 'idle';
      if (existing) client.removeQueries({ queryKey: aiWorkspaceKeys.workflow(scope, workflowId) });
      void client.invalidateQueries({ queryKey: aiWorkspaceKeys.workflows(scope) });
      void message.success(existing ? '工作流已保存，可开始运行' : '工作流已创建，可开始运行');
      navigate(`/ai/workflows?run=${savedId}`);
    } catch (error) {
      if (!isCurrent()) return;
      setSubmissionState(!(error instanceof ApiError) || error.code >= 500 ? 'unknown' : 'idle');
    } finally { finishWrite(); }
  }
  if (!canEdit || !canReadTools) return <Result status="403" title={canEdit ? '需要查看工具权限，才能配置工作流' : `没有${existing ? '编辑' : '新建'}工作流权限`} extra={<Button onClick={() => navigate('/ai/workflows')}>返回列表</Button>} />;
  const kbStepWithoutAccess = requiresKnowledge && !canReadKnowledge;
  const incoming = params.get('tool');
  return <div className={`${styles.workspace} ${styles.editorPage}`}>
    <div className={styles.pageHeading}><Space><Button type="text" icon={<ArrowLeftOutlined />} aria-label="返回工作流列表" onClick={() => navigate('/ai/workflows')} /><Typography.Title level={3}>{existing ? '编辑工作流' : '新建工作流'}</Typography.Title></Space><Space><Typography.Text type="secondary">{writing ? '正在保存…' : submission === 'unknown' ? '保存结果待核对 · 草稿已保留' : dirty ? storageError || restored.error ? '有未保存修改' : '未提交 · 当前标签页草稿已保存' : '准备编辑'}</Typography.Text>{seed && <Button type="text" danger disabled={writing} onClick={discardDraft}>丢弃草稿</Button>}</Space></div>
    {(storageError || restored.error) && <Alert type="warning" showIcon title={storageError || '浏览器草稿存储不可用，刷新或关闭前请自行保留当前内容。'} />}
    {restoredNotice && <Alert type="success" showIcon closable onClose={() => setRestoredNotice(false)} title="已恢复当前标签页的未保存草稿" />}
    {incoming && toolsQuery.isSuccess && !tools.some(tool => tool.name === incoming) && <Alert type="warning" title="所选工具已不可用，请重新选择起步方式。" />}
    {submission === 'unknown' && <Alert type="warning" showIcon title="上次保存结果未知，草稿已保留" description="请先在工作流列表核对是否已保存，确认后再继续。页面不会自动重复提交。" action={<Space><Button href="/ai/workflows" target="_blank">打开列表核对</Button><Button onClick={() => modal.confirm({ title: '已核对保存结果？', content: '确认需要再次保存后继续。新建操作可能产生重复工作流，请先确认列表。', okText: '已核对，继续编辑', cancelText: '返回核对', onOk: () => setSubmissionState('idle') })}>已核对</Button></Space>} />}
    {conflict && <Alert type="warning" showIcon title="已保存的工作流在编辑期间发生变化" description="当前草稿保持不变。请与列表中的最新定义核对；继续保存会覆盖服务端当前内容。" action={<Button onClick={() => modal.confirm({ title: '保留草稿并继续保存？', content: '当前服务端没有版本锁。确认已核对最新定义后，后续保存将使用当前草稿覆盖它。', okText: '已核对，保留草稿', cancelText: '继续核对', onOk: () => { baseSnapshot.current = currentSnapshot; setSeed(current => current ? { ...current, baseSnapshot: currentSnapshot } : current); if (liveDraft.current) persist(liveDraft.current); } })}>核对后继续</Button>} />}
    <QueryState error={toolsQuery.error} onRetry={toolsQuery.refetch} />
    {(existing || copyId > 0) && <QueryState error={detail.error} onRetry={detail.refetch} />}
    {definitionError && <Alert type="warning" title="此定义暂不可编辑" description={definitionError} />}
    {toolsQuery.isSuccess && !tools.length && <Alert type="info" title="暂时没有可用工具，请联系管理员检查工具配置后刷新。" />}
    {!existing && tools.length > 0 && <div className={styles.templateGrid}>
      {tools.some(tool => tool.name === 'kb_analyze') && <Button className={styles.templateButton} disabled={writing || !canReadKnowledge || submission !== 'idle'} onClick={() => selectTemplate('kb_analyze')}><DatabaseOutlined /><span><strong>知识库分析</strong><small>{canReadKnowledge ? '运行时填写问题，分析并统计结果' : '需要查看知识库权限'}</small></span></Button>}
      {tools.some(tool => tool.name === 'text_stats') && <Button className={styles.templateButton} disabled={writing || submission !== 'idle'} onClick={() => selectTemplate('text_stats')}><FileTextOutlined /><span><strong>文本处理</strong><small>整理本次 JSON，再统计处理结果</small></span></Button>}
      <Button className={styles.templateButton} disabled={writing || submission !== 'idle'} onClick={() => selectTemplate('')}><PlusOutlined /><span><strong>空白起步</strong><small>按需要添加和安排步骤</small></span></Button>
    </div>}
    {seed ? <Card className={styles.editorCard}>
      {(toolsQuery.isFetching || existing && detail.isFetching) && <Alert type="info" title="正在核对工具和已保存内容，当前草稿已保留，核对完成后可保存。" />}
      {kbStepWithoutAccess && <Alert type="warning" title="当前步骤需要查看知识库权限，请联系管理员授权后继续保存。" />}
      <WorkflowEditor key={seed.key} initial={seed.draft} tools={tools} knowledge={{ scope, canRead: canReadKnowledge }} existing={existing} saving={writing} saveDisabled={submission !== 'idle' || conflict || toolsQuery.isFetching || !!toolsQuery.error || !tools.length || !!kbStepWithoutAccess || existing && (detail.isFetching || !!detail.error)} onChange={changeDraft} onSave={save} onCancel={() => navigate('/ai/workflows')} />
    </Card> : !definitionError && !toolsQuery.error && (!existing || !detail.error) && <Spin className="route-spinner" />}
  </div>;
}
