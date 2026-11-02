import { useState } from 'react';
import { App, Button, Form, Input, Modal, Popconfirm, Result, Space, Table, Tag } from 'antd';
import { KeyOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createTenant, deleteTenant, fetchTenants, resetTenantAdmin, startTenant, stopTenant } from '../../api/tenant';
import type { CreateTenantReq, PageResult, TenantInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { formatTime } from '../../utils/format';
import { invalidateTenantData } from '../../utils/queryInvalidation';

interface TenantFilters { page: number; page_size: number; status?: string }

const STATUS_META: Record<string, { color: string; text: string }> = {
  creating: { color: 'processing', text: '创建中' },
  active: { color: 'success', text: '运行中' },
  stopped: { color: 'warning', text: '已停用' },
  error: { color: 'error', text: '异常' },
};

/** 一次性密码展示弹窗：关闭即销毁，不落任何存储 */
function PasswordResultModal({ open, title, password, onClose }: { open: boolean; title: string; password: string; onClose: () => void }) {
  return (
    <Modal open={open} title={title} footer={<Button type="primary" onClick={onClose}>我已妥善保存</Button>} onCancel={onClose} maskClosable={false} keyboard={false}>
      <Result status="success" title="操作成功" subTitle="以下密码仅此一次展示，关闭后无法找回，请立即复制保存。" />
      <Input.Password value={password} readOnly bordered={false} style={{ fontSize: 16, textAlign: 'center' }} />
    </Modal>
  );
}

export default function TenantListPage() {
  const [filters, setFilters] = useState<TenantFilters>({ page: 1, page_size: 20 });
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm<CreateTenantReq>();
  const [secret, setSecret] = useState<{ title: string; password: string } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<TenantInfo | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState('');
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  // creating 状态存在时轮询（provisioning 为异步，通常 1-3 分钟）
  const query = usePagedQuery<TenantFilters, TenantInfo>({
    params: filters, setParams: setFilters,
    queryKey: ['tenant-list', filters],
    queryFn: () => fetchTenants(filters),
    refetchInterval: (q: { state: { data?: PageResult<TenantInfo> } }) =>
      (q.state.data?.list ?? []).some((t: TenantInfo) => t.status === 'creating') ? 5000 : false,
  } as never);

  const refresh = () => invalidateTenantData(queryClient);
  const lifecycle = useMutation({
    mutationFn: async ({ action, id }: { action: 'start' | 'stop'; id: number }) => {
      if (action === 'start') await startTenant(id); else await stopTenant(id);
    },
    onSuccess: () => { message.success('操作已提交'); refresh(); },
    onError: () => refresh(),
  });
  const remove = useMutation({
    mutationFn: ({ id, slug }: { id: number; slug: string }) => deleteTenant(id, slug),
    onSuccess: () => { message.success('租户已删除'); setDeleteTarget(null); setDeleteConfirm(''); refresh(); },
    onError: () => refresh(),
  });
  const reset = useMutation({
    mutationFn: (id: number) => resetTenantAdmin(id),
    onSuccess: (data) => setSecret({ title: '租户管理员密码已重置', password: data.new_password }),
    onError: () => refresh(),
  });
  const create = useMutation({
    mutationFn: (values: CreateTenantReq) => createTenant(values),
    onSuccess: (data) => {
      setCreateOpen(false); createForm.resetFields();
      setSecret({ title: `租户「${data.tenant.slug}」创建中`, password: data.admin_password });
      refresh();
    },
    onError: () => undefined,
  });

  const columns = [
    { title: '标识', dataIndex: 'slug', render: (v: string, r: TenantInfo) => <span>{v} {r.admin_account_ready ? null : <Tag>初始化中</Tag>}</span> },
    { title: '域名', dataIndex: 'domain' },
    { title: '状态', dataIndex: 'status', render: (v: string, r: TenantInfo) => (
      <Space direction="vertical" size={0}>
        <Tag color={STATUS_META[v]?.color}>{STATUS_META[v]?.text ?? v}</Tag>
        {v === 'error' && r.error_msg ? <span style={{ fontSize: 12, color: '#cf1322' }}>{r.error_msg}</span> : null}
      </Space>
    ) },
    { title: '创建时间', dataIndex: 'created_at', render: formatTime },
    {
      title: '操作', key: 'actions', render: (_: unknown, r: TenantInfo) => (
        <Space wrap>
          <Auth code="sys:tenant:lifecycle">
            {r.status === 'active' ? (
              <Popconfirm title={`确认停用「${r.slug}」？`} description="停用后该租户所有服务停止响应。" onConfirm={() => lifecycle.mutate({ action: 'stop', id: r.id })}>
                <Button type="link" size="small" loading={lifecycle.isPending}>停用</Button>
              </Popconfirm>
            ) : r.status === 'stopped' || r.status === 'error' ? (
              <Popconfirm title={`确认启用「${r.slug}」？`} onConfirm={() => lifecycle.mutate({ action: 'start', id: r.id })}>
                <Button type="link" size="small" loading={lifecycle.isPending}>启用</Button>
              </Popconfirm>
            ) : <Tag>创建中</Tag>}
          </Auth>
          <Auth code="sys:tenant:reset">
            <Popconfirm title={`重置「${r.slug}」管理员密码？`} description="旧密码立即失效，新密码仅此一次展示。" onConfirm={() => reset.mutate(r.id)}>
              <Button type="link" size="small" icon={<KeyOutlined />} loading={reset.isPending}>重置密码</Button>
            </Popconfirm>
          </Auth>
          <Auth code="sys:tenant:delete">
            <Button type="link" size="small" danger onClick={() => { setDeleteTarget(r); setDeleteConfirm(''); }}>删除</Button>
          </Auth>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <PageHeader title="租户管理" description="每租户独立整套服务与数据库，子域名经 Nginx 路由。" extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => query.refetch()}>刷新</Button>
          <Auth code="sys:tenant:create">
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>创建租户</Button>
          </Auth>
        </Space>
      } />
      <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      <Table<TenantInfo>
        rowKey="id" loading={query.isFetching} columns={columns} dataSource={query.data?.list ?? []}
        pagination={{ current: filters.page, pageSize: filters.page_size, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (page, pageSize) => setFilters({ page, page_size: pageSize }) }}
      />

      <Modal open={createOpen} title="创建租户" onCancel={() => setCreateOpen(false)} onOk={() => createForm.submit()} confirmLoading={create.isPending} destroyOnHidden>
        <Form form={createForm} layout="vertical" onFinish={(values) => create.mutate(values)}>
          <Form.Item name="slug" label="租户标识" rules={[{ required: true, pattern: /^[a-z0-9](?:[a-z0-9-]{0,30}[a-z0-9])?$/, message: '2-32 位小写字母、数字、连字符，首尾不能为连字符' }]}>
            <Input placeholder="如 t2，将用于子域名 t2.localhost" />
          </Form.Item>
          <Form.Item name="admin_username" label="管理员用户名" rules={[{ pattern: /^[a-zA-Z][a-zA-Z0-9_]{2,31}$/, message: '3-32 位字母、数字或下划线，以字母开头' }]}>
            <Input placeholder="默认 admin" />
          </Form.Item>
          <Form.Item name="admin_password" label="管理员初始密码" tooltip="留空则自动生成随机密码，创建成功后仅此一次展示。">
            <Input.Password placeholder="留空自动生成" />
          </Form.Item>
        </Form>
      </Modal>

      <PasswordResultModal open={!!secret} title={secret?.title ?? ''} password={secret?.password ?? ''} onClose={() => setSecret(null)} />

      <Modal open={!!deleteTarget} title={`删除租户「${deleteTarget?.slug}」`} onCancel={() => setDeleteTarget(null)}
        okButtonProps={{ danger: true, disabled: deleteConfirm !== deleteTarget?.slug, loading: remove.isPending }}
        onOk={() => deleteTarget && remove.mutate({ id: deleteTarget.id, slug: deleteConfirm })}>
        <p>将执行 <code>docker compose down -v</code>：该租户<strong>全部服务与数据卷被永久删除</strong>，不可恢复。</p>
        <p>请输入租户标识 <Tag>{deleteTarget?.slug}</Tag> 以确认：</p>
        <Input value={deleteConfirm} onChange={(e) => setDeleteConfirm(e.target.value)} placeholder={deleteTarget?.slug} />
      </Modal>
    </div>
  );
}
