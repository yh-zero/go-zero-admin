import { useState } from 'react';
import { Alert, Button, Empty, Form, Input, Select, Space, Switch, Tag, Typography } from 'antd';
import { ArrowDownOutlined, ArrowUpOutlined, DeleteOutlined, PlusOutlined, SaveOutlined } from '@ant-design/icons';
import type { AiRunWorkflowResp, AiTaskStatus, AiToolInfo } from '../../api/types';
import { moveWorkflowStep, newWorkflowStep, parseStepInput, parseWorkflow, serializeWorkflow, validateWorkflowName } from './workflowDefinition';
import type { WorkflowDraft, WorkflowStepDraft } from './workflowDefinition';
import styles from './workflow.module.css';

interface WorkflowEditorProps {
  initial: WorkflowDraft;
  tools: readonly AiToolInfo[];
  existing: boolean;
  saving: boolean;
  saveDisabled?: boolean;
  onSave: (draft: WorkflowDraft, definition: string) => Promise<void>;
  onCancel: () => void;
}

export function WorkflowEditor({ initial, tools, existing, saving, saveDisabled, onSave, onCancel }: WorkflowEditorProps) {
  const [form] = Form.useForm<WorkflowDraft>();
  const steps: WorkflowStepDraft[] = Form.useWatch('steps', form) ?? initial.steps;
  const [selectedKey, setSelectedKey] = useState(initial.steps[0]?.key);
  const [validationError, setValidationError] = useState('');
  const selectedIndex = Math.max(0, steps.findIndex(step => step.key === selectedKey));
  const nameRule = { validator: (_: unknown, value: string) => {
    const error = validateWorkflowName(value ?? '');
    return error ? Promise.reject(new Error(error)) : Promise.resolve();
  } };

  function replaceSteps(next: WorkflowStepDraft[]) { form.setFieldValue('steps', next); setValidationError(''); }
  function addStep() {
    const current: WorkflowStepDraft[] = form.getFieldValue('steps');
    if (current.length >= 32) return;
    const next = newWorkflowStep(tools[0]?.name);
    replaceSteps([...current, next]);
    setSelectedKey(next.key);
  }
  function removeStep(key: string) {
    const current: WorkflowStepDraft[] = form.getFieldValue('steps');
    if (current.length <= 1) return;
    const index = current.findIndex(step => step.key === key);
    const next = current.filter(step => step.key !== key);
    replaceSteps(next);
    setSelectedKey(next[Math.min(index, next.length - 1)].key);
  }
  async function onFinish(draft: WorkflowDraft) {
    if (saving || saveDisabled) return;
    setValidationError('');
    let definition: string;
    try { definition = serializeWorkflow(draft.steps, tools); }
    catch (error) { setValidationError(error instanceof Error ? error.message : '请核对工作流定义'); return; }
    await onSave(draft, definition);
  }

  return <Form form={form} layout="vertical" initialValues={initial} onFinish={onFinish} onFinishFailed={({ errorFields }) => {
    const path = errorFields[0]?.name;
    if (path?.[0] === 'steps' && typeof path[1] === 'number' && steps[path[1]]) setSelectedKey(steps[path[1]].key);
    setValidationError('请修正表单中标出的名称或参数错误');
  }} disabled={saving} className={styles.editorForm}>
    <div className={styles.nameRow}>
      <Form.Item label="工作流名称" name="name" rules={[nameRule]} className={styles.nameField}><Input placeholder="例如：文本整理与复核" /></Form.Item>
      {existing ? <Form.Item label="启用" name="enabled" valuePropName="checked"><Switch /></Form.Item> : <Tag color="success">新建默认启用</Tag>}
    </div>
    <Alert type="info" showIcon title="顺序执行，每个步骤使用独立的固定参数。运行输入仅作记录，不会注入步骤。" />
    <div className={styles.editorGrid}>
      <aside className={styles.stepNavigation} aria-label="执行步骤">
        <div className={styles.stepHeading}><Typography.Text type="secondary">执行步骤 · {steps.length}/32</Typography.Text><Button size="small" icon={<PlusOutlined />} aria-label="添加步骤" disabled={saving || steps.length >= 32 || !tools.length} onClick={addStep} /></div>
        {steps.map((step, index) => <button key={step.key} type="button" className={`${styles.stepButton} ${index === selectedIndex ? styles.selectedStep : ''}`} aria-pressed={index === selectedIndex} disabled={saving} onClick={() => setSelectedKey(step.key)}>
          <span className={styles.stepNumber}>{index + 1}</span><span className={styles.stepSummary}><strong>{step.name || '未命名步骤'}</strong><small>{step.approval ? '执行前确认' : step.tool || '请选择工具'}</small></span>
        </button>)}
      </aside>
      <div className={styles.stepEditor}>
        <Form.List name="steps">{fields => fields.map(field => {
          const step = steps[field.name];
          const tool = tools.find(item => item.name === step?.tool);
          return <section key={step?.key ?? field.key} hidden={field.name !== selectedIndex} aria-label={`第 ${field.name + 1} 步参数`}>
            <div className={styles.stepHeading}><Typography.Title level={5}>第 {field.name + 1} 步</Typography.Title><Space size={4}>
              <Button size="small" aria-label="上移步骤" icon={<ArrowUpOutlined />} disabled={saving || field.name === 0} onClick={() => replaceSteps(moveWorkflowStep(form.getFieldValue('steps'), step.key, -1))} />
              <Button size="small" aria-label="下移步骤" icon={<ArrowDownOutlined />} disabled={saving || field.name === steps.length - 1} onClick={() => replaceSteps(moveWorkflowStep(form.getFieldValue('steps'), step.key, 1))} />
              <Button size="small" danger icon={<DeleteOutlined />} disabled={saving || steps.length <= 1} onClick={() => removeStep(step.key)}>删除</Button>
            </Space></div>
            <Form.Item name={[field.name, 'key']} hidden><Input /></Form.Item>
            <Form.Item label="步骤名称" name={[field.name, 'name']} rules={[nameRule]}><Input /></Form.Item>
            <Form.Item label="使用工具" name={[field.name, 'tool']} rules={[{ required: true, message: '请选择已注册工具' }]}>
              <Select options={tools.map(item => ({ value: item.name, label: item.name }))} onChange={value => { form.setFieldValue(['steps', field.name, 'input'], newWorkflowStep(value).input); }} />
            </Form.Item>
            {tool && <Typography.Paragraph type="secondary">{tool.description}</Typography.Paragraph>}
            <Form.Item label="固定参数 JSON" name={[field.name, 'input']} extra="必须为对象；text 最多 16000 UTF-8 字节" rules={[{ validator: (_: unknown, value: string) => {
              try { parseStepInput(value, tools.find(item => item.name === form.getFieldValue(['steps', field.name, 'tool']))); return Promise.resolve(); }
              catch (error) { return Promise.reject(error); }
            } }]}><Input.TextArea autoSize={{ minRows: 7, maxRows: 14 }} spellCheck={false} className={styles.jsonInput} /></Form.Item>
            {tool && <details className={styles.schema}><summary>查看工具参数 Schema</summary><pre className={styles.code}>{tool.parameters_schema}</pre></details>}
            <Form.Item label="执行前由本人确认" name={[field.name, 'approval']} valuePropName="checked" extra="确认后才会执行此步骤，包括最后一步。"><Switch /></Form.Item>
          </section>;
        })}</Form.List>
      </div>
    </div>
    {validationError && <Alert type="error" showIcon title={validationError} />}
    <footer className={styles.editorFooter}><Button onClick={onCancel} disabled={saving}>取消</Button><Button type="primary" icon={<SaveOutlined />} htmlType="submit" loading={saving} disabled={saveDisabled || !tools.length}>保存工作流</Button></footer>
  </Form>;
}

