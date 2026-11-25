import { Button, Descriptions, Modal, Table } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchFileDetail } from '../../api/file';
import type { FileReferenceInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatBytes, formatTime } from '../../utils/format';
import { queryKeys } from '../../utils/queryKeys';

const refNames: Record<string, string> = { admin_avatar: '管理员头像', app_avatar: 'APP用户头像' };

export function FileDetailDialog({ id, onClose }: { id: number; onClose: () => void }) {
  const query = useQuery({ queryKey: [queryKeys.fileDetail, id], queryFn: () => fetchFileDetail(id) });
  const file = query.data?.file;
  const references = query.data?.references ?? [];
  return <Modal open title="文件详情与引用" width={780} onCancel={onClose} footer={<Button onClick={onClose}>关闭</Button>}>
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
    {file && <>
      <Descriptions column={1} size="small" items={[
        { key: 'name', label: '文件名', children: file.original_name },
        { key: 'uploader', label: '上传者', children: file.uploader_name || file.uploader_id },
        { key: 'mime', label: '类型', children: file.mime },
        { key: 'size', label: '大小', children: formatBytes(file.size) },
        { key: 'visibility', label: '可见性', children: file.visibility === 'private' ? '私有' : '公开' },
        { key: 'key', label: '对象键', children: <span className="code-text">{file.object_key}</span> },
        { key: 'sha', label: 'SHA-256', children: <span className="code-text">{file.sha256 || '历史文件未记录'}</span> },
        { key: 'time', label: '上传时间', children: formatTime(file.created_at) },
      ]} />
      <Table<FileReferenceInfo> rowKey="id" size="small" style={{ marginTop: 18 }} dataSource={references} pagination={false}
        locale={{ emptyText: '暂无业务引用' }} columns={[
          { title: '引用类型', dataIndex: 'ref_type', render: (value: string) => refNames[value] ?? value },
          { title: '业务对象 ID', dataIndex: 'ref_id' },
          { title: '引用创建人', dataIndex: 'created_by' },
          { title: '引用时间', dataIndex: 'created_at', render: formatTime },
        ]} />
    </>}
  </Modal>;
}
