export interface ToolGuide {
  title: string;
  purpose: string;
  input: string;
  output: string;
  note?: string;
}

// 示例帮助理解用途；可用工具及参数始终以服务端注册表为准。
export const toolGuides: Record<string, ToolGuide> = {
  get_current_time: {
    title: '当前时间', purpose: '在流程中取得上海时区的当前服务器时间。',
    input: '{}', output: '2026-11-07 09:00:00 CST', note: '示例时间仅展示格式，运行时返回实际时间。',
  },
  text_stats: {
    title: '文本统计', purpose: '统计输入文字或上一步结果的字符、单词和行数。',
    input: '{"text":"你好 世界"}', output: '{"chars":5,"words":2,"lines":1}',
  },
  json_formatter: {
    title: 'JSON 整理', purpose: '检查并排版 JSON，便于阅读结构化数据。',
    input: JSON.stringify({ text: '{"name":"示例"}' }), output: '{\n  "name": "示例"\n}', note: '输入必须是合法 JSON 文本，非法内容会返回失败。',
  },
  kb_search: {
    title: '知识库检索', purpose: '按问题查找资料原文片段，适合先确认是否有相关依据。',
    input: '{"kb_id":1,"query":"发布失败后如何处理？"}',
    output: '返回命中文档、原文片段、来源标识与相似度；无命中时返回空列表。',
    note: '请使用自己的知识库；返回的是候选片段，相似度不等于答案正确率。检索会调用 Embedding。',
  },
  kb_analyze: {
    title: '知识库分析', purpose: '选择资料库和分析目标，生成附参考片段的分析。',
    input: '{"kb_id":1,"query":"整理发布失败后的处理步骤"}',
    output: '返回分析正文、实际使用的资料片段和已知聊天用量。',
    note: '仅用于工作流。会调用检索及聊天模型；资料不足时应明确说明。',
  },
};
