import { useEffect } from 'react';
import { Alert, Button, Form, Input, Typography } from 'antd';
import { ArrowRightOutlined, LockOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Navigate, useLocation, useNavigate } from 'react-router-dom';
import { fetchCaptcha, login } from '../../api/auth';
import type { LoginReq } from '../../api/types';
import { useAuthStore } from '../../stores/auth';
import { saveSession } from '../../utils/session';
import { BrandMark } from '../../components/BrandMark';
import { CaptchaField } from './CaptchaField';
import { LoginMascot } from './LoginMascot';
import { useLoginFeedback } from './useLoginFeedback';
import './login.css';

function LoginBrand({ feedback }: { feedback: ReturnType<typeof useLoginFeedback> }) {
  return <section className="login-brand-panel" aria-label="go-zero-admin 管理工作台">
    <div className="login-brand"><BrandMark /><span>go-zero-admin</span></div>
    <div className="login-brand-copy">
      <span className="login-eyebrow">管理工作台</span>
      <h1>统一管理，<br />专注每一天的工作。</h1>
      <p>连接用户、权限与业务，让日常管理井然有序。</p>
    </div>
    <LoginMascot enabled={feedback.enabled} coverEyes={feedback.coverEyes} result={feedback.result} lookAtForm={feedback.formFocused} onReady={feedback.setReady} />
    <div className="login-brand-caption"><span />简洁 · 有序 · 高效</div>
  </section>;
}

export default function LoginPage() {
  const [form] = Form.useForm<LoginReq>();
  const navigate = useNavigate();
  const location = useLocation();
  const auth = useAuthStore();
  const feedback = useLoginFeedback();
  const captcha = useQuery({ queryKey: ['captcha'], queryFn: fetchCaptcha, staleTime: 0, gcTime: 0, refetchInterval: 90_000 });
  const captchaEnabled = captcha.data?.enabled !== false;
  useEffect(() => { form.setFieldValue('captcha', ''); }, [captcha.data?.captcha_id, captcha.data?.enabled, form]);
  const mutation = useMutation({
    mutationFn: login,
    onSuccess: (data) => {
      feedback.showSuccess(() => {
        void saveSession(data).then(() => {
          const from = (location.state as { from?: string } | null)?.from;
          navigate(from?.startsWith('/') && !from.startsWith('//') && !from.startsWith('/login') ? from : '/dashboard', { replace: true });
        }).catch(() => feedback.showFailure());
      });
    },
    onError: () => { feedback.showFailure(); form.setFieldValue('captcha', ''); void captcha.refetch(); },
  });
  if (auth.token) return <Navigate to="/dashboard" replace />;
  const submit = (values: LoginReq) => {
    if (mutation.isPending || mutation.isSuccess) return;
    if (!captcha.isSuccess || !captcha.data || captcha.isFetching) { void captcha.refetch(); return; }
    if (captchaEnabled && captcha.data.expires_at * 1000 <= Date.now()) {
      form.setFields([{ name: 'captcha', errors: ['验证码已过期，请刷新后重试'] }]); feedback.showFailure(); void captcha.refetch(); return;
    }
    feedback.reset();
    mutation.mutate(captchaEnabled ? { ...values, captcha_id: captcha.data.captcha_id } : { username: values.username, password: values.password });
  };
  return <main className="login-page">
    <div className="login-shell">
      <LoginBrand feedback={feedback} />
      <section className="login-form-panel" aria-labelledby="login-title">
        <div className="login-form-content">
          <div className="login-heading">
            <span className="login-form-eyebrow">账号登录</span>
            <Typography.Title level={2} id="login-title">欢迎回来</Typography.Title>
            <Typography.Paragraph>登录账号，开始今天的工作。</Typography.Paragraph>
          </div>
          <Form name="login" form={form} layout="vertical" onFinish={submit} onFinishFailed={feedback.showFailure}
            onFocusCapture={() => { feedback.clearFailure(); feedback.setFormFocused(true); }}
            onBlurCapture={event => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) feedback.setFormFocused(false); }}
            onValuesChange={feedback.clearFailure} disabled={mutation.isPending || mutation.isSuccess} size="large" autoComplete="on">
            <Form.Item name="username" label="用户名" rules={[{ required: true, whitespace: true, message: '请输入用户名' }]}>
              <Input prefix={<UserOutlined />} autoComplete="username" placeholder="请输入用户名" maxLength={64} />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password prefix={<LockOutlined />} autoComplete="current-password" placeholder="请输入密码" onFocus={() => feedback.setPasswordFocused(true)} onBlur={() => feedback.setPasswordFocused(false)} />
            </Form.Item>
            {captchaEnabled && <Form.Item label="验证码" required>
              <div className="captcha-row">
                <Form.Item name="captcha" noStyle preserve={false} rules={[{ required: true, message: '请输入验证码' }]}>
                  <Input autoComplete="off" placeholder="请输入验证码" maxLength={8} />
                </Form.Item>
                <CaptchaField data={captcha.data} loading={captcha.isFetching} onReload={() => void captcha.refetch()} />
              </div>
            </Form.Item>}
            {captcha.isError && <Form.Item><Alert type="error" title="登录配置加载失败" action={<Button size="small" onClick={() => void captcha.refetch()}>重试</Button>} /></Form.Item>}
            <Button className="login-submit" block type="primary" htmlType="submit" loading={mutation.isPending || mutation.isSuccess} disabled={!captcha.isSuccess || captcha.isFetching || mutation.isSuccess}>
              {mutation.isSuccess ? '登录成功，正在进入' : '登录工作台'}<ArrowRightOutlined />
            </Button>
          </Form>
          <div className="login-form-note"><LockOutlined /><span>使用您的管理账号登录</span></div>
        </div>
        <div className="login-footer">go-zero-admin · 管理工作台</div>
      </section>
    </div>
  </main>;
}
