import { useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Pagination, Select, Space, Switch, Typography } from 'antd';
import { useQuery } from '@tanstack/react-query';
import type { AiToolInfo } from '../../api/types';
import { fetchAiKb } from '../../api/ai';
import { aiWorkspaceKeys, kbListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { isWorkflowReference, simpleToolFields, updateStepInputField } from './workflowDefinition';
import type { WorkflowStepDraft } from './workflowDefinition';
import styles from './workflow.module.css';

export interface WorkflowKnowledgeAccess { scope: AiWorkspaceScope; canRead: boolean }
interface ParameterProps { value?: string; onChange?: (value: string) => void; tool?: AiToolInfo; knowledge?: WorkflowKnowledgeAccess; disabled?: boolean; id?: string; previousSteps?: readonly WorkflowStepDraft[] }

export function KnowledgeChoice({ value, onChange, knowledge, optional, disabled }: { value?: unknown; onChange?: (value: unknown) => void; knowledge?: WorkflowKnowledgeAccess; optional?: boolean; disabled?: boolean }) {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const query = useQuery({ ...kbListOptions(knowledge?.scope ?? ['', 0, 0], { page, page_size: 20, query: search }), enabled: !!knowledge?.canRead });
  const selectedId = typeof value === 'number' && Number.isSafeInteger(value) && value > 0 ? value : 0;
  const selectedInPage = query.data?.list.find(kb => kb.id === selectedId);
  const selected = useQuery({ queryKey: [...aiWorkspaceKeys.kbs(knowledge?.scope ?? ['', 0, 0]), 'editor-choice', selectedId], enabled: !!knowledge?.canRead && query.isSuccess && selectedId > 0 && !selectedInPage, staleTime: 30_000, retry: false,
    queryFn: async ({ signal }) => { signal.throwIfAborted(); const data = await fetchAiKb(selectedId, signal); signal.throwIfAborted(); return data; } });
  if (!knowledge?.canRead) return <Alert type="warning" showIcon title="需要查看知识库权限，才能选择自己的知识库。请联系管理员授权。" />;
  return <Space orientation="vertical" className={styles.fullWidth}>
    <Select aria-label="知识库" className={styles.fullWidth} value={typeof value === 'number' ? value : optional ? 0 : undefined} allowClear={optional} disabled={disabled || query.isError} loading={query.isFetching} placeholder="按名称搜索知识库" onChange={onChange}
      showSearch={{ filterOption: false, onSearch: value => { setSearch(value); setPage(1); } }}
      options={[...(optional ? [{ value: 0, label: '全部本人知识库' }] : []), ...(query.data?.list ?? []).map(kb => ({ value: kb.id, label: `${kb.name} · ${kb.ready_count} 篇可使用` }))]}
      labelRender={option => query.data?.list.find(kb => kb.id === option.value)?.name ?? (option.value === 0 ? '全部本人知识库' : selected.data?.kb.name ?? (selected.isError ? '原知识库不可用，请重新选择' : '正在核对已选知识库…'))}
      notFoundContent={query.isError ? '知识库加载失败' : query.isFetching ? '正在加载…' : '暂无知识库，请先在知识库页面上传资料'}
      popupRender={menu => <>{menu}{(query.data?.total ?? 0) > 20 && <div className={styles.choicePagination} onMouseDown={event => event.preventDefault()}><Pagination simple size="small" current={page} pageSize={20} total={query.data?.total} onChange={setPage} /></div>}</>} />
    {query.isError && <Alert type="warning" title="知识库加载失败，现有选择已保留。" action={<Button size="small" onClick={() => void query.refetch()}>重试</Button>} />}
    {selectedId > 0 && !selectedInPage && selected.isError && <Alert type="warning" title="无法读取已选知识库，可能已删除或无权访问。原选择已保留，请重新选择或重试。" action={<Button size="small" onClick={() => void selected.refetch()}>重试</Button>} />}
    {query.isSuccess && query.data.total === 0 && <Typography.Text type="secondary">还没有知识库，请先到知识库页面添加资料，再回来继续编辑。</Typography.Text>}
  </Space>;
}

export function WorkflowParameters({ value = '{}', onChange, tool, knowledge, disabled, id, previousSteps = [] }: ParameterProps) {
  const [advanced, setAdvanced] = useState(false);
  const [editError, setEditError] = useState('');
  const fields = tool ? simpleToolFields(tool) : null;
  let input: Record<string, unknown> | null = null;
  try { const parsed = JSON.parse(value); if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) input = parsed; } catch { /* 原始字符串保留供高级编辑修正。 */ }
  const hasUnknown = input && fields && Object.keys(input).some(key => !fields.some(field => field.name === key));
  const basic = !!fields && !!input && !hasUnknown;
  function change(name: string, next: unknown) {
    try { onChange?.(updateStepInputField(value, name, next)); setEditError(''); }
    catch (error) { setEditError(error instanceof Error ? error.message : '请修正参数'); }
  }
  return <div id={id} className={styles.parameterFields}>
    {!basic && <Alert type="info" showIcon title="这一步包含高级参数，已保留完整内容。请在高级参数中核对。" />}
    {basic && fields.map(field => {
      const current = input![field.name];
      const reference = isWorkflowReference(current) ? current.$ref : undefined;
      return <Form.Item key={field.name} label={field.name === 'query' ? tool?.name === 'kb_analyze' ? '分析目标' : '检索问题' : field.label} required={field.required} extra={field.name === 'query' ? '用自然语言说明希望从资料中了解什么，最多 2000 UTF-8 字节。' : undefined}>
      <div data-parameter={field.name}>
        {(field.type === 'string' && !field.enum || field.name === 'kb_id') && <Select aria-label={`${field.label}来源`} className={styles.fullWidth} disabled={disabled} value={reference ?? 'literal'} onChange={source => change(field.name, source === 'literal' ? field.name === 'kb_id' ? undefined : '' : { $ref: source })}
          options={field.name === 'kb_id' ? [{ value: 'literal', label: '使用固定知识库' }, { value: 'input.kb_id', label: '每次运行时选择知识库' }] : [{ value: 'literal', label: '填写固定内容' }, { value: 'input.question', label: '使用本次运行的问题' }, { value: 'input.text', label: '使用本次运行的文本' }, ...previousSteps.flatMap((step, index) => [
            { value: `steps.${step.id ?? step.key}`, label: `第 ${index + 1} 步「${step.name}」完整结果` },
            ...(step.tool === 'kb_analyze' ? [{ value: `steps.${step.id ?? step.key}.analysis`, label: `第 ${index + 1} 步「${step.name}」分析正文` }] : []),
          ]), ...(reference && !['input.question', 'input.text'].includes(reference) && !previousSteps.some(step => reference === `steps.${step.id ?? step.key}` || reference === `steps.${step.id ?? step.key}.analysis`) ? [{ value: reference, label: `高级变量：${reference}` }] : [])]} />}
        {reference ? <Typography.Paragraph type="secondary">将在本次执行时读取变量；实际输入会在任务步骤和审批确认中展示。</Typography.Paragraph> :
        field.name === 'kb_id' && (tool?.name === 'kb_analyze' || tool?.name === 'kb_search') ? <KnowledgeChoice value={input![field.name]} onChange={next => change(field.name, next)} knowledge={knowledge} optional={tool.name === 'kb_search'} disabled={disabled} />
          : field.enum ? <Select aria-label={field.label} className={styles.fullWidth} disabled={disabled} allowClear={!field.required} value={typeof input![field.name] === 'boolean' ? String(input![field.name]) : input![field.name] as string | number | undefined} onChange={next => change(field.name, field.enum!.find(option => String(option) === String(next)))} options={field.enum.map(option => ({ value: typeof option === 'boolean' ? String(option) : option, label: String(option) }))} />
            : field.type === 'boolean' ? <Switch aria-label={field.label} disabled={disabled} checked={input![field.name] === true} onChange={next => change(field.name, next)} />
              : field.type === 'integer' || field.type === 'number' ? <InputNumber aria-label={field.label} disabled={disabled} value={typeof input![field.name] === 'number' ? input![field.name] as number : null} precision={field.type === 'integer' ? 0 : undefined} min={field.minimum} max={field.maximum} onChange={next => change(field.name, next ?? undefined)} />
                : <Input.TextArea aria-label={field.name === 'query' ? tool?.name === 'kb_analyze' ? '分析目标' : '检索问题' : field.label} disabled={disabled} value={typeof input![field.name] === 'string' ? input![field.name] as string : ''} onChange={event => change(field.name, event.target.value)} autoSize={{ minRows: field.name === 'query' ? 5 : 6, maxRows: 18 }} placeholder={field.name === 'query' ? '例如：总结这份资料的关键结论，列出需要关注的风险，并注明依据。' : '请输入需要处理的内容'} />}
      </div>
    </Form.Item>;
    })}
    {basic && fields.length === 0 && <Typography.Paragraph type="secondary">这一步无需填写参数。</Typography.Paragraph>}
    <details open={!basic || advanced} onToggle={event => setAdvanced(event.currentTarget.open)} className={styles.advancedParameters}>
      <summary>高级参数</summary>
      <Input.TextArea aria-label="固定参数 JSON" disabled={disabled} value={value} onChange={event => onChange?.(event.target.value)} autoSize={{ minRows: 6, maxRows: 18 }} spellCheck={false} className={styles.jsonInput} />
      {tool && <details className={styles.schema}><summary>查看参数说明</summary><pre className={styles.code}>{tool.parameters_schema}</pre></details>}
    </details>
    {editError && <Alert type="error" title={editError} />}
  </div>;
}
