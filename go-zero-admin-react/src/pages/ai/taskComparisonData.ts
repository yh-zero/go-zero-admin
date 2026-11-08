import type { AiTaskInfo, AiWorkflowStepInfo } from '../../api/types';

export interface TaskStepPair {
  key: string; left?: AiWorkflowStepInfo; right?: AiWorkflowStepInfo;
  moved: boolean; inputChanged: boolean; outputChanged: boolean; statusChanged: boolean;
}
export const isTerminalTask = (status: string) => ['succeeded', 'failed', 'rejected', 'cancelled'].includes(status);
export function recordedElapsed(started: number, completed: number): number | null {
  return Number.isSafeInteger(started) && started > 0 && Number.isSafeInteger(completed) && completed >= started ? completed - started : null;
}
function hasStableSteps(task: AiTaskInfo): boolean {
  try {
    const snapshot = JSON.parse(task.definition) as { version?: unknown; steps?: { id?: unknown }[] } | null;
    if (snapshot?.version !== 2 || !Array.isArray(snapshot.steps)) return false;
    const ids = snapshot.steps.map(step => step?.id);
    if (ids.some(id => typeof id !== 'string' || !id) || new Set(ids).size !== ids.length) return false;
    const recorded = (Array.isArray(task.steps) ? task.steps : []).map(step => step.id);
    return new Set(recorded).size === recorded.length && recorded.every(id => !!id && ids.includes(id));
  } catch { return false; }
}
export function pairTaskSteps(left: AiTaskInfo, right: AiTaskInfo) {
  const stable = left.workflow_id > 0 && left.workflow_id === right.workflow_id && hasStableSteps(left) && hasStableSteps(right);
  const a = [...(Array.isArray(left.steps) ? left.steps : [])].sort((one, two) => one.index - two.index);
  const b = [...(Array.isArray(right.steps) ? right.steps : [])].sort((one, two) => one.index - two.index);
  const pairs: { left?: AiWorkflowStepInfo; right?: AiWorkflowStepInfo }[] = [];
  if (stable) {
    const rightById = new Map(b.map(step => [step.id, step]));
    for (const step of a) { pairs.push({ left: step, right: rightById.get(step.id) }); rightById.delete(step.id); }
    for (const step of rightById.values()) pairs.push({ right: step });
  } else {
    for (let position = 0; position < Math.max(a.length, b.length); position++) pairs.push({ left: a[position], right: b[position] });
  }
  const rows: TaskStepPair[] = pairs.map(pair => ({ ...pair,
    key: `${left.id}:${pair.left?.id ?? ''}:${pair.left?.index ?? ''}|${right.id}:${pair.right?.id ?? ''}:${pair.right?.index ?? ''}`,
    moved: !!(stable && pair.left && pair.right && pair.left.index !== pair.right.index),
    inputChanged: !!(pair.left && pair.right && pair.left.input !== pair.right.input),
    outputChanged: !!(pair.left && pair.right && pair.left.output !== pair.right.output),
    statusChanged: !!(pair.left && pair.right && pair.left.status !== pair.right.status),
  }));
  return { mode: stable ? 'stable_id' as const : 'order' as const, rows,
    explanation: stable ? '同一流程按两次任务快照中的稳定步骤ID配对；位置变化单独标记。'
      : left.workflow_id <= 0 || left.workflow_id !== right.workflow_id ? '不同流程或一次性运行按执行顺序对照，不表示对应步骤职责相同。'
        : '旧记录或原快照缺少可确认的稳定步骤ID，按执行顺序对照。',
  };
}
