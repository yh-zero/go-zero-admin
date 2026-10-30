import { Tag } from 'antd';
export function StatusTag({ status }: { status: number }) { return <Tag color={status === 1 ? 'success' : 'default'}>{status === 1 ? '启用' : status === 2 ? '停用' : '未知'}</Tag>; }
