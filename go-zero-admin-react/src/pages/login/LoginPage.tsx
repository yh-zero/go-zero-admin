import { useEffect } from 'react';
import { Button, Form, Input, Typography } from 'antd';
import { AppstoreOutlined, ArrowRightOutlined, LockOutlined, SafetyCertificateOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Navigate, useLocation, useNavigate } from 'react-router-dom';
import { fetchCaptcha, login } from '../../api/auth';
import type { LoginReq } from '../../api/types';
import { useAuthStore } from '../../stores/auth';
import { queryClient } from '../../utils/queryClient';
import { BrandMark } from '../../components/BrandMark';
import { CaptchaField } from './CaptchaField';
import './login.css';

function LoginBrand() {
  return <section className="login-brand-panel" aria-label="go-zero-admin 管理工作台">
    <div className="login-brand"><BrandMark /><span>go-zero-admin</span></div>
    <div className="login-brand-copy">
      <span className="login-eyebrow">管理工作台</span>
      <h1>统一管理，<br />专注每一天的工作。</h1>
      <p>连接用户、权限与业务，让日常管理井然有序。</p>
    </div>
    <div className="login-illustration" aria-hidden="true">
      <div className="login-orbit login-orbit-outer" />
      <div className="login-orbit login-orbit-inner" />
      <div className="login-illustration-core"><BrandMark /></div>
      <span className="login-illustration-node login-node-users"><TeamOutlined /></span>
      <span className="login-illustration-node login-node-security"><SafetyCertificateOutlined /></span>
      <span className="login-illustration-node login-node-apps"><AppstoreOutlined /></span>
    </div>
    <div className="login-brand-caption"><span />简洁 · 有序 · 高效</div>
  </section>;
}

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
  return <main className="login-page">
    <div className="login-shell">
      <LoginBrand />
      <section className="login-form-panel" aria-labelledby="login-title">
        <div className="login-form-content">
          <div className="login-heading">
            <span className="login-form-eyebrow">账号登录</span>
            <Typography.Title level={2} id="login-title">欢迎回来</Typography.Title>
            <Typography.Paragraph>登录账号，开始今天的工作。</Typography.Paragraph>
          </div>
          <Form name="login" form={form} layout="vertical" onFinish={submit} size="large" autoComplete="on">
            <Form.Item name="username" label="用户名" rules={[{ required: true, whitespace: true, message: '请输入用户名' }]}>
              <Input prefix={<UserOutlined />} autoComplete="username" placeholder="请输入用户名" maxLength={64} />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password prefix={<LockOutlined />} autoComplete="current-password" placeholder="请输入密码" />
            </Form.Item>
            <Form.Item label="验证码" required>
              <div className="captcha-row">
                <Form.Item name="captcha" noStyle rules={[{ required: true, message: '请输入验证码' }]}>
                  <Input autoComplete="off" placeholder="请输入验证码" maxLength={8} />
                </Form.Item>
                <CaptchaField data={captcha.data} loading={captcha.isFetching} onReload={() => void captcha.refetch()} />
              </div>
            </Form.Item>
            <Button className="login-submit" block type="primary" htmlType="submit" loading={mutation.isPending} disabled={!captcha.data || captcha.isFetching}>
              登录工作台<ArrowRightOutlined />
            </Button>
          </Form>
          <div className="login-form-note"><LockOutlined /><span>使用您的管理账号登录</span></div>
        </div>
        <div className="login-footer">go-zero-admin · 管理工作台</div>
      </section>
    </div>
  </main>;
}
