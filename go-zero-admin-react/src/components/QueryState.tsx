import { Button, Result, Skeleton } from 'antd';
export function QueryState({ isPending, error, onRetry }: { isPending?: boolean; error?: unknown; onRetry?: () => unknown }) {
  if (isPending) return <Skeleton active className="query-skeleton" />;
  if (!error) return null;
  return <Result status="warning" title="暂时无法加载数据" subTitle={error instanceof Error ? error.message : '请稍后重试'} extra={onRetry && <Button onClick={() => void onRetry()}>重新加载</Button>} />;
}
