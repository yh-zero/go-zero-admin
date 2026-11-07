import type { AiToolInfo } from '../../api/types';

export interface WorkflowStepDraft { key: string; id?: string; name: string; tool: string; input: string; approval: boolean; toolInputs?: Record<string, string> }
export interface WorkflowDraft { name: string; enabled: boolean; steps: WorkflowStepDraft[] }
export interface WorkflowDraftSeed { key: string; draft: WorkflowDraft }

export function retainWorkflowDraftSeed(current: WorkflowDraftSeed | null, key: string | undefined, initial?: WorkflowDraft): WorkflowDraftSeed | null {
  if (!key) return null;
  if (current?.key === key) return current;
  return initial ? { key, draft: initial } : null;
}

const byteLength = (value: string) => new TextEncoder().encode(value).length;
const isObject = (value: unknown): value is Record<string, unknown> => value !== null && typeof value === 'object' && !Array.isArray(value);

export const workflowToolLabels: Record<string, string> = { kb_analyze: '知识库分析', kb_search: '知识库检索', text_stats: '文本统计', json_formatter: 'JSON 整理', get_current_time: '当前时间' };
export const workflowFieldLabels: Record<string, string> = { kb_id: '知识库', query: '分析目标 / 检索问题', text: '处理文本', timezone: '时区', format: '输出格式', indent: '缩进', pretty: '美化输出' };
export interface SimpleToolField { name: string; label: string; type: string; required: boolean; description?: string; enum?: (string | number | boolean)[]; minimum?: number; maximum?: number }

function readToolSchema(tool: AiToolInfo): Record<string, unknown> {
  let schema: unknown;
  try { schema = JSON.parse(tool.parameters_schema); } catch { throw new Error('工具参数 Schema 无效，请刷新工具目录'); }
  if (!isObject(schema) || schema.type !== 'object') throw new Error('工具参数 Schema 须为对象，暂不可安全保存');
  validateSchemaDefinition(schema);
  return schema;
}

function validateSchemaDefinition(rule: Record<string, unknown>) {
  const supported = ['type', 'properties', 'required', 'additionalProperties', 'description', 'title', '$schema', 'enum', 'minimum', 'maximum', 'minLength', 'maxLength', 'default', 'items'];
  if (Object.keys(rule).some(key => !supported.includes(key)) || !['object', 'array', 'string', 'number', 'integer', 'boolean'].includes(String(rule.type))) throw new Error('工具参数 Schema 含暂不支持的约束，无法安全保存');
  for (const key of ['minimum', 'maximum', 'minLength', 'maxLength']) if (rule[key] !== undefined && (typeof rule[key] !== 'number' || !Number.isFinite(rule[key]))) throw new Error('工具参数 Schema 数值约束无效');
  if (rule.enum !== undefined && (!Array.isArray(rule.enum) || rule.enum.length === 0)) throw new Error('工具参数 Schema 选项无效');
  if (rule.type === 'object') {
    if (rule.properties !== undefined && !isObject(rule.properties)) throw new Error('工具参数 Schema 字段无效');
    if (rule.additionalProperties !== undefined && typeof rule.additionalProperties !== 'boolean') throw new Error('工具参数 Schema 含暂不支持的扩展字段约束');
    if (rule.required !== undefined && (!Array.isArray(rule.required) || rule.required.some(name => typeof name !== 'string' || !Object.hasOwn(rule.properties ?? {}, name)))) throw new Error('工具参数 Schema 必填字段无效');
    for (const child of Object.values(rule.properties ?? {})) {
      if (!isObject(child)) throw new Error('工具参数 Schema 字段无效');
      validateSchemaDefinition(child);
    }
  }
  if (rule.type === 'array') {
    if (!isObject(rule.items)) throw new Error('工具参数 Schema 数组约束无效');
    validateSchemaDefinition(rule.items);
  }
}

