import { Button, Spin } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { CaptchaResp } from '../../api/types';
import { captchaSource } from '../../utils/captcha';
export function CaptchaField({ data, loading, onReload }: { data?: CaptchaResp; loading: boolean; onReload: () => void }) {
  const source = captchaSource(data?.image_base64 ?? '');
  return <div className="captcha-display"><Spin spinning={loading}>
    {source.kind === 'image' ? <img src={source.value} alt="登录验证码" /> : source.kind === 'text' ? <span className="captcha-text" aria-label="登录验证码">{source.value}</span> : <span className="captcha-unavailable">验证码不可用</span>}
  </Spin><Button className="captcha-reload" type="text" icon={<ReloadOutlined />} aria-label="刷新验证码" title="刷新验证码" onClick={onReload} disabled={loading} /></div>;
}
