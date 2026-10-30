import { useState } from 'react';
import { Button, Card, Descriptions, Form, Input, InputNumber, Modal, Select, Space, Table, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { fetchAuditLogs } from '../../api/audit';
import type { AuditLogInfo, GetAuditLogListReq } from '../../api/types';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { formatTime } from '../../utils/format';
export default function AuditLogList() {
  const [form] = Form.useForm<GetAuditLogListReq>();
  const [params, setParams] = useState<GetAuditLogListReq>({ page: 1, page_size: 20 });
  const [detail, setDetail] = useState<AuditLogInfo | null>(null);
  const query = useQuery({ queryKey: ['audit-logs', params], queryFn: () => fetchAuditLogs(params) });
  return <div className="page-stack"><PageHeader title="审计日志" description="查询操作结果与请求记录" />
    <Card><Form name="audit-filter" form={form} layout="inline" onFinish={(values) => setParams({ ...values, page: 1, page_size: params.page_size })}>
      <Form.Item name="module" label="模块"><Input placeholder="如 sys/user" allowClear /></Form.Item>
      <Form.Item name="action" label="动作"><Input placeholder="如 create" allowClear /></Form.Item>
      <Form.Item name="user_id" label="用户 ID"><InputNumber min={1} precision={0} /></Form.Item>
      <Form.Item name="result" label="结果"><Select allowClear className="filter-select" placeholder="全部" options={[{ label: '成功', value: 1 }, { label: '失败', value: 2 }]} /></Form.Item>
      <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { form.resetFields(); setParams({ page: 1, page_size: params.page_size }); }}>重置</Button><Button aria-label="刷新审计日志" icon={<ReloadOutlined />} onClick={() => void query.refetch()} /></Space></Form.Item>
    </Form></Card>
    <Card><QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      <Table<AuditLogInfo> rowKey="id" dataSource={query.data?.list ?? []} loading={query.isFetching} scroll={{ x: 1000 }} pagination={{ current: params.page, pageSize: params.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, page_size) => setParams({ ...params, page, page_size }) }} columns={[
        { title: '操作人', dataIndex: 'username' }, { title: '模块', dataIndex: 'module' }, { title: '动作', dataIndex: 'action' },
        { title: '方法', dataIndex: 'method' }, { title: '路径', dataIndex: 'path', ellipsis: true },
        { title: '结果', dataIndex: 'result', render: (value: number) => <Tag color={value === 1 ? 'success' : 'error'}>{value === 1 ? '成功' : '失败'}</Tag> },
        { title: '耗时', dataIndex: 'duration_ms', render: (value: number) => value + ' ms' },
        { title: '时间', dataIndex: 'created_at', render: formatTime },
        { title: '操作', key: 'actions', render: (_, row) => <Button type="link" onClick={() => setDetail(row)}>详情</Button> },
      ]} />
    </Card>
    <Modal open={!!detail} title="审计详情" onCancel={() => setDetail(null)} footer={<Button onClick={() => setDetail(null)}>关闭</Button>}>
      {detail && <Descriptions column={1} items={[
        { key: 'user', label: '操作人', children: detail.username || '—' },
        { key: 'request', label: '请求', children: detail.method + ' ' + detail.path },
        { key: 'ip', label: '来源 IP', children: detail.ip || '—' },
        { key: 'error', label: '错误信息', children: detail.error_msg || '—' },
        { key: 'time', label: '时间', children: formatTime(detail.created_at) },
      ]} />}
    </Modal>
  </div>;
}