export function simpleToolFields(tool: AiToolInfo): SimpleToolField[] | null {
  try {
    const schema = readToolSchema(tool);
    if (Object.keys(schema).some(key => !['type', 'properties', 'required', 'additionalProperties', 'description', 'title', '$schema'].includes(key))) return null;
    if (schema.additionalProperties && schema.additionalProperties !== false) return null;
    const properties = isObject(schema.properties) ? schema.properties : {};
    const fields: SimpleToolField[] = [];
    for (const [name, rule] of Object.entries(properties)) {
      if (!isObject(rule) || !['string', 'integer', 'number', 'boolean'].includes(String(rule.type))) return null;
      if (Object.keys(rule).some(key => !['type', 'description', 'title', 'enum', 'minimum', 'maximum', 'minLength', 'maxLength', 'default'].includes(key))) return null;
      if (rule.enum !== undefined && (!Array.isArray(rule.enum) || rule.enum.some(item => !['string', 'number', 'boolean'].includes(typeof item)))) return null;
      fields.push({ name, label: workflowFieldLabels[name] ?? (typeof rule.title === 'string' ? rule.title : name), type: String(rule.type), required: Array.isArray(schema.required) && schema.required.includes(name), description: typeof rule.description === 'string' ? rule.description : undefined,
        enum: rule.enum as SimpleToolField['enum'], minimum: typeof rule.minimum === 'number' ? rule.minimum : undefined, maximum: typeof rule.maximum === 'number' ? rule.maximum : undefined });
    }
    return fields;
  } catch { return null; }
}

export function updateStepInputField(raw: string, field: string, value: unknown): string {
  let input: unknown;
  try { input = JSON.parse(raw); } catch { throw new Error('请先修正高级 JSON，现有参数已保留'); }
  if (!isObject(input)) throw new Error('步骤参数必须为 JSON 对象');
  if (value === undefined) delete input[field]; else input[field] = value;
  return JSON.stringify(input, null, 2);
}

function validateSchema(value: unknown, rule: Record<string, unknown>, path: string) {
  if (isWorkflowReference(value)) { validateReferencePath(value.$ref); return; }
  const type = rule.type;
  if (type === 'object') {
    if (!isObject(value)) throw new Error(`${path}必须为 JSON 对象`);
    const properties = isObject(rule.properties) ? rule.properties : {};
    rejectUnknown(value, Object.keys(properties));
    if (Array.isArray(rule.required)) for (const name of rule.required) if (typeof name === 'string' && !Object.hasOwn(value, name)) throw new Error(`${workflowFieldLabels[name] ?? name} (${name}) 为必填字段`);
    for (const [name, item] of Object.entries(value)) {
      const child = properties[name];
      if (!isObject(child)) throw new Error('工具参数 Schema 无效');
      validateSchema(item, child, `${workflowFieldLabels[name] ?? name} (${name}) `);
    }
  } else if (type === 'array') {
    if (!Array.isArray(value) || !isObject(rule.items)) throw new Error(`${path}须为符合 Schema 的数组`);
    for (const item of value) validateSchema(item, rule.items, path);
  } else if (type === 'string') {
    if (typeof value !== 'string') throw new Error(`${path}必须为字符串`);
    if (typeof rule.minLength === 'number' && Array.from(value).length < rule.minLength) throw new Error(`${path}内容太短`);
    if (typeof rule.maxLength === 'number' && Array.from(value).length > rule.maxLength) throw new Error(`${path}内容太长`);
  } else if (type === 'integer' || type === 'number') {
    if (typeof value !== 'number' || !Number.isFinite(value) || type === 'integer' && !Number.isSafeInteger(value)) throw new Error(`${path}必须为${type === 'integer' ? '整数' : '数字'}`);
    if (typeof rule.minimum === 'number' && value < rule.minimum || typeof rule.maximum === 'number' && value > rule.maximum) throw new Error(`${path}超出允许范围`);
  } else if (type === 'boolean') {
    if (typeof value !== 'boolean') throw new Error(`${path}必须为布尔值`);
  } else throw new Error('工具参数 Schema 类型暂不支持，无法安全保存');
  if (Array.isArray(rule.enum) && !rule.enum.includes(value)) throw new Error(`${path}请选择有效选项`);
}

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
    validateSchema(input, readToolSchema(tool), '步骤参数');
    if (tool.name === 'kb_analyze' || tool.name === 'kb_search') {
      if (!isWorkflowReference(input.query) && (typeof input.query !== 'string' || !input.query.trim() || byteLength(input.query) > 2000)) throw new Error(`${tool.name === 'kb_analyze' ? '分析目标' : '检索问题'} (query) 不能为空白且最多 2000 UTF-8 字节`);
      if (tool.name === 'kb_analyze' || input.kb_id !== undefined) {
        if (!isWorkflowReference(input.kb_id) && (!Number.isSafeInteger(input.kb_id) || Number(input.kb_id) < (tool.name === 'kb_analyze' ? 1 : 0))) throw new Error(`知识库 (kb_id) 须为${tool.name === 'kb_analyze' ? '正' : '非负'}整数`);
      }
    }
  }
  return input;
}

