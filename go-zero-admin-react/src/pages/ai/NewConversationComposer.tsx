import { useRef, useState } from 'react';
import { Alert, Button, Card, Form, Input, Space, Tag, Typography } from 'antd';
import { DownOutlined, SendOutlined } from '@ant-design/icons';
import type { AiSendMessageResult } from '../../api/types';
import type { AiChatScope } from './chatQueries';
import { validateMessageContent } from './chatState';
import { useMessageSend } from './useMessageSend';
import { useChatScroll } from './useChatScroll';
import type { ChatScrollPosition } from './useChatScroll';
import styles from './chat.module.css';

type ComposerValues = { content: string };
interface Props {
  scope: AiChatScope; canSend: boolean; writing: boolean; isCurrent: () => boolean;
  scrollPosition: { current: ChatScrollPosition | null };
  beginWrite: (permission: string) => boolean; finishWrite: () => void;
  onCompleted: (result: AiSendMessageResult) => void;
}

// 第一条消息由流式发送自动创建会话，成功结果直接填充正式会话缓存。
export function NewConversationComposer({ scope, canSend, writing, scrollPosition, isCurrent, beginWrite, finishWrite, onCompleted }: Props) {
  const [form] = Form.useForm<ComposerValues>();
  const draft = Form.useWatch('content', form) ?? '';
  const revision = useRef(0);
  const [retainedResult, setRetainedResult] = useState<AiSendMessageResult | null>(null);
  const { start, recheck, feedback, streamState, sending } = useMessageSend({ conversationId: 0, scope, isCurrent, beginWrite, finishWrite,
    onCompleted: (result, request) => {
      const currentDraft = form.getFieldValue('content') as string | undefined;
      if (result.conversation && currentDraft?.trim() && currentDraft !== request.content) {
        setRetainedResult(result);
      } else {
        setRetainedResult(null);
        onCompleted(result);
      }
    } });
  const scroll = useChatScroll({ sending, position: scrollPosition });
  function handleSend(values: ComposerValues) {
    if (validateMessageContent(values.content)) return;
    const submittedRevision = revision.current;
    start(values.content, request => {
      if (revision.current === submittedRevision && form.getFieldValue('content') === request.content) form.setFieldValue('content', '');
    });
  }
  return <Card className={styles.conversation} title="新会话">
    {retainedResult && <Alert className={styles.feedback} type="success" showIcon title="会话已创建，新草稿已保留"
      description="继续发送将使用已创建的会话。" />}
    <div className={styles.messagesArea}>
    <div className={styles.messages} ref={scroll.viewportRef} onScroll={scroll.onScroll}
      onWheel={scroll.onWheel} onKeyDown={scroll.onKeyDown} onTouchStart={scroll.onTouchStart} onTouchMove={scroll.onTouchMove}
      role="log" aria-label="聊天消息" aria-live="polite" tabIndex={0}>
      <div className={styles.messageList} ref={scroll.contentRef}>
      {streamState && <article className={styles.message}>
        <div className={styles.messageMeta}><strong>我</strong><Tag color="processing">发送中</Tag></div>
        <div className={styles.messageText}>{feedback?.request.content}</div>
      </article>}
      {streamState && <article className={styles.message}>
        <div className={styles.messageMeta}><strong>AI 助手</strong><Tag color="processing">回复中</Tag></div>
        {streamState.tools.map(tool => <div className={styles.tool} key={tool.key}>
          <Typography.Text type="secondary">工具 {tool.name} {tool.status === 'start' ? '调用中…' : '已完成'}</Typography.Text>
        </div>)}
        {streamState.content ? <div className={styles.messageText}>{streamState.content}</div> : <Typography.Text type="secondary">正在生成…</Typography.Text>}
      </article>}
      </div>
    </div>
    {scroll.showLatest && <Button className={styles.jumpToLatest} size="small" icon={<DownOutlined />} onClick={scroll.scrollToLatest} aria-label="回到最新消息">回到最新</Button>}
    </div>
    {feedback?.state === 'unknown' && <Alert className={styles.feedback} type="warning" showIcon title={feedback.error}
      description={<Space direction="vertical">
        <span>本次请求未确认成功，草稿已保留。重查复用原请求 ID；此前未送达时可能首次开始处理。</span>
        <span>重新发送会创建新请求，可能再次计费。</span>
        {canSend && <Button size="small" disabled={writing} onClick={recheck}>重查本次结果</Button>}
      </Space>} />}
    {canSend ? <Form<ComposerValues> form={form} className={styles.composer} onFinish={handleSend} initialValues={{ content: '' }}>
      <Form.Item name="content" rules={[{ validator: (_, value: string) => {
        const error = validateMessageContent(value ?? '');
        return error ? Promise.reject(new Error(error)) : Promise.resolve();
      } }]}>
        <Input.TextArea autoSize={{ minRows: 3, maxRows: 8 }} placeholder="输入第一条消息，发送后自动创建会话..." aria-label="消息内容"
          onChange={() => { revision.current += 1; }}
          onPressEnter={event => {
            if (event.shiftKey || event.nativeEvent.isComposing) return;
            event.preventDefault();
            if (!writing && !sending) form.submit();
          }} />
      </Form.Item>
      <div className={styles.composerActions}>
        <Typography.Text type="secondary" className={styles.recordNote}>{new TextEncoder().encode(draft).length} / 16000 UTF-8 字节</Typography.Text>
        <Button type="primary" htmlType="submit" icon={<SendOutlined />} loading={sending} disabled={writing || !draft.trim()}>
          {feedback?.state === 'unknown' ? '重新发送（新请求）' : '发送'}
        </Button>
      </div>
    </Form> : <Typography.Text type="secondary">当前账号没有发送消息权限。</Typography.Text>}
  </Card>;
}
