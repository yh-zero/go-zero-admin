import type { ReactNode } from 'react';
import { Typography } from 'antd';
export function PageHeader({ title, description, extra }: { title: string; description?: string; extra?: ReactNode }) {
  return <div className="page-header"><div><Typography.Title level={3}>{title}</Typography.Title>{description && <Typography.Text type="secondary">{description}</Typography.Text>}</div>{extra && <div className="page-actions">{extra}</div>}</div>;
}
