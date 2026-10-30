import { App, Form, Input, Modal } from 'antd';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { resetUserPassword } from '../../api/user';
import type { UserInfo } from '../../api/types';

interface Props { user: UserInfo; onClose: () => void }
interface PasswordForm { new_password: string; confirmation: string }

export function ResetPasswordDialog({ user, onClose }: Props) {
  const [form] = Form.useForm<PasswordForm>();
  const { message } = App.useApp();
  const client = useQueryClient();
  const reset = useMutation({
    mutationFn: (values: PasswordForm) => resetUserPassword(user.id, values.new_password),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['users'] });
      message.success('密码已重置');
      onClose();
    },
  });
  return <Modal open title={`重置密码 · ${user.username}`} onCancel={onClose} onOk={() => form.submit()}
    confirmLoading={reset.isPending} cancelButtonProps={{ disabled: reset.isPending }} closable={!reset.isPending} maskClosable={!reset.isPending}>
    <Form name="user-password-reset" form={form} layout="vertical" disabled={reset.isPending} onFinish={(values) => reset.mutate(values)}>
      <Form.Item name="new_password" label="新密码" rules={[
        { required: true }, { min: 8, message: '请至少输入 8 个字符' },
        { validator: (_, value: string) => !value || new TextEncoder().encode(value).length <= 72
          ? Promise.resolve() : Promise.reject(new Error('密码最多 72 字节')) },
      ]}><Input.Password autoComplete="new-password" /></Form.Item>
      <Form.Item name="confirmation" label="确认新密码" dependencies={['new_password']} rules={[
        { required: true }, ({ getFieldValue }) => ({ validator: (_, value: string) => !value || value === getFieldValue('new_password')
          ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致')) }),
      ]}><Input.Password autoComplete="new-password" /></Form.Item>
    </Form>
  </Modal>;
}
