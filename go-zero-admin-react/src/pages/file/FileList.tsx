import { useState } from 'react';
import { App, Button, Card, Form, InputNumber, Popconfirm, Select, Space, Table, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteFile, fetchFiles } from '../../api/file';
import type { FileInfo, GetFileListReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { formatBytes, formatTime } from '../../utils/format';
import { FileLinkDialog } from './FileLinkDialog';
export default function FileList() {
  const [form] = Form.useForm<GetFileListReq>();
  const [params, setParams] = useState<GetFileListReq>({ page: 1, page_size: 20 });
  const [linkId, setLinkId] = useState<number | null>(null);
  const client = useQueryClient();
  const { message } = App.useApp();
  const query = usePagedQuery({ params, setParams, queryKey: ['files', params], queryFn: () => fetchFiles(params) });
  const remove = useMutation({ mutationFn: deleteFile, onSuccess: async () => { await client.invalidateQueries({ queryKey: ['files'] }); void message.success('文件已删除'); } });
  return <div className="page-stack"><PageHeader title="文件管理" description="查询文件资源、获取访问链接和管理引用" />
    <Card><Form name="file-filter" form={form} layout="inline" onFinish={(values) => setParams({ ...values, page: 1, page_size: params.page_size })}>
      <Form.Item name="visibility" label="可见性"><Select allowClear className="filter-select" placeholder="全部" options={[{ value: 'public', label: '公开' }, { value: 'private', label: '私有' }]} /></Form.Item>
      <Form.Item name="uploader_id" label="上传者 ID"><InputNumber min={1} precision={0} /></Form.Item>
      <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { form.resetFields(); setParams({ page: 1, page_size: params.page_size }); }}>重置</Button><Button aria-label="刷新文件列表" icon={<ReloadOutlined />} onClick={() => void query.refetch()} /></Space></Form.Item>
    </Form></Card>
    <Card><QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      <Table<FileInfo> rowKey="id" dataSource={query.data?.list ?? []} loading={query.isFetching} scroll={{ x: 1000 }} pagination={{ current: params.page, pageSize: params.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, page_size) => setParams({ ...params, page, page_size }) }} columns={[
        { title: '文件名称', dataIndex: 'original_name', ellipsis: true },
        { title: '类型', dataIndex: 'mime' }, { title: '大小', dataIndex: 'size', render: formatBytes },
        { title: '可见性', dataIndex: 'visibility', render: (value: string) => <Tag color={value === 'private' ? 'gold' : 'blue'}>{value === 'private' ? '私有' : '公开'}</Tag> },
        { title: '引用数', dataIndex: 'ref_count' }, { title: '上传时间', dataIndex: 'created_at', render: formatTime },
        { title: '操作', key: 'actions', render: (_, row) => <Space><Button type="link" onClick={() => setLinkId(row.id)}>访问链接</Button>
          <Auth code="sys:file:delete"><Popconfirm title="确认删除文件？" description="文件内容将被删除，无法撤销。" disabled={row.ref_count > 0} onConfirm={() => remove.mutateAsync(row.id).catch(() => undefined)} okButtonProps={{ loading: remove.isPending }}>
            <Button type="link" danger disabled={row.ref_count > 0 || remove.isPending} title={row.ref_count > 0 ? '文件仍被引用，不能删除' : undefined}>删除</Button>
          </Popconfirm></Auth></Space> },
      ]} />
    </Card>{linkId !== null && <FileLinkDialog key={linkId} id={linkId} onClose={() => setLinkId(null)} />}
  </div>;
}
