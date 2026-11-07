import { memo } from 'react';
import { Tag, Typography } from 'antd';
import type { AiChatMessage } from '../../api/types';
import { formatTime } from '../../utils/format';
import { CopyTextButton, SafeMarkdown } from './SafeMarkdown';
import { ChatMessageFeedback } from './ChatMessageFeedback';
import type { ChatFeedbackContext } from './ChatMessageFeedback';
import styles from './chat.module.css';

interface Props { message: AiChatMessage; feedbackContext?: ChatFeedbackContext }
interface ToolCallRecord { id?: string; type?: string; function: { name: string; arguments: string }; key: string }

function readToolCalls(raw: string): ToolCallRecord[] | null {
  if (!raw) return [];
  try {
    const value: unknown = JSON.parse(raw);
    if (!Array.isArray(value)) return null;
    if (!value.every(call => call && typeof call === 'object' && 'function' in call && call.function
      && typeof call.function.name === 'string' && typeof call.function.arguments === 'string')) return null;
    const occurrences = new Map<string, number>();
    return value.map(call => {
      const identity = call.id || JSON.stringify(call.function);
      const occurrence = (occurrences.get(identity) ?? 0) + 1;
      occurrences.set(identity, occurrence);
      return { ...call, key: `${identity}:${occurrence}` } as ToolCallRecord;
    });
  } catch { return null; }
}

// 工具参数美化：text_stats/json_formatter 的参数是 JSON，做缩进格式化。
function formatToolArgs(name: string, args: string): string {
  if (name !== 'text_stats' && name !== 'json_formatter') return args;
  try {
    return JSON.stringify(JSON.parse(args), null, 2);
  } catch { return args; }
}

const statuses = {
  running: { color: 'processing', title: '处理中' },
  succeeded: { color: 'success', title: '已完成' },
  failed: { color: 'error', title: '失败' },
};

export const MessageRecord = memo(function MessageRecord({ message, feedbackContext }: Props) {
  const status = statuses[message.status] ?? (message.status ? { color: 'default', title: message.status } : statuses.succeeded);
  const calls = readToolCalls(message.tool_calls);
  return <article className={`${styles.message} ${message.role === 'user' ? styles.userMessage : ''}`}>
    <div className={styles.messageMeta}>
      <strong>{message.role === 'assistant' ? 'AI 助手' : message.role === 'user' ? '我' : message.role}</strong>
      <Tag color={status.color}>{status.title}</Tag>
      <Typography.Text type="secondary">{formatTime(message.created_at)}</Typography.Text>
    </div>
    {message.content && (message.role === 'assistant' ? <SafeMarkdown content={message.content} /> : <div className={styles.messageText}>{message.content}</div>)}
    {message.content && <CopyTextButton text={message.content} label={message.role === 'assistant' ? '复制回答' : '复制内容'} />}
    {message.role === 'assistant' && (message.sources ?? []).length > 0 && <div className={styles.sources}><Typography.Text strong>参考资料（实际检索片段）</Typography.Text>{message.sources.map((source, position) => <details key={`${source.kb_id}:${source.chunk_id}`}>
      <summary>[{position + 1}] {source.doc_title || `文档 #${source.doc_id}`}</summary><Typography.Paragraph type="secondary">知识库 ID：{source.kb_id} · 文档 ID：{source.doc_id} · 分块 ID：{source.chunk_id} · 相似度：{source.score}</Typography.Paragraph><div className={styles.messageText}>{source.snippet}</div>
    </details>)}</div>}
    {message.role === 'assistant' && message.status === 'succeeded' && feedbackContext && <ChatMessageFeedback message={message} context={feedbackContext} />}
    {message.error && <Typography.Paragraph type="danger" className={styles.messageText}>{message.error}</Typography.Paragraph>}
    {message.role === 'assistant' && <Typography.Paragraph type="secondary" className={styles.usage}>
      已知 token：{message.tokens}。用量可能不完整，不代表最终计费。
    </Typography.Paragraph>}
    {calls === null ? <Typography.Paragraph type="secondary">工具调用记录格式异常</Typography.Paragraph> : calls.length > 0 && <div className={styles.tools}>
      <Typography.Text type="secondary">工具调用记录（仅名称与参数）</Typography.Text>
      {calls.map(call => <div className={styles.tool} key={call.key}>
        <strong>{call.function.name}</strong>
        <pre>{formatToolArgs(call.function.name, call.function.arguments)}</pre>
      </div>)}
    </div>}
  </article>;
});
