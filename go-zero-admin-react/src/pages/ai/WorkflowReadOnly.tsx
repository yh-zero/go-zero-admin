import { Alert, Empty, Space, Tag, Typography } from 'antd';
import type { AiRunWorkflowResp, AiTaskStatus } from '../../api/types';
import { parseWorkflow } from './workflowDefinition';
import type { WorkflowStepDraft } from './workflowDefinition';
import styles from './workflow.module.css';

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
  pending: '待执行', running: '执行中', pending_approval: '待本人确认', succeeded: '已完成', failed: '执行失败', rejected: '已拒绝', cancelled: '已取消',
};

export function WorkflowRunResultNotice({ result }: { result: AiRunWorkflowResp }) {
  return <Alert type="success" showIcon title={<Space>已创建任务 #{result.task_id}<Tag>{taskStatusLabels[result.status] ?? '未知状态'}</Tag></Space>} description="当前没有查看任务权限，请联系管理员授予任务查看权限后核对进展。" />;
}
