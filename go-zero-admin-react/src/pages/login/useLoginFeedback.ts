import { useEffect, useRef, useState, useSyncExternalStore } from 'react';

export type LoginResult = 'idle' | 'success' | 'failure';

const desktopQuery = '(min-width: 768px)';

function subscribeViewport(onChange: () => void) {
  const query = window.matchMedia(desktopQuery);
  query.addEventListener('change', onChange);
  return () => query.removeEventListener('change', onChange);
}

export function useLoginFeedback() {
  const enabled = useSyncExternalStore(subscribeViewport, () => window.matchMedia(desktopQuery).matches, () => false);
  const [ready, setReady] = useState(false);
  const [passwordFocused, setPasswordFocused] = useState(false);
  const [formFocused, setFormFocused] = useState(false);
  const [result, setResult] = useState<LoginResult>('idle');
  const mounted = useRef(false);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; clearTimeout(timer.current); };
  }, []);

  const showSuccess = (onComplete: () => void) => {
    if (!mounted.current) return;
    clearTimeout(timer.current);
    setResult('success');
    // 反馈只占一秒；资源加载失败或窄屏时直接完成登录。
    if (enabled && ready) timer.current = setTimeout(onComplete, 1100);
    else onComplete();
  };

  return {
    enabled, result, setReady, setPasswordFocused, formFocused, setFormFocused,
    coverEyes: passwordFocused && result === 'idle',
    clearFailure: () => { if (result === 'failure') setResult('idle'); },
    reset: () => { setResult('idle'); setPasswordFocused(false); },
    showFailure: () => { if (mounted.current) setResult('failure'); },
    showSuccess,
  };
}
