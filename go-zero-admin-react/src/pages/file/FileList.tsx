import { useState } from 'react';
import { App, Button, Card, Col, DatePicker, Form, Input, InputNumber, Popconfirm, Row, Select, Space, Statistic, Table, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Dayjs } from 'dayjs';
import { deleteFile, fetchFiles, fetchFileStats } from '../../api/file';
import type { FileInfo, GetFileListReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { QueryState } from '../../components/QueryState';
import { AdvancedFilterToggle } from '../../components/AdvancedFilterToggle';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { formatBytes, formatTime } from '../../utils/format';
import { FileLinkDialog } from './FileLinkDialog';
import { FileDetailDialog } from './FileDetailDialog';
import { FileUploadButton } from '../../components/FileUploadButton';
import { queryKeys } from '../../utils/queryKeys';
import { useTablePagination } from '../../hooks/useTablePagination';

type FileForm = GetFileListReq & { start_time?: Dayjs; end_time?: Dayjs };
const seconds = (value?: Dayjs) => value?.unix();

export default function FileList() {
  const [form] = Form.useForm<FileForm>();
  const [uploadVisibility, setUploadVisibility] = useState<'public' | 'private'>('private');
  const [params, setParams] = useState<GetFileListReq>({ page: 1, page_size: 20 });
  const [linkFile, setLinkFile] = useState<Pick<FileInfo, 'id' | 'visibility'> | null>(null);
  const [detailId, setDetailId] = useState<number | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const client = useQueryClient();
  const { message } = App.useApp();
  const listParams = { ...params, include_cleanup_pending: true };
  const query = usePagedQuery({ params, setParams, queryKey: [queryKeys.files, listParams], queryFn: () => fetchFiles(listParams) });
  const stats = useQuery({ queryKey: [queryKeys.fileStats], queryFn: fetchFileStats });
  const pagination = useTablePagination(params, setParams, query.data?.total ?? 0);
  const remove = useMutation({ mutationFn: deleteFile,
    onSuccess: () => { void message.success('文件已删除'); },
    onSettled: async () => Promise.all([
      client.invalidateQueries({ queryKey: [queryKeys.files] }),
      client.invalidateQueries({ queryKey: [queryKeys.fileStats] }),
    ]),
  });
  const submit = (values: FileForm) => setParams({ ...values, start_time: seconds(values.start_time), end_time: seconds(values.end_time), page: 1, page_size: params.page_size });
  const advancedCount = [params.ext, params.uploader_id, params.min_size, params.max_size, params.start_time, params.end_time].filter(value => value !== undefined && value !== '').length;
  const cards = [
    { title: '有效文件', value: stats.data?.total_files, bytes: stats.data?.total_bytes },
    { title: '公开文件', value: stats.data?.public_files, bytes: stats.data?.public_bytes },
    { title: '私有文件', value: stats.data?.private_files, bytes: stats.data?.private_bytes },
    { title: '待清理', value: stats.data?.cleanup_pending_files, bytes: stats.data?.cleanup_pending_bytes },
  ];
  return <div className="page-stack">
    <Row gutter={[16, 16]}>{cards.map(card => <Col xs={24} sm={12} xl={6} key={card.title}><Card loading={stats.isPending}>
      <Statistic title={card.title} value={card.value ?? '—'} /><span className="table-secondary">{card.bytes === undefined ? '暂不可用' : formatBytes(card.bytes)}</span>
    </Card></Col>)}</Row>
    <Card className="filter-card"><Form name="file-filter" className={`filter-form${showAdvanced ? ' is-expanded' : ''}`} form={form} layout="inline" onFinish={submit}>
      <Form.Item name="name" label="文件名"><Input allowClear maxLength={255} placeholder="模糊匹配文件名" /></Form.Item>
      <Form.Item name="mime" label="类型"><Select allowClear className="filter-select" options={['text/plain', 'text/csv', 'application/pdf', 'image/png', 'image/jpeg', 'image/gif'].map(value => ({ value, label: value }))} /></Form.Item>
      <Form.Item name="visibility" label="可见性"><Select allowClear className="filter-select" placeholder="全部" options={[{ value: 'public', label: '公开' }, { value: 'private', label: '私有' }]} /></Form.Item>
      <Form.Item name="cleanup_pending_only" label="清理状态"><Select allowClear className="filter-select" placeholder="全部文件" options={[{ value: true, label: '仅待清理' }]} /></Form.Item>
      {showAdvanced && <>
        <Form.Item name="ext" label="扩展名"><Input allowClear maxLength={16} placeholder="如 .pdf" /></Form.Item>
        <Form.Item name="uploader_id" label="上传者 ID"><InputNumber min={1} precision={0} /></Form.Item>
        <Form.Item name="min_size" label="最小大小"><InputNumber min={0} precision={0} suffix="B" /></Form.Item>
        <Form.Item name="max_size" label="最大大小"><InputNumber min={0} precision={0} suffix="B" /></Form.Item>
        <Form.Item name="start_time" label="开始时间"><DatePicker showTime /></Form.Item>
        <Form.Item name="end_time" label="结束时间"><DatePicker showTime /></Form.Item>
      </>}
      <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { form.resetFields(); setParams({ page: 1, page_size: params.page_size }); setShowAdvanced(false); }}>重置</Button><Button aria-label="刷新文件列表" icon={<ReloadOutlined />} onClick={() => void query.refetch()} />
        <AdvancedFilterToggle open={showAdvanced} activeCount={advancedCount} onChange={setShowAdvanced} /></Space></Form.Item>
    </Form></Card>
    <Card className="data-card" title="文件列表" extra={<Space wrap>
      <Select aria-label="上传文件可见性" value={uploadVisibility} onChange={setUploadVisibility}
        options={[{ value: 'private', label: '私有文件' }, { value: 'public', label: '公开文件' }]} />
      <FileUploadButton visibility={uploadVisibility} />
    </Space>}><QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      <Table<FileInfo> rowKey="id" dataSource={query.data?.list ?? []} loading={query.isFetching} scroll={{ x: 1200 }} pagination={pagination} columns={[
        { title: '文件名称', dataIndex: 'original_name', ellipsis: true, render: (value: string, row) => <Space><span className="table-primary">{value}</span>{row.cleanup_pending && <Tag color="orange">待确认清理</Tag>}</Space> },
        { title: '类型', dataIndex: 'mime' }, { title: '大小', dataIndex: 'size', render: formatBytes },
        { title: '上传者', dataIndex: 'uploader_name', render: (value: string) => value || '—' },
        { title: '可见性', dataIndex: 'visibility', render: (value: string) => <Tag color={value === 'private' ? 'gold' : 'blue'}>{value === 'private' ? '私有' : '公开'}</Tag> },
        { title: '引用数', dataIndex: 'ref_count' }, { title: '上传时间', dataIndex: 'created_at', render: formatTime },
        { title: '操作', key: 'actions', render: (_, row) => <Space>
          <Button type="link" onClick={() => setDetailId(row.id)}>详情</Button>
          {!row.cleanup_pending && <Button type="link" onClick={() => setLinkFile({ id: row.id, visibility: row.visibility })}>访问链接</Button>}
          <Auth code="sys:file:delete"><Popconfirm title={row.cleanup_pending ? '重试清理文件？' : '确认删除文件？'} description={row.cleanup_pending ? '将再次向存储服务确认删除结果。' : '文件内容将被删除，无法撤销。'} disabled={row.ref_count > 0} onConfirm={() => remove.mutateAsync(row.id).catch(() => undefined)} okButtonProps={{ loading: remove.isPending }}>
            <Button type="link" danger disabled={row.ref_count > 0 || remove.isPending} title={row.ref_count > 0 ? '文件仍被引用，不能删除' : undefined}>{row.cleanup_pending ? '重试清理' : '删除'}</Button>
          </Popconfirm></Auth></Space> },
      ]} />
    </Card>
    {linkFile !== null && <FileLinkDialog key={linkFile.id} id={linkFile.id} visibility={linkFile.visibility} onClose={() => setLinkFile(null)} />}
    {detailId !== null && <FileDetailDialog key={detailId} id={detailId} onClose={() => setDetailId(null)} />}
  </div>;
}
