import { useEffect } from 'react';
import { Button, Card, Form, Input, Typography } from 'antd';
import { LockOutlined, UserOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Navigate, useLocation, useNavigate } from 'react-router-dom';
import { fetchCaptcha, login } from '../../api/auth';
import type { LoginReq } from '../../api/types';
import { useAuthStore } from '../../stores/auth';
import { queryClient } from '../../utils/queryClient';
import { CaptchaField } from './CaptchaField';

export default function LoginPage() {
  const [form] = Form.useForm<LoginReq>();
  const navigate = useNavigate();
  const location = useLocation();
  const auth = useAuthStore();
  const captcha = useQuery({ queryKey: ['captcha'], queryFn: fetchCaptcha, staleTime: 0, gcTime: 0, refetchInterval: 90_000 });
  useEffect(() => { form.setFieldValue('captcha', ''); }, [captcha.data?.captcha_id, form]);
  const mutation = useMutation({
    mutationFn: login,
    onSuccess: (data) => {
      queryClient.clear();
      auth.setSession(data.access_token, data.expires_at);
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from?.startsWith('/') && !from.startsWith('//') && !from.startsWith('/login') ? from : '/dashboard', { replace: true });
    },
    onError: () => { form.setFieldValue('captcha', ''); void captcha.refetch(); },
  });
  if (auth.token) return <Navigate to="/dashboard" replace />;
  const submit = (values: LoginReq) => {
    if (!captcha.data || captcha.data.expires_at * 1000 <= Date.now()) {
      form.setFields([{ name: 'captcha', errors: ['验证码已过期，请刷新后重试'] }]); void captcha.refetch(); return;
    }
    mutation.mutate({ ...values, captcha_id: captcha.data.captcha_id });
  };
  return <main className="login-page"><Card className="login-card">
    <div className="login-heading"><SafetyCertificateOutlined className="login-symbol" /><Typography.Title level={2}>go-zero-admin</Typography.Title><Typography.Text type="secondary">登录管理工作台</Typography.Text></div>
    <Form name="login" form={form} layout="vertical" onFinish={submit} size="large" autoComplete="on">
      <Form.Item name="username" label="用户名" rules={[{ required: true, whitespace: true, message: '请输入用户名' }]}><Input prefix={<UserOutlined />} autoComplete="username" placeholder="请输入用户名" maxLength={64} /></Form.Item>
      <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}><Input.Password prefix={<LockOutlined />} autoComplete="current-password" placeholder="请输入密码" /></Form.Item>
      <Form.Item label="验证码" required><div className="captcha-row"><Form.Item name="captcha" noStyle rules={[{ required: true, message: '请输入验证码' }]}><Input autoComplete="off" placeholder="请输入验证码" maxLength={8} /></Form.Item><CaptchaField data={captcha.data} loading={captcha.isFetching} onReload={() => void captcha.refetch()} /></div></Form.Item>
      <Button block type="primary" htmlType="submit" loading={mutation.isPending} disabled={!captcha.data || captcha.isFetching}>登录</Button>
    </Form>
  </Card></main>;
}
