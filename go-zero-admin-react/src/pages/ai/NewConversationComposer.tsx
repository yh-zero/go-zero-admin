import { useRef, useState } from 'react';
import { Alert, Button, Card, Form, Input, Tag, Typography } from 'antd';
import { SendOutlined } from '@ant-design/icons';
import { sendAiMessageStream } from '../../api/aiStream';
import { validateMessageContent } from './chatState';
import styles from './chat.module.css';

type ComposerValues = { content: string };

interface Props {
  canSend: boolean;
  writing: boolean;
  isCurrent: () => boolean;
  beginWrite: (permission: string) => boolean;
  finishWrite: () => void;
  onCreated: (conversationId: number) => Promise<void>;
}

// 新会话作曲家：不等创建会话 API，直接输入并发送；conversation_id=0 由后端自动建会话并命名。
// 流式 delta 实时渲染在本组件内，完成后切到正式会话视图。
export function NewConversationComposer({ canSend, writing, isCurrent, beginWrite, finishWrite, onCreated }: Props) {
  const [form] = Form.useForm<ComposerValues>();
  const draft = Form.useWatch('content', form) ?? '';
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // 流式回复草稿：本地状态实时追加 delta。
  const [streamState, setStreamState] = useState<{ content: string; tools: { key: number; name: string; status: string }[] } | null>(null);
  const [conversationId, setConversationId] = useState<number | null>(null);
  const visible = useRef<HTMLDivElement>(null);

  async function handleSend(values: ComposerValues) {
    const validationError = validateMessageContent(values.content);
    if (validationError || !beginWrite('sys:ai:message:send')) return;
    setSending(true);
    setError(null);
    setStreamState({ content: '', tools: [] });
    setConversationId(null);
    try {
      const request = { request_id: crypto.randomUUID(), conversation_id: 0, content: values.content };
      const meta = await sendAiMessageStream(request, {
        onDelta: content => {
          if (isCurrent()) setStreamState(s => (s ? { ...s, content: s.content + content } : s));
        },
        onTool: (name, status) => {
          if (isCurrent()) setStreamState(s => (s ? { ...s, tools: [...s.tools, { key: s.tools.length, name, status }] } : s));
        },
      });
      if (!isCurrent()) return;
      form.setFieldValue('content', '');
      setConversationId(meta.conversation_id);
      // 流式完成，切到正式会话
      await onCreated(meta.conversation_id);
    } catch (err) {
      if (isCurrent()) setError(err instanceof Error ? err.message : '发送失败');
    } finally {
      setSending(false);
      finishWrite();
    }
  }

  return <Card className={styles.conversation} title="新会话">
    <div className={styles.messages} ref={visible}>
      {streamState && <article className={styles.message}>
        <div className={styles.messageMeta}>
          <strong>我</strong>
          <Tag color="success">已完成</Tag>
        </div>
        <div className={styles.messageText}>{form.getFieldValue('content')}</div>
      </article>}
      {streamState && <article className={styles.message}>
        <div className={styles.messageMeta}>
          <strong>AI 助手</strong>
          <Tag color="processing">回复中</Tag>
        </div>
        {streamState.tools.map(tool => <div className={styles.tool} key={tool.key}>
          <Typography.Text type="secondary">工具 {tool.name} {tool.status === 'start' ? '调用中…' : '已完成'}</Typography.Text>
        </div>)}
        {streamState.content
          ? <div className={styles.messageText}>{streamState.content}</div>
          : <Typography.Text type="secondary">正在生成…</Typography.Text>}
      </article>}
    </div>
    {error && <Alert className={styles.feedback} type="error" showIcon title={error} />}
    {conversationId && <Alert className={styles.feedback} type="success" showIcon title="会话已创建" description="正在切换到会话视图..." />}
    {canSend ? <Form<ComposerValues> form={form} className={styles.composer} onFinish={handleSend} initialValues={{ content: '' }}>
      <Form.Item name="content" rules={[{ validator: (_, value: string) => {
        const error = validateMessageContent(value ?? '');
        return error ? Promise.reject(new Error(error)) : Promise.resolve();
      } }]}>
        <Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} placeholder="输入第一条消息，发送后自动创建会话..." aria-label="消息内容"
          onPressEnter={event => {
            if (event.shiftKey || event.nativeEvent.isComposing) return;
            event.preventDefault();
            if (!writing && !sending) form.submit();
          }} />
      </Form.Item>
      <div className={styles.composerActions}>
        <Typography.Text type="secondary" className={styles.recordNote}>{new TextEncoder().encode(draft).length} / 16000 UTF-8 字节</Typography.Text>
        <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={sending}
          disabled={writing || !draft.trim()}>发送</Button>
      </div>
    </Form> : <Typography.Text type="secondary">当前账号没有发送消息权限。</Typography.Text>}
  </Card>;
}