export function WorkflowDefinitionView({ definition, name }: { definition: string; name: string }) {
  let steps: WorkflowStepDraft[] = [];
  let error = '';
  try { steps = parseWorkflow(definition); }
  catch (cause) { error = cause instanceof Error ? cause.message : '定义无法解析'; }
  return <div className={styles.definitionView}>
    <Typography.Title level={4}>{name}</Typography.Title>
    {error ? <Alert type="warning" showIcon title="定义无法使用步骤编辑器展示" description={error} /> : steps.length ? <ol className={styles.readSteps}>{steps.map(step => <li key={step.key}><strong>{step.name}</strong> <Tag>{step.tool}</Tag>{step.approval && <Tag color="gold">执行前确认</Tag>}<pre className={styles.code}>{step.input}</pre></li>)}</ol> : <Empty description="暂无步骤" />}
    <details open={!!error}><summary>已保存的原始定义</summary><pre className={styles.code}>{definition}</pre></details>
  </div>;
}

const taskStatusLabels: Record<AiTaskStatus, string> = {
  pending: '待执行', running: '执行中', pending_approval: '待本人确认', succeeded: '已完成', failed: '执行失败', rejected: '已拒绝',
};

export function WorkflowRunResultNotice({ result }: { result: AiRunWorkflowResp }) {
  return <Alert type="success" showIcon title={<Space>已创建任务 #{result.task_id}<Tag>{taskStatusLabels[result.status] ?? '未知状态'}</Tag></Space>} description="当前没有查看任务权限，请联系管理员授予任务查看权限后核对进展。" />;
}
