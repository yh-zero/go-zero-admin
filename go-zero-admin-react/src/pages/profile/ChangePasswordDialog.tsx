import { App, Form, Input, Modal } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { changePassword } from '../../api/profile';
import type { ChangePasswordReq } from '../../api/types';
import { clearSession } from '../../utils/session';
export function ChangePasswordDialog({ onClose }: { onClose: () => void }) {
  const [form] = Form.useForm<ChangePasswordReq & { confirm_password: string }>();
  const { message } = App.useApp();
  const navigate = useNavigate();
  const mutation = useMutation({ mutationFn: changePassword, onSuccess: () => {
    void message.success('密码已修改，请重新登录'); clearSession(); navigate('/login', { replace: true });
  } });
  return <Modal open title="修改密码" onCancel={onClose} onOk={() => form.submit()} confirmLoading={mutation.isPending} destroyOnHidden>
    <Form name="profile-password" form={form} layout="vertical" onFinish={({ old_password, new_password }) => mutation.mutate({ old_password, new_password })}>
      <Form.Item name="old_password" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}><Input.Password autoComplete="current-password" /></Form.Item>
      <Form.Item name="new_password" label="新密码" rules={[{ required: true, min: 8, message: '请输入至少8位的新密码' }, { validator: (_, value: string) => !value || new TextEncoder().encode(value).length <= 72 ? Promise.resolve() : Promise.reject(new Error('密码不能超过72字节')) }]}><Input.Password autoComplete="new-password" /></Form.Item>
      <Form.Item name="confirm_password" label="确认新密码" dependencies={['new_password']} rules={[{ required: true, message: '请再次输入新密码' }, ({ getFieldValue }) => ({ validator: (_, value: string) => !value || getFieldValue('new_password') === value ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致')) })]}><Input.Password autoComplete="new-password" /></Form.Item>
    </Form>
  </Modal>;
}
