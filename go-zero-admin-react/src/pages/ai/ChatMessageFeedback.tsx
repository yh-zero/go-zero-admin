import { useRef, useState } from 'react';
import { Button, Form, Input, Modal, Select, Space, Typography } from 'antd';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { setAiMessageFeedback } from '../../api/ai';
import type { AiChatMessage, AiMessageFeedbackReq } from '../../api/types';
import { aiChatKeys } from './chatQueries';
import type { AiChatScope } from './chatQueries';

export interface ChatFeedbackContext { conversationId: number; scope: AiChatScope; isCurrent: () => boolean }
const labels = { helpful: '有帮助', irrelevant_source: '来源不相关', missing_knowledge: '资料缺失' };
export function ChatMessageFeedback({ message, context }: { message: AiChatMessage; context: ChatFeedbackContext }) {
  const [open, setOpen] = useState(false);
  return <>
    <Space size="small"><Button size="small" type="text" onClick={() => setOpen(true)}>评价回答</Button>{message.feedback && <Typography.Text type="secondary">{labels[message.feedback]}</Typography.Text>}</Space>
    {open && <MessageFeedbackDialog message={message} context={context} onClose={() => setOpen(false)} />}
  </>;
}

// 每次打开创建独立 Form 实例；StrictMode 模拟卸载不能清掉仍挂载表单的初值。
function MessageFeedbackDialog({ message, context, onClose }: { message: AiChatMessage; context: ChatFeedbackContext; onClose: () => void }) {
  const [form] = Form.useForm<AiMessageFeedbackReq>();
  const client = useQueryClient();
  const busy = useRef(false);
  const save = useMutation({ mutationFn: (data: AiMessageFeedbackReq) => setAiMessageFeedback(context.conversationId, message.id, data), retry: false });
  async function submit(values: AiMessageFeedbackReq) {
    if (busy.current || !context.isCurrent()) return;
    busy.current = true;
    try {
      await save.mutateAsync(values);
      if (!context.isCurrent()) return;
      onClose();
      await client.invalidateQueries({ queryKey: aiChatKeys.messages(context.scope, context.conversationId), exact: true });
    } catch { /* 失败时保留反馈草稿，由统一请求层显示错误。 */ }
    finally { busy.current = false; }
  }
  return <Modal open title="评价这条回答" confirmLoading={save.isPending} mask={{ closable: !save.isPending }} closable={!save.isPending}
      onCancel={() => { if (!save.isPending) onClose(); }} onOk={() => form.submit()}>
      <Form<AiMessageFeedbackReq> form={form} initialValues={{ feedback: message.feedback || 'helpful', note: message.feedback_note || '' }} onFinish={submit} layout="vertical">
        <Form.Item label="反馈" name="feedback" rules={[{ validator: (_, value: unknown) => typeof value === 'string' && ['', ...Object.keys(labels)].includes(value) ? Promise.resolve() : Promise.reject(new Error('请选择反馈类型')) }]}><Select options={[...Object.entries(labels).map(([value, label]) => ({ value, label })), { value: '', label: '清除评价' }]} /></Form.Item>
        <Form.Item label="补充说明（可选）" name="note" rules={[{ validator: (_, value: string) => new TextEncoder().encode(value ?? '').length <= 1000 ? Promise.resolve() : Promise.reject(new Error('说明最多 1000 UTF-8 字节')) }]}><Input.TextArea rows={3} placeholder="例如：引用文档不是当前版本，或缺少具体操作步骤" /></Form.Item>
      </Form>
    </Modal>;
}
