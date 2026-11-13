import type { AiCapabilities, AiToolInfo } from '../../api/types';

export interface ToolGuide {
  title: string;
  category: string;
  order: number;
  purpose: string;
  input: string;
  output: string;
  note?: string;
}

// 示例帮助理解用途；可用工具及参数始终以服务端注册表为准。
export const toolGuides: Record<string, ToolGuide> = {
  get_current_time: {
    title: '当前时间', category: '基础能力', order: 5, purpose: '在流程中取得上海时区的当前服务器时间。',
    input: '{}', output: '2026-11-07 09:00:00 CST', note: '示例时间仅展示格式，运行时返回实际时间。',
  },
  text_stats: {
    title: '文本统计', category: '文本处理', order: 4, purpose: '统计输入文字或上一步结果的字符、单词和行数。',
    input: '{"text":"你好 世界"}', output: '{"chars":5,"words":2,"lines":1}',
  },
  json_formatter: {
    title: 'JSON 格式整理', category: '文本处理', order: 3, purpose: '输入合法 JSON，输出清晰排版，便于阅读结构化数据。',
    input: JSON.stringify({ text: '{"name":"示例"}' }), output: '{\n  "name": "示例"\n}', note: '输入必须是合法 JSON 文本，非法内容会返回失败。',
  },
  kb_search: {
    title: '查找资料', category: '知识库', order: 2, purpose: '查找与问题有关的资料片段，适合先核对是否存在相关依据。',
    input: '{"kb_id":1,"query":"发布失败后如何处理？"}',
    output: '返回命中文档、原文片段、来源标识与相似度；无命中时返回空列表。',
    note: '请使用自己的知识库；返回的是候选片段，相似度不等于答案正确率。检索会调用 Embedding。',
  },
  kb_analyze: {
    title: '资料分析', category: '知识库', order: 1, purpose: '分析资料并提炼结论，保留引用来源，说明哪些信息尚不足。',
    input: '{"kb_id":1,"query":"整理发布失败后的处理步骤"}',
    output: '返回分析正文、实际使用的资料片段和已知聊天用量。',
    note: '仅用于常用流程。会调用检索及聊天模型；资料不足时应明确说明。',
  },
};

export function filterToolsByPurpose(tools: AiToolInfo[], search: string): AiToolInfo[] {
  const words = search.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return tools.filter(tool => {
    const guide = toolGuides[tool.name];
    const content = [tool.name, tool.description, guide?.title, guide?.category, guide?.purpose].join('\n').toLocaleLowerCase();
    return words.every(word => content.includes(word));
  }).sort((left, right) => (toolGuides[left.name]?.order ?? 99) - (toolGuides[right.name]?.order ?? 99) || left.name.localeCompare(right.name));
}

export function getToolAvailability(name: string, capabilities?: AiCapabilities): { label: string; description: string; color: 'success' | 'warning' | 'default' } {
  if (name !== 'kb_search' && name !== 'kb_analyze') {
    return toolGuides[name]
      ? { label: '本地能力', description: '无需调用模型。', color: 'success' }
      : { label: '已注册', description: '能力与参数以服务端注册定义为准。', color: 'default' };
  }
  if (!capabilities) return { label: '配置状态未知', description: '暂未取得模型配置状态，请刷新后确认。', color: 'default' };
  if (!capabilities.retrieval_configured) return { label: '检索能力未配置', description: '运行前需由管理员配置 Embedding 模型。', color: 'warning' };
  if (name === 'kb_analyze' && !capabilities.analysis_configured) return { label: '分析模型未配置', description: '检索已配置，运行分析前还需配置聊天模型。', color: 'warning' };
  return name === 'kb_analyze'
    ? { label: '资料分析已配置', description: '运行会调用检索和聊天模型；资料不足时会说明。', color: 'success' }
    : { label: '资料检索已配置', description: '检索会调用 Embedding 模型，相似度不表示答案正确率。', color: 'success' };
}
