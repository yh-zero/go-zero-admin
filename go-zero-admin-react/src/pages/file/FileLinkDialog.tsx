import { useQuery } from '@tanstack/react-query';
import { Button, Descriptions, Modal, Typography } from 'antd';
import { fetchFileURL } from '../../api/file';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
function safeURL(value?: string) {
  try { const url = new URL(value ?? ''); return ['http:', 'https:'].includes(url.protocol) ? url.href : undefined; } catch { return undefined; }
}
export function FileLinkDialog({ id, visibility, onClose }: { id: number; visibility: string; onClose: () => void }) {
  const query = useQuery({ queryKey: ['file-url', id], queryFn: () => fetchFileURL(id), staleTime: 0, gcTime: 0 });
  const data = query.isSuccess ? query.data : undefined;
  const href = safeURL(data?.url);
  const isPublic = visibility === 'public';
  return <Modal open title="文件访问链接" onCancel={onClose} footer={<Button onClick={onClose}>关闭</Button>}>
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {data && <><Descriptions column={1} items={[{ key: 'expires', label: '有效期', children: isPublic ? '长期有效' : data.expires_at ? formatTime(data.expires_at) : '未提供' }]} />
      {href ? <Typography.Link href={href} target="_blank" rel="noopener noreferrer">打开文件</Typography.Link> : <Typography.Text type="danger">未取得可用的文件链接</Typography.Text>}
      <Typography.Paragraph type="secondary">{isPublic ? '公开文件可通过此链接访问。' : '私有文件链接含临时访问权限，请妥善保管。'}</Typography.Paragraph></>}
  </Modal>;
}
