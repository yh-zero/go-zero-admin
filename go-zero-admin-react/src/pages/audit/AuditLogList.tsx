import { useState } from 'react';
import { App, AutoComplete, Button, Card, DatePicker, Descriptions, Form, Input, InputNumber, Modal, Select, Space, Table, Tag } from 'antd';
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import dayjs, { type Dayjs } from 'dayjs';
import { exportAuditLogs, fetchAuditLogs } from '../../api/audit';
import type { AuditLogInfo, GetAuditLogListReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { AdvancedFilterToggle } from '../../components/AdvancedFilterToggle';
import { formatTime } from '../../utils/format';
import { queryKeys } from '../../utils/queryKeys';
import { useTablePagination } from '../../hooks/useTablePagination';

const actionOptions = [
  { value: 'post', label: 'POST 请求（含创建）' }, { value: 'put', label: 'PUT 请求' }, { value: 'delete', label: 'DELETE 请求' },
  { value: 'login', label: '登录（login）' }, { value: 'logout', label: '退出（logout）' },
  { value: 'refresh', label: '刷新令牌（refresh）' }, { value: 'sessions', label: '撤销设备会话（sessions）' },
  { value: 'register', label: '注册（register）' }, { value: 'email-code', label: '发送邮箱验证码（email-code）' },
];
type AuditForm = GetAuditLogListReq & { start_time?: Dayjs; end_time?: Dayjs };
const seconds = (value?: Dayjs) => value?.unix();

export default function AuditLogList() {
  const [form] = Form.useForm<AuditForm>();
  const [params, setParams] = useState<GetAuditLogListReq>({ page: 1, page_size: 20 });
  const [detail, setDetail] = useState<AuditLogInfo | null>(null);
  const [exporting, setExporting] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const { message } = App.useApp();
  const query = useQuery({ queryKey: [queryKeys.auditLogs, params], queryFn: () => fetchAuditLogs(params) });
  const pagination = useTablePagination(params, setParams, query.data?.total ?? 0);
  const submit = (values: AuditForm) => setParams({ ...values, action: values.action || undefined, start_time: seconds(values.start_time), end_time: seconds(values.end_time), page: 1, page_size: params.page_size });
  const advancedCount = [params.user_id, params.method, params.path, params.ip, params.min_duration_ms, params.max_duration_ms].filter(value => value !== undefined && value !== '').length;
  const download = async () => {
    setExporting(true);
    try {
      const blob = await exportAuditLogs(params);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url; anchor.download = 'audit-logs-' + dayjs().format('YYYYMMDD-HHmmss') + '.csv';
      document.body.appendChild(anchor); anchor.click(); anchor.remove(); URL.revokeObjectURL(url);
      void message.success('审计日志已导出');
    } finally { setExporting(false); }
  };
  return <div className="page-stack">
    <Card className="filter-card"><Form name="audit-filter" className={`filter-form${showAdvanced ? ' is-expanded' : ''}`} form={form} layout="inline" onFinish={submit}>
      <Form.Item name="module" label="模块"><Input placeholder="如 sys/user" allowClear /></Form.Item>
      <Form.Item name="action" label="动作"><AutoComplete allowClear className="filter-select" placeholder="选择或输入动作"
        options={actionOptions} popupMatchSelectWidth={false} showSearch={{ filterOption: (input, option) => option?.label.toLowerCase().includes(input.toLowerCase()) ?? false }} /></Form.Item>
      <Form.Item name="username" label="操作人"><Input allowClear maxLength={64} placeholder="精确用户名" /></Form.Item>
      <Form.Item name="result" label="结果"><Select allowClear className="filter-select" placeholder="全部" options={[{ label: '成功', value: 1 }, { label: '失败', value: 2 }]} /></Form.Item>
      <Form.Item name="start_time" label="开始时间"><DatePicker showTime /></Form.Item>
      <Form.Item name="end_time" label="结束时间"><DatePicker showTime /></Form.Item>
      {showAdvanced && <>
        <Form.Item name="user_id" label="用户 ID"><InputNumber min={1} precision={0} /></Form.Item>
        <Form.Item name="method" label="方法"><Select allowClear className="filter-select" options={['GET', 'POST', 'PUT', 'DELETE'].map(value => ({ value, label: value }))} /></Form.Item>
        <Form.Item name="path" label="路径"><Input allowClear maxLength={255} placeholder="模糊匹配路径" /></Form.Item>
        <Form.Item name="ip" label="来源 IP"><Input allowClear maxLength={64} placeholder="精确 IP" /></Form.Item>
        <Form.Item name="min_duration_ms" label="最小耗时"><InputNumber min={0} precision={0} suffix="ms" /></Form.Item>
        <Form.Item name="max_duration_ms" label="最大耗时"><InputNumber min={0} precision={0} suffix="ms" /></Form.Item>
      </>}
      <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { form.resetFields(); setParams({ page: 1, page_size: params.page_size }); setShowAdvanced(false); }}>重置</Button>
        <Button aria-label="刷新审计日志" icon={<ReloadOutlined />} onClick={() => void query.refetch()} />
        <AdvancedFilterToggle open={showAdvanced} activeCount={advancedCount} onChange={setShowAdvanced} /></Space></Form.Item>
    </Form></Card>
    <Card className="data-card" title="操作记录" extra={<Button icon={<DownloadOutlined />} loading={exporting} onClick={() => void download()}>导出 CSV</Button>}><QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
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
        { key: 'agent', label: 'User-Agent', children: detail.user_agent || '—' },
        { key: 'duration', label: '耗时', children: detail.duration_ms + ' ms' },
        { key: 'error', label: '错误信息', children: detail.error_msg || '—' },
        { key: 'time', label: '时间', children: formatTime(detail.created_at) },
      ]} />}
    </Modal>
  </div>;
}
