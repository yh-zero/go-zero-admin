import { Tag } from 'antd';
export function StatusTag({ status }: { status: number }) {
  return <Tag className="status-tag" color={status === 1 ? 'success' : 'default'}><span className="status-tag-dot" aria-hidden="true" />{status === 1 ? '启用' : status === 2 ? '停用' : '未知'}</Tag>;
}
