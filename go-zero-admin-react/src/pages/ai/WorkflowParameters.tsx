import { useState } from 'react';
import { Alert, Form, Input, InputNumber, Select, Switch, Typography } from 'antd';
import type { AiToolInfo } from '../../api/types';
import { KnowledgeChoice } from './KnowledgeChoice';
import type { WorkflowKnowledgeAccess } from './KnowledgeChoice';
import { isWorkflowReference, simpleToolFields, updateStepInputField } from './workflowDefinition';
import type { WorkflowStepDraft } from './workflowDefinition';
import styles from './workflow.module.css';

export { KnowledgeChoice } from './KnowledgeChoice';
export type { WorkflowKnowledgeAccess } from './KnowledgeChoice';
interface ParameterProps { value?: string; onChange?: (value: string) => void; tool?: AiToolInfo; knowledge?: WorkflowKnowledgeAccess; disabled?: boolean; id?: string; previousSteps?: readonly WorkflowStepDraft[] }

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
      const sourceOptions = field.name === 'kb_id' ? [{ value: 'literal', label: '使用固定知识库' }, { value: 'input.kb_id', label: '每次运行时选择知识库' }] : [{ value: 'literal', label: '填写固定内容' }, { value: 'input.question', label: '使用本次运行的问题' }, { value: 'input.text', label: '使用本次运行的文本' }, ...previousSteps.flatMap((step, index) => [
        { value: `steps.${step.id ?? step.key}`, label: `第 ${index + 1} 步「${step.name}」完整结果` },
        ...(step.tool === 'kb_analyze' ? [{ value: `steps.${step.id ?? step.key}.analysis`, label: `第 ${index + 1} 步「${step.name}」分析正文` }] : []),
      ]), ...(reference && !['input.question', 'input.text'].includes(reference) && !previousSteps.some(step => reference === `steps.${step.id ?? step.key}` || reference === `steps.${step.id ?? step.key}.analysis`) ? [{ value: reference, label: `高级变量：${reference}` }] : [])];
      return <Form.Item key={field.name} label={field.name === 'query' ? tool?.name === 'kb_analyze' ? '分析目标' : '检索问题' : field.label} required={field.required} extra={field.name === 'query' ? '用自然语言说明希望从资料中了解什么，最多 2000 UTF-8 字节。' : undefined}>
      <div data-parameter={field.name}>
        {(field.type === 'string' && !field.enum || field.name === 'kb_id') && <details className={styles.parameterSource}>
          <summary>{sourceOptions.find(option => option.value === (reference ?? 'literal'))?.label ?? `高级变量：${reference}`}<span>更改</span></summary>
          <Select aria-label={`${field.label}来源`} className={styles.fullWidth} disabled={disabled} value={reference ?? 'literal'} onChange={source => change(field.name, source === 'literal' ? field.name === 'kb_id' ? undefined : '' : { $ref: source })} options={sourceOptions} />
        </details>}
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