export function newWorkflowStep(tool = ''): WorkflowStepDraft {
  return {
    key: crypto.randomUUID(), name: workflowToolLabels[tool] ?? '新步骤', tool,
    input: tool === 'text_stats' || tool === 'json_formatter' ? '{\n  "text": ""\n}' : tool === 'kb_analyze' || tool === 'kb_search' ? '{\n  "query": ""\n}' : '{}', approval: false,
  };
}

export function changeWorkflowStepTool(step: WorkflowStepDraft, tool: string): WorkflowStepDraft {
  const toolInputs = { ...step.toolInputs, [step.tool]: step.input };
  return { ...step, tool, toolInputs, input: toolInputs[tool] ?? newWorkflowStep(tool).input };
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
  const inputs = steps.map(step => parseStepInput(step.input, tools?.find(tool => tool.name === step.tool)));
  const v2 = steps.some(step => step.id) || inputs.some(input => workflowReferences(input).length > 0);
  const prior = new Set<string>();
  const definition = JSON.stringify({ ...(v2 ? { version: 2 } : {}), steps: steps.map((step, index) => {
    const nameError = validateWorkflowName(step.name);
    if (nameError) throw new Error(`步骤${nameError}`);
    const tool = tools?.find(item => item.name === step.tool);
    if (typeof step.tool !== 'string' || !step.tool || (tools && !tool)) throw new Error('请选择已注册工具');
    if (typeof step.approval !== 'boolean') throw new Error('步骤审批开关必须为布尔值');
    const id = step.id ?? step.key;
    if (v2 && (!/^[A-Za-z0-9_-]{1,64}$/.test(id) || prior.has(id))) throw new Error('步骤 ID 无效或重复');
    for (const ref of workflowReferences(inputs[index])) {
      const parts = validateReferencePath(ref);
      if (parts[0] === 'steps' && !prior.has(parts[1])) throw new Error('只能引用前序步骤；请调整步骤或重新选择变量');
    }
    prior.add(id);
    return { ...(v2 ? { id } : {}), name: step.name, tool: step.tool, input: inputs[index], approval: step.approval };
  }) });
  if (byteLength(definition) > 65536) throw new Error('工作流定义最多 65536 UTF-8 字节');
  return definition;
}

export function parseWorkflow(raw: string, tools?: readonly AiToolInfo[]): WorkflowStepDraft[] {
  if (!raw || byteLength(raw) > 65536) throw new Error('工作流定义须为 1..65536 UTF-8 字节');
  let value: unknown;
  try { value = JSON.parse(raw); } catch { throw new Error('工作流定义必须为有效 JSON'); }
  if (!isObject(value)) throw new Error('工作流定义必须为 JSON 对象');
  rejectUnknown(value, ['steps', 'version']);
  if (value.version !== undefined && value.version !== 2) throw new Error('工作流版本不支持');
  if (!Array.isArray(value.steps) || value.steps.length < 1 || value.steps.length > 32) throw new Error('工作流须包含 1..32 个步骤');
  const steps = value.steps.map((step): WorkflowStepDraft => {
    if (!isObject(step)) throw new Error('步骤必须为 JSON 对象');
    rejectUnknown(step, ['id', 'name', 'tool', 'input', 'approval']);
    if (value.version === 2 && (typeof step.id !== 'string' || !/^[A-Za-z0-9_-]{1,64}$/.test(step.id))) throw new Error('步骤 ID 无效');
    if (typeof step.name !== 'string' || typeof step.tool !== 'string') throw new Error('步骤名称和工具必须为字符串');
    if (!isObject(step.input)) throw new Error('步骤参数必须为 JSON 对象');
    if (step.approval !== undefined && typeof step.approval !== 'boolean') throw new Error('步骤审批开关必须为布尔值');
    return { key: typeof step.id === 'string' ? step.id : crypto.randomUUID(), ...(typeof step.id === 'string' ? { id: step.id } : {}), name: step.name, tool: step.tool, input: JSON.stringify(step.input, null, 2), approval: step.approval ?? false };
  });
  serializeWorkflow(steps, tools);
  return steps;
}

