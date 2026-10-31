import { useRef } from 'react';
import { App, Form, Input, Modal } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { changePassword } from '../../api/profile';
import type { ChangePasswordReq } from '../../api/types';
import { clearSession } from '../../utils/session';
import { createPasswordConfirmationRules, newPasswordRules } from '../../utils/passwordRules';
export function ChangePasswordDialog({ onClose }: { onClose: () => void }) {
  const [form] = Form.useForm<ChangePasswordReq & { confirm_password: string }>();
  const submitting = useRef(false);
  const { message } = App.useApp();
  const navigate = useNavigate();
  const mutation = useMutation({ mutationFn: changePassword, onSuccess: () => {
    void message.success('密码已修改，请重新登录'); clearSession(); navigate('/login', { replace: true });
  }, onSettled: () => { submitting.current = false; } });
  return <Modal open title="修改密码" onCancel={() => { if (!submitting.current) onClose(); }}
    onOk={() => form.submit()} confirmLoading={mutation.isPending} destroyOnHidden
    cancelButtonProps={{ disabled: mutation.isPending }} closable={!mutation.isPending}
    mask={{ closable: !mutation.isPending }} keyboard={!mutation.isPending}>
    <Form name="profile-password" form={form} layout="vertical" disabled={mutation.isPending}
      onFinish={({ old_password, new_password }) => {
        if (submitting.current) return;
        submitting.current = true;
        mutation.mutate({ old_password, new_password });
      }}>
      <Form.Item name="old_password" label="当前密码" rules={[{ required: true, message: '请输入当前密码' }]}><Input.Password autoComplete="current-password" /></Form.Item>
      <Form.Item name="new_password" label="新密码" rules={newPasswordRules}><Input.Password autoComplete="new-password" /></Form.Item>
      <Form.Item name="confirm_password" label="确认新密码" dependencies={['new_password']} rules={createPasswordConfirmationRules('new_password')}><Input.Password autoComplete="new-password" /></Form.Item>
    </Form>
  </Modal>;
}
