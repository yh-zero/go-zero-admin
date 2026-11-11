import { useEffect, useState } from 'react';
import type { ComponentType } from 'react';
import { Alert, Button, Skeleton } from 'antd';

interface Props<P extends object> { load: () => Promise<ComponentType<P>>; componentProps: P }

export function AsyncContent<P extends object>({ load, componentProps }: Props<P>) {
  const [Content, setContent] = useState<ComponentType<P> | null>(null);
  const [error, setError] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    void load().then(component => { if (active) setContent(() => component); }, () => { if (active) setError(true); });
    return () => { active = false; };
  }, [load, attempt]);
  if (Content) return <Content {...componentProps} />;
  if (error) return <Alert type="warning" showIcon title="面板加载失败，当前页面和草稿已保留" action={<Button onClick={() => { setError(false); setAttempt(value => value + 1); }}>重新加载面板</Button>} />;
  return <div role="status" aria-label="正在加载面板"><Skeleton active /></div>;
}