export function validateRunInput(input: string): string | undefined {
  if (byteLength(input) > 16000) return '运行输入最多 16000 UTF-8 字节';
  if (input === '') return;
  try { if (!isObject(JSON.parse(input))) return '运行输入必须为 JSON 对象，或留空'; } catch { return '运行输入必须为有效 JSON，或留空'; }
}

export function isWorkflowReference(value: unknown): value is { $ref: string } {
  return isObject(value) && Object.hasOwn(value, '$ref') && Object.keys(value).length === 1 && typeof value.$ref === 'string';
}

function validateReferencePath(ref: string) {
  const parts = ref.split('.');
  if (parts.length < 2 || parts.length > 8 || !['input', 'steps'].includes(parts[0]) || parts.some(part => !part || ['__proto__', 'prototype', 'constructor'].includes(part))) throw new Error('变量引用路径无效');
  return parts;
}

export function workflowReferences(value: unknown, depth = 0): string[] {
  if (depth > 8) throw new Error('参数嵌套最多 8 层');
  if (isObject(value) && Object.hasOwn(value, '$ref')) {
    if (!isWorkflowReference(value)) throw new Error('变量引用须为单键 $ref 对象');
    validateReferencePath(value.$ref); return [value.$ref];
  }
  if (Array.isArray(value)) return value.flatMap(child => workflowReferences(child, depth + 1));
  if (isObject(value)) {
    if (Object.keys(value).some(key => ['__proto__', 'prototype', 'constructor'].includes(key))) throw new Error('参数字段无效');
    return Object.values(value).flatMap(child => workflowReferences(child, depth + 1));
  }
  return [];
}

export function workflowRunFields(definition: string) {
  const steps = parseWorkflow(definition);
  const refs = steps.flatMap(step => workflowReferences(JSON.parse(step.input)));
  return { question: refs.some(ref => ref === 'input.question'), text: refs.some(ref => ref === 'input.text'), kb_id: refs.some(ref => ref === 'input.kb_id') };
}

export function prepareWorkflowRunInput(values: { input?: string; question?: string; text?: string; kb_id?: number }) {
  const error = validateRunInput(values.input ?? ''); if (error) throw new Error(error);
  const input: Record<string, unknown> = values.input ? JSON.parse(values.input) : {};
  if (values.question !== undefined) input.question = values.question;
  if (values.text !== undefined) input.text = values.text;
  if (values.kb_id !== undefined) {
    if (!Number.isSafeInteger(values.kb_id) || values.kb_id < 1) throw new Error('请选择有效的知识库');
    input.kb_id = values.kb_id;
  }
  const raw = JSON.stringify(input); if (byteLength(raw) > 16000) throw new Error('运行输入最多 16000 UTF-8 字节');
  return raw;
}

export function copyWorkflowStep(step: WorkflowStepDraft): WorkflowStepDraft {
  const key = crypto.randomUUID(); return { ...step, key, ...(step.id ? { id: key } : {}), name: `${Array.from(step.name).slice(0, 125).join('')} 副本`, toolInputs: step.toolInputs ? { ...step.toolInputs } : undefined };
}

export function workflowTemplate(tool: string, tools: readonly AiToolInfo[]): WorkflowDraft {
  if (tool === 'kb_analyze') {
    const first = newWorkflowStep(tool); first.name = '分析知识库资料'; first.input = JSON.stringify({ kb_id: { $ref: 'input.kb_id' }, query: { $ref: 'input.question' } }, null, 2);
    return { name: '资料分析', enabled: true, steps: [first] };
  }
  if (tool === 'json_formatter') {
    const first = newWorkflowStep(tool); first.name = '整理本次 JSON'; first.input = JSON.stringify({ text: { $ref: 'input.text' } }, null, 2);
    return { name: 'JSON 格式整理', enabled: true, steps: [first] };
  }
  if (tool === 'text_stats' && tools.some(item => item.name === 'json_formatter')) {
    const first = newWorkflowStep('json_formatter'); first.name = '整理本次 JSON'; first.input = JSON.stringify({ text: { $ref: 'input.text' } }, null, 2);
    const next = newWorkflowStep('text_stats'); next.name = '统计整理结果'; next.input = JSON.stringify({ text: { $ref: `steps.${first.key}` } }, null, 2);
    return { name: 'JSON 整理与统计', enabled: true, steps: [first, next] };
  }
  const first = newWorkflowStep(tool);
  return { name: tool ? `${workflowToolLabels[tool] ?? tool}工作流` : '我的工作流', enabled: true, steps: [first] };
}
