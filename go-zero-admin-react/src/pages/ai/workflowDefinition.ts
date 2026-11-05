import type { AiToolInfo } from '../../api/types';

export interface WorkflowStepDraft { key: string; name: string; tool: string; input: string; approval: boolean }
export interface WorkflowDraft { name: string; enabled: boolean; steps: WorkflowStepDraft[] }
export interface WorkflowDraftSeed { key: string; draft: WorkflowDraft }

export function retainWorkflowDraftSeed(current: WorkflowDraftSeed | null, key: string | undefined, initial?: WorkflowDraft): WorkflowDraftSeed | null {
  if (!key) return null;
  if (current?.key === key) return current;
  return initial ? { key, draft: initial } : null;
}

const byteLength = (value: string) => new TextEncoder().encode(value).length;
const isObject = (value: unknown): value is Record<string, unknown> => value !== null && typeof value === 'object' && !Array.isArray(value);

export function validateWorkflowName(name: string): string | undefined {
  if (!name.trim() || Array.from(name).length > 128) return '名称不能为空且最多 128 个 Unicode 字符';
}

function rejectUnknown(value: Record<string, unknown>, fields: readonly string[]) {
  if (Object.keys(value).some(key => !fields.includes(key))) throw new Error('包含未知字段，请核对定义或工具参数');
}

export function parseStepInput(raw: string, tool?: AiToolInfo): Record<string, unknown> {
  let input: unknown;
  try { input = JSON.parse(raw); } catch { throw new Error('步骤参数必须为有效 JSON 对象'); }
  if (!isObject(input)) throw new Error('步骤参数必须为 JSON 对象');
  if (typeof input.text === 'string' && byteLength(input.text) > 16000) throw new Error('text 最多 16000 UTF-8 字节');
  if (tool) {
    let schema: unknown;
    try { schema = JSON.parse(tool.parameters_schema); } catch { throw new Error('工具参数 Schema 无效，请刷新工具目录'); }
    if (!isObject(schema)) throw new Error('工具参数 Schema 无效，请刷新工具目录');
    const properties = isObject(schema.properties) ? schema.properties : {};
    rejectUnknown(input, Object.keys(properties));
    if (Array.isArray(schema.required)) {
      for (const field of schema.required) {
        if (typeof field === 'string' && !Object.hasOwn(input, field)) throw new Error(`工具参数缺少必填字段 ${field}`);
      }
    }
    for (const [field, value] of Object.entries(input)) {
      const rule = properties[field];
      if (isObject(rule) && rule.type === 'string' && typeof value !== 'string') throw new Error(`工具参数 ${field} 必须为字符串`);
    }
  }
  return input;
}

export function newWorkflowStep(tool = ''): WorkflowStepDraft {
  return {
    key: crypto.randomUUID(), name: '新步骤', tool,
    input: tool === 'text_stats' || tool === 'json_formatter' ? '{\n  "text": ""\n}' : '{}', approval: false,
  };
}

export function moveWorkflowStep(steps: WorkflowStepDraft[], key: string, direction: -1 | 1): WorkflowStepDraft[] {
  const index = steps.findIndex(step => step.key === key);
  const target = index + direction;
  if (index < 0 || target < 0 || target >= steps.length) return steps;
  const next = [...steps];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

export function serializeWorkflow(steps: WorkflowStepDraft[], tools?: readonly AiToolInfo[]): string {
  if (steps.length < 1 || steps.length > 32) throw new Error('工作流须包含 1..32 个步骤');
  const definition = JSON.stringify({ steps: steps.map(step => {
    const nameError = validateWorkflowName(step.name);
    if (nameError) throw new Error(`步骤${nameError}`);
    const tool = tools?.find(item => item.name === step.tool);
    if (typeof step.tool !== 'string' || !step.tool || (tools && !tool)) throw new Error('请选择已注册工具');
    if (typeof step.approval !== 'boolean') throw new Error('步骤审批开关必须为布尔值');
    return { name: step.name, tool: step.tool, input: parseStepInput(step.input, tool), approval: step.approval };
  }) });
  if (byteLength(definition) > 65536) throw new Error('工作流定义最多 65536 UTF-8 字节');
  return definition;
}

export function parseWorkflow(raw: string, tools?: readonly AiToolInfo[]): WorkflowStepDraft[] {
  if (!raw || byteLength(raw) > 65536) throw new Error('工作流定义须为 1..65536 UTF-8 字节');
  let value: unknown;
  try { value = JSON.parse(raw); } catch { throw new Error('工作流定义必须为有效 JSON'); }
  if (!isObject(value)) throw new Error('工作流定义必须为 JSON 对象');
  rejectUnknown(value, ['steps']);
  if (!Array.isArray(value.steps) || value.steps.length < 1 || value.steps.length > 32) throw new Error('工作流须包含 1..32 个步骤');
  const steps = value.steps.map((step): WorkflowStepDraft => {
    if (!isObject(step)) throw new Error('步骤必须为 JSON 对象');
    rejectUnknown(step, ['name', 'tool', 'input', 'approval']);
    if (typeof step.name !== 'string' || typeof step.tool !== 'string') throw new Error('步骤名称和工具必须为字符串');
    if (!isObject(step.input)) throw new Error('步骤参数必须为 JSON 对象');
    if (step.approval !== undefined && typeof step.approval !== 'boolean') throw new Error('步骤审批开关必须为布尔值');
    return { key: crypto.randomUUID(), name: step.name, tool: step.tool, input: JSON.stringify(step.input, null, 2), approval: step.approval ?? false };
  });
  serializeWorkflow(steps, tools);
  return steps;
}

export function validateRunInput(input: string): string | undefined {
  if (byteLength(input) > 16000) return '运行输入最多 16000 UTF-8 字节';
  if (input === '') return;
  try { JSON.parse(input); } catch { return '运行输入必须为有效 JSON，或留空'; }
}
