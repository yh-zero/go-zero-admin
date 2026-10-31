import { useEffect, useState } from 'react';
import type { ComponentType } from 'react';
import type { LoginResult } from './useLoginFeedback';

export interface LoginCatProps {
  coverEyes: boolean;
  result: LoginResult;
  lookAtForm?: boolean;
  onReady: (ready: boolean) => void;
}

interface LoginMascotProps extends LoginCatProps { enabled: boolean }

export function LoginMascot({ enabled, ...props }: LoginMascotProps) {
  const [Animation, setAnimation] = useState<ComponentType<LoginCatProps> | null>(null);
  useEffect(() => {
    if (!enabled) return;
    let active = true;
    void import('./RiveLoginCat').then(module => {
      if (active) setAnimation(() => module.default);
    }).catch(() => { /* 动画加载失败时保留静态猫咪，登录表单照常使用。 */ });
    return () => { active = false; };
  }, [enabled]);

  return <figure className="login-mascot">
    <div className="login-mascot-message" aria-live="polite">{props.result === 'success' ? '登录成功，一起开工吧' : props.result === 'failure' ? '没关系，再试一次吧' : props.coverEyes ? '你放心输入，我不偷看' : props.lookAtForm ? '我在这里等你' : '嗨，欢迎回来'}</div>
    <div className="login-mascot-stage" aria-hidden="true">
      {enabled && Animation ? <Animation {...props} /> : <img className="login-mascot-fallback" src={`${import.meta.env.BASE_URL}mascots/login-cat.png`} alt="" />}
    </div>
  </figure>;
}
