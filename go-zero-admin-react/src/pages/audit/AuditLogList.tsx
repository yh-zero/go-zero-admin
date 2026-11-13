import { useState } from 'react';
import { Button, Card, Descriptions, Form, Input, InputNumber, Modal, Select, Space, Table, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { fetchAuditLogs } from '../../api/audit';
import type { AuditLogInfo, GetAuditLogListReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
import { queryKeys } from '../../utils/queryKeys';
import { useTablePagination } from '../../hooks/useTablePagination';
export default function AuditLogList() {
  const [form] = Form.useForm<GetAuditLogListReq>();
  const [params, setParams] = useState<GetAuditLogListReq>({ page: 1, page_size: 20 });
  const [detail, setDetail] = useState<AuditLogInfo | null>(null);
  const query = useQuery({ queryKey: [queryKeys.auditLogs, params], queryFn: () => fetchAuditLogs(params) });
  const pagination = useTablePagination(params, setParams, query.data?.total ?? 0);

  return <div className="page-stack">
    <Card className="filter-card"><Form name="audit-filter" className="filter-form" form={form} layout="inline" onFinish={(values) => setParams({ ...values, page: 1, page_size: params.page_size })}>
      <Form.Item name="module" label="模块"><Input placeholder="如 sys/user" allowClear /></Form.Item>
      <Form.Item name="action" label="动作"><Input placeholder="如 create" allowClear /></Form.Item>
      <Form.Item name="user_id" label="用户 ID"><InputNumber min={1} precision={0} /></Form.Item>
      <Form.Item name="result" label="结果"><Select allowClear className="filter-select" placeholder="全部" options={[{ label: '成功', value: 1 }, { label: '失败', value: 2 }]} /></Form.Item>
      <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { form.resetFields(); setParams({ page: 1, page_size: params.page_size }); }}>重置</Button><Button aria-label="刷新审计日志" icon={<ReloadOutlined />} onClick={() => void query.refetch()} /></Space></Form.Item>
    </Form></Card>
    <Card className="data-card" title="操作记录"><QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      <Table<AuditLogInfo> rowKey="id" dataSource={query.data?.list ?? []} loading={query.isFetching} scroll={{ x: 1000 }} pagination={pagination} columns={[
        { title: '操作人', dataIndex: 'username', render: (value: string) => <span className="table-primary">{value || '—'}</span> },
        { title: '模块', dataIndex: 'module' }, { title: '动作', dataIndex: 'action' },
        { title: '方法', dataIndex: 'method' }, { title: '路径', dataIndex: 'path', ellipsis: true, render: (value: string) => <span className="code-text">{value}</span> },
        { title: '结果', dataIndex: 'result', render: (value: number) => <Tag color={value === 1 ? 'success' : 'error'}>{value === 1 ? '成功' : '失败'}</Tag> },
        { title: '耗时', dataIndex: 'duration_ms', render: (value: number) => value + ' ms' },
        { title: '时间', dataIndex: 'created_at', render: formatTime },
        { title: '操作', key: 'actions', render: (_, row) => <Button type="link" onClick={() => setDetail(row)}>详情</Button> },
      ]} />
    </Card>
    <Modal open={!!detail} title="审计详情" width={680} onCancel={() => setDetail(null)} footer={<Button onClick={() => setDetail(null)}>关闭</Button>}>
      {detail && <Descriptions column={1} items={[
        { key: 'user', label: '操作人', children: detail.username || '—' },
        { key: 'request', label: '请求', children: <span className="code-text">{detail.method + ' ' + detail.path}</span> },
        { key: 'ip', label: '来源 IP', children: detail.ip || '—' },
        { key: 'error', label: '错误信息', children: detail.error_msg || '—' },
        { key: 'time', label: '时间', children: formatTime(detail.created_at) },
      ]} />}
    </Modal>
  </div>;
}
