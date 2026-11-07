import { useRef, useState } from 'react';
import { Alert, Button, Form, Input, Space, Typography } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { runAiDraft } from '../../api/ai';
import type { AiCapabilities, AiKbInfo, AiToolInfo } from '../../api/types';
import { ApiError } from '../../utils/response';
import { useAuthStore } from '../../stores/auth';
import type { AiWorkspaceScope } from './workspaceQueries';
import { aiWorkspaceKeys } from './workspaceQueries';
import type { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { WorkflowDraft } from './workflowDefinition';
import { prepareWorkflowRunInput, serializeWorkflow, validateRunInput, validateWorkflowName, workflowRunFields } from './workflowDefinition';
import { clearDraftRunAttempt, draftRunAttemptKey, readDraftRunAttempt, writeDraftRunAttempt } from './draftRunAttempts';
import type { DraftRunAttempt } from './draftRunAttempts';
import { KnowledgeChoice } from './WorkflowParameters';
import { TaskDetail } from './TaskDetail';
import { getToolAvailability } from './toolGuides';
import styles from './workspace.module.css';

interface RunValues { question?: string; text?: string; input?: string; kb_id?: number }
interface Props {
  scope: AiWorkspaceScope; guard: ReturnType<typeof useAiWorkspaceGuard>; draft: WorkflowDraft;
  attemptId: string; tools?: readonly AiToolInfo[]; fixedKb?: AiKbInfo; capabilities?: AiCapabilities;
  disabled?: boolean; onSave?: (taskId: number) => void; initialInput?: string;
}

export function WorkflowDraftRun({ scope, guard, draft, attemptId, tools, fixedKb, capabilities, disabled, onSave, initialInput }: Props) {
  const departmentId = useAuthStore(state => state.selectedDeptId);
  const key = draftRunAttemptKey([scope[0], departmentId], attemptId);
  const [restored] = useState(() => {
    try { return { attempt: readDraftRunAttempt(sessionStorage, key), error: '' }; }
    catch { return { attempt: null, error: '浏览器无法恢复上次运行标识，请先到运行记录核对。' }; }
  });
  const [attempt, setAttempt] = useState<DraftRunAttempt | null>(restored.attempt);
  const attemptRef = useRef(attempt);
  const [unknown, setUnknown] = useState(!!restored.attempt || !!restored.error);
  const [feedback, setFeedback] = useState(restored.error);
  const [taskId, setTaskId] = useState<number | null>(() => {
    try { const value = Number(sessionStorage.getItem(`${key}:task`)); return Number.isSafeInteger(value) && value > 0 ? value : null; } catch { return null; }
  });
  const [form] = Form.useForm<RunValues>();
  const [initialValues] = useState<RunValues>(() => {
    const raw = restored.attempt?.input ?? initialInput;
    if (!raw) return { question: '', text: '', input: '' };
    try { const input = JSON.parse(raw); return { question: typeof input.question === 'string' ? input.question : '', text: typeof input.text === 'string' ? input.text : '', kb_id: typeof input.kb_id === 'number' ? input.kb_id : undefined, input: raw }; } catch { return { input: raw }; }
  });
  const mutation = useMutation({ mutationFn: runAiDraft, retry: false });
  const client = useQueryClient();
  let fields = { question: false, text: false, kb_id: false };
  try { fields = workflowRunFields(serializeWorkflow(draft.steps)); } catch { /* 编辑中的无效定义在提交时给出详细错误。 */ }
  const needsAnalysis = draft.steps.some(step => step.tool === 'kb_analyze');
  const needsRetrieval = draft.steps.some(step => step.tool === 'kb_analyze' || step.tool === 'kb_search');
  const availability = getToolAvailability(needsAnalysis ? 'kb_analyze' : 'kb_search', capabilities);
  const configurationMissing = needsRetrieval && (!capabilities?.retrieval_configured || needsAnalysis && !capabilities.analysis_configured);
  const canRun = guard.can('sys:ai:workflow:run');
  const busy = guard.writing || mutation.isPending;

  async function submit(values: RunValues) {
    if (!canRun || !unknown && (disabled || configurationMissing || fixedKb && fixedKb.ready_count < 1)) return;
    let request = attemptRef.current;
    if (!unknown) {
      try {
        const nameError = validateWorkflowName(draft.name);
        if (nameError) throw new Error(nameError);
        request = { name: draft.name.trim(), definition: serializeWorkflow(draft.steps, tools), input: prepareWorkflowRunInput({ ...values, ...(fixedKb ? { kb_id: fixedKb.id } : {}) }), request_id: crypto.randomUUID() };
      } catch (error) { setFeedback(error instanceof Error ? error.message : '请核对分析配置'); return; }
    }
    if (!request || !guard.beginWrite('sys:ai:workflow:run')) return;
    attemptRef.current = request; setAttempt(request); setFeedback('');
    try { writeDraftRunAttempt(sessionStorage, key, request); }
    catch { setFeedback('运行标识暂时保留在当前页面，刷新前请先核对本次运行。'); }
    try {
      const result = await mutation.mutateAsync(request);
      if (!guard.isCurrent()) return;
      if (!Number.isSafeInteger(result.task_id) || result.task_id <= 0) { setUnknown(true); return; }
      setTaskId(result.task_id); setUnknown(false); setAttempt(null); attemptRef.current = null;
      try { clearDraftRunAttempt(sessionStorage, key); sessionStorage.setItem(`${key}:task`, String(result.task_id)); } catch { /* 已取得任务ID，结果留在页面核对。 */ }
      void client.invalidateQueries({ queryKey: aiWorkspaceKeys.tasks(scope) });
    } catch (error) {
      if (!guard.isCurrent()) return;
      const uncertain = !(error instanceof ApiError) || error.code >= 500;
      setUnknown(uncertain);
      if (!uncertain) { attemptRef.current = null; setAttempt(null); try { clearDraftRunAttempt(sessionStorage, key); } catch { /* 下次恢复仍要求核对原操作。 */ } }
      setFeedback(uncertain ? '结果尚未确认。核对时复用原标识、定义与输入，不会发起另一项运行。' : '本次未能开始，输入已保留，请处理请求错误后重试。');
    } finally { guard.finishWrite(); }
  }

  return <div className={styles.analysisPanel}>
    <Typography.Title level={4}>{fixedKb ? '用这些资料完成工作' : '用真实问题试一次'}</Typography.Title>
    <Typography.Paragraph type="secondary">{fixedKb ? '按问题检索相关片段，生成带来源的结果。资料不足时会说明。' : '直接运行当前编辑内容，每次可以换资料库和问题。试运行也会保留正式运行记录。'}</Typography.Paragraph>
    <Form form={form} layout="vertical" onFinish={values => void submit(values)} disabled={busy || unknown} initialValues={initialValues}>
      {fields.kb_id && !fixedKb && <Form.Item label="本次使用的知识库" name="kb_id" rules={[{ required: true, message: '请选择本次知识库' }]}><KnowledgeChoice knowledge={{ scope, canRead: guard.can('sys:ai:kb:list') }} disabled={busy || unknown} /></Form.Item>}
      {fields.question && <Form.Item label={fixedKb ? '你希望了解什么？' : '本次问题'} name="question" rules={[{ required: true, whitespace: true, message: '请输入本次问题' }]}><Input.TextArea rows={4} placeholder="例如：总结关键步骤、列出风险与依据，并说明还缺哪些信息。" /></Form.Item>}
      {fields.text && <Form.Item label="本次文本" name="text" rules={[{ required: true, message: '请输入本次文本' }]}><Input.TextArea rows={5} placeholder={draft.steps.some(step => step.tool === 'json_formatter') ? '粘贴合法 JSON 文本' : '粘贴本次需要处理的文本'} /></Form.Item>}
      <details className={styles.advanced}><summary>高级运行参数</summary><Form.Item name="input" label="输入 JSON（可选）" rules={[{ validator: (_, value: string) => { const error = validateRunInput(value ?? ''); return error ? Promise.reject(new Error(error)) : Promise.resolve(); } }]}><Input.TextArea rows={3} placeholder="{}" /></Form.Item></details>
      {configurationMissing && <Alert type="warning" showIcon title={availability.label} description={availability.description} />}
      {fixedKb && fixedKb.ready_count === 0 && <Alert type="info" showIcon title="先添加资料并等待处理完成，再开始分析。" />}
      {!canRun && <Alert type="info" title="需要运行流程权限，才能执行分析。" />}
      <Button type="primary" icon={<PlayCircleOutlined />} loading={busy} disabled={!canRun || disabled || !!configurationMissing || !!fixedKb && fixedKb.ready_count < 1 || unknown} htmlType="submit">{fixedKb ? '开始分析' : '试运行并查看结果'}</Button>
    </Form>
    {unknown && <Alert type="warning" showIcon title="上次运行结果待确认" description={attempt ? `请求标识：${attempt.request_id}` : '请先到运行记录核对，页面不会自动重新提交。'} action={attempt ? <Button disabled={busy || !canRun} onClick={() => void submit({})}>核对原运行</Button> : <Button href="/ai/tasks">查看运行记录</Button>} />}
    {feedback && <Alert type="info" title={feedback} />}
    {taskId && <div className={styles.resultPanel}><Typography.Title level={4}>本次结果</Typography.Title>
      {guard.can('sys:ai:task:list') ? <TaskDetail key={taskId} inline showSaveAction={false} scope={scope} taskId={taskId} guard={guard} onClose={() => setTaskId(null)} /> : <Alert type="success" title={`已创建运行记录 #${taskId}，需要任务查看权限才能查看结果。`} />}
      {onSave && guard.can('sys:ai:task:list') && guard.can('sys:ai:workflow:list') && guard.can('sys:ai:workflow:create') && guard.can('sys:ai:tool:list') && <Space><Button onClick={() => onSave(taskId)} disabled={busy}>保存为常用流程</Button></Space>}
    </div>}
  </div>;
}
