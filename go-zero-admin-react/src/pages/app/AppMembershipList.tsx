import { useEffect, useState } from 'react';
import { App, Avatar, Button, Card, Checkbox, Form, Input, InputNumber, Popconfirm, Space, Table, Tag } from 'antd';
import { CrownOutlined, PlusOutlined, ReloadOutlined, RollbackOutlined, SearchOutlined, StopOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { fetchAppMemberships, revokeAppMembership } from '../../api/app';
import type { AppMembershipInfo, GetAppMembershipListReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { formatTime } from '../../utils/format';
import { invalidateAppData } from '../../utils/queryInvalidation';
import { AppMembershipGrantDialog } from './AppMembershipGrantDialog';
import './app.css';

type MembershipFilters = Omit<GetAppMembershipListReq, 'page' | 'page_size'>;
interface PageProps {
  initialFilters: MembershipFilters; pageSize: number;
  onPageSizeChange: (value: number) => void; onFiltersChange: (values: MembershipFilters) => void;
}

function queryUserId(value: string | null): number | undefined {
  const number = value === null || !/^\d+$/.test(value) ? 0 : Number(value);
  return Number.isSafeInteger(number) && number > 0 ? number : undefined;
}
function membershipStatus(row: AppMembershipInfo, now: number) {
  if (row.revoked) return <Tag>已撤销</Tag>;
  if (row.starts_at > now) return <Tag color="processing">未生效</Tag>;
  if (row.expires_at <= now) return <Tag color="orange">已过期</Tag>;
  return <Tag color="success">有效</Tag>;
}

function MembershipDate({ seconds }: { seconds: number }) {
  const [date, time] = formatTime(seconds).split(' ');
  return <Space orientation="vertical" size={2} className="app-date-text"><span className="app-primary-text">{date}</span>
    {time && <span className="app-secondary-text">{time}</span>}</Space>;
}

export default function AppMembershipList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [pageSize, setPageSize] = useState(20);
  const filters: MembershipFilters = { user_id: queryUserId(searchParams.get('user_id')),
    plan_code: searchParams.get('plan_code')?.trim() || undefined,
    active_only: searchParams.get('active_only') === 'true' || undefined };
  const applyFilters = (values: MembershipFilters) => setSearchParams(current => {
    const next = new URLSearchParams(current);
    if (values.user_id) next.set('user_id', String(values.user_id)); else next.delete('user_id');
    if (values.plan_code) next.set('plan_code', values.plan_code); else next.delete('plan_code');
    if (values.active_only) next.set('active_only', 'true'); else next.delete('active_only');
    return next;
  }, { replace: true });
  return <MembershipPage key={JSON.stringify(filters)} initialFilters={filters} pageSize={pageSize}
    onPageSizeChange={setPageSize} onFiltersChange={applyFilters} />;
}

function MembershipPage({ initialFilters, pageSize, onPageSizeChange, onFiltersChange }: PageProps) {
  const [form] = Form.useForm<GetAppMembershipListReq>();
  const [params, setParams] = useState<GetAppMembershipListReq>(() => ({ ...initialFilters, page: 1, page_size: pageSize }));
  const [grantOpen, setGrantOpen] = useState(false);
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  const { message } = App.useApp();
  const client = useQueryClient();
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Math.floor(Date.now() / 1000)), 30_000);
    return () => window.clearInterval(timer);
  }, []);
  const memberships = usePagedQuery({ params, setParams, queryKey: ['app-memberships', params], queryFn: () => fetchAppMemberships(params) });
  const revoke = useMutation({ mutationFn: revokeAppMembership, onSuccess: async () => {
    await invalidateAppData(client);
    void message.success('会员已撤销');
  } });
  const submitFilters = (values: GetAppMembershipListReq) => {
    const filters = { ...values, plan_code: values.plan_code?.trim() || undefined };
    onFiltersChange(filters);
    setParams({ ...filters, page: 1, page_size: params.page_size });
  };
  return <div className="page-stack app-page">
    <div className="app-page-heading"><span className="app-heading-icon app-heading-icon--amber" aria-hidden="true"><CrownOutlined /></span>
    <PageHeader title="会员管理" description="查看 APP 用户的会员权益、有效期和开通记录" extra={<Auth code="sys:app:membership:grant">
      <Button type="primary" icon={<PlusOutlined />} onClick={() => setGrantOpen(true)}>开通会员</Button>
    </Auth>} /></div>
    <Card className="filter-card app-filter-card"><Form<GetAppMembershipListReq> form={form} name="app-membership-filter"
      className="filter-form" layout="inline" initialValues={initialFilters} onFinish={submitFilters}>
      <Form.Item name="user_id" label="用户 ID" rules={[
        { type: 'integer', min: 1, max: Number.MAX_SAFE_INTEGER, message: '请输入有效用户 ID' }]}>
        <InputNumber prefix={<UserOutlined />} min={1} max={Number.MAX_SAFE_INTEGER} precision={0} placeholder="全部用户" />
      </Form.Item>
      <Form.Item name="plan_code" label="套餐标识" rules={[
        { validator: (_, value?: string) => !value || new TextEncoder().encode(value.trim()).length <= 32
          ? Promise.resolve() : Promise.reject(new Error('套餐标识最多 32 字节')) }]}>
        <Input allowClear prefix={<SearchOutlined />} placeholder="搜索套餐标识" maxLength={32} />
      </Form.Item>
      <Form.Item name="active_only" label="有效状态" valuePropName="checked"><Checkbox>仅有效会员</Checkbox></Form.Item>
      <Form.Item><Space>
        <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>查询</Button>
        <Button icon={<RollbackOutlined />} onClick={() => { form.resetFields(); form.setFieldsValue({ user_id: undefined, plan_code: undefined, active_only: false }); submitFilters({}); }}>重置</Button>
        <Button icon={<ReloadOutlined />} aria-label="刷新会员列表" onClick={() => void memberships.refetch()} loading={memberships.isFetching} />
      </Space></Form.Item>
    </Form></Card>
    <Card className="data-card app-data-card" title={<div className="app-section-title"><span className="app-section-icon" aria-hidden="true"><CrownOutlined /></span>
      <span>会员记录</span>{memberships.isSuccess && <span className="app-count">{memberships.data.total}</span>}</div>}>
      <QueryState isPending={memberships.isPending} error={memberships.error} onRetry={() => memberships.refetch()} />
      <Table<AppMembershipInfo> rowKey="id" dataSource={memberships.data?.list ?? []} loading={memberships.isFetching} scroll={{ x: 1200 }}
        pagination={{ current: params.page, pageSize: params.page_size, total: memberships.data?.total ?? 0, showSizeChanger: true,
          showTotal: total => `共 ${total} 条记录`, onChange: (page, page_size) => {
            onPageSizeChange(page_size); setParams({ ...params, page, page_size });
          } }} columns={[
          { title: '用户', dataIndex: 'user_id', width: 180, render: (id: number) => <div className="app-table-identity">
            <Avatar size={34} className="app-avatar--amber" icon={<UserOutlined />} />
            <div className="app-identity-copy"><span className="app-primary-text">#{id}</span><span className="app-secondary-text">用户 ID</span></div>
          </div> },
          { title: '套餐标识', dataIndex: 'plan_code', width: 150, render: (value: string) => <span className="app-plan-pill"><CrownOutlined aria-hidden />{value}</span> },
          { title: '会员权益', dataIndex: 'benefits', ellipsis: true, render: (value: string) => <span className="app-secondary-text">{value || '—'}</span> },
          { title: '状态', key: 'status', width: 105, render: (_, row) => membershipStatus(row, now) },
          { title: '生效时间', dataIndex: 'starts_at', width: 150, render: (seconds: number) => <MembershipDate seconds={seconds} /> },
          { title: '到期时间', dataIndex: 'expires_at', width: 150, render: (seconds: number) => <MembershipDate seconds={seconds} /> },
          { title: '开通时间', dataIndex: 'created_at', width: 150, render: (seconds: number) => <MembershipDate seconds={seconds} /> },
          { title: '操作', key: 'actions', width: 95, render: (_, row) => <Space className="app-table-actions"><Auth code="sys:app:membership:revoke">
            <Popconfirm title="确认撤销这项会员？" description="撤销后对应权益立即失效，原记录会保留。"
              disabled={row.revoked || revoke.isPending} onConfirm={() => revoke.mutateAsync(row.id).catch(() => undefined)}
              okButtonProps={{ loading: revoke.isPending && revoke.variables === row.id }}>
              <Button type="link" danger icon={<StopOutlined />} disabled={row.revoked || revoke.isPending}>撤销</Button>
            </Popconfirm>
          </Auth></Space> },
        ]} />
    </Card>
    {grantOpen && <AppMembershipGrantDialog initialUserId={params.user_id} onClose={() => setGrantOpen(false)} />}
  </div>;
}
