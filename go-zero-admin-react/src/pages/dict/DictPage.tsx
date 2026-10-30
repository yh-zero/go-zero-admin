import { useState } from 'react';
import { App, Button, Card, Form, Input, Popconfirm, Select, Space, Table } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteDict, fetchDicts } from '../../api/dict';
import type { DictInfo, GetDictListReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { formatTime } from '../../utils/format';
import { DictDetails } from './DictDetails';
import { DictForm } from './DictForm';

export default function DictPage() {
  const { message } = App.useApp();
  const [filters, setFilters] = useState<GetDictListReq>({ page: 1, page_size: 20 });
  const [editing, setEditing] = useState<{ open: boolean; code: string | null }>({ open: false, code: null });
  const [detailCode, setDetailCode] = useState<string | null>(null);
  const [filterForm] = Form.useForm<GetDictListReq>();
  const queryClient = useQueryClient();
  const query = usePagedQuery({ params: filters, setParams: setFilters, queryKey: ['dicts', 'list', filters], queryFn: () => fetchDicts(filters) });
  const remove = useMutation({ mutationFn: deleteDict, onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['dicts'] });
    message.success('字典已删除');
  } });
  return <>
    <PageHeader title="字典管理" description="配置字典编码、启停状态和可选值。"
      extra={<Auth code="sys:dict:add"><Button type="primary" icon={<PlusOutlined />} onClick={() => setEditing({ open: true, code: null })}>新增字典</Button></Auth>} />
    <Card>
      <Form name="dict-filter" form={filterForm} layout="inline" onFinish={(values) => setFilters({ ...values, page: 1, page_size: filters.page_size })}>
        <Form.Item name="name" label="名称"><Input allowClear /></Form.Item>
        <Form.Item name="code" label="编码"><Input allowClear /></Form.Item>
        <Form.Item name="status" label="状态"><Select allowClear placeholder="全部" style={{ minWidth: 110 }} options={[{ value: 1, label: '启用' }, { value: 2, label: '停用' }]} /></Form.Item>
        <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { filterForm.resetFields(); setFilters({ page: 1, page_size: 20 }); }}>重置</Button>
          <Button icon={<ReloadOutlined />} loading={query.isFetching} onClick={() => { void query.refetch(); }}>刷新</Button></Space></Form.Item>
      </Form>
      <QueryState isPending={query.isPending} error={query.error} onRetry={() => { void query.refetch(); }} />
      {!query.isPending && !query.error && <Table<DictInfo> rowKey="id" dataSource={query.data?.list ?? []} scroll={{ x: 1000 }}
        pagination={{ current: filters.page, pageSize: filters.page_size, total: query.data?.total ?? 0, showSizeChanger: true,
          onChange: (page, pageSize) => setFilters({ ...filters, page, page_size: pageSize }) }} columns={[
          { title: '名称', dataIndex: 'name', width: 160 }, { title: '编码', dataIndex: 'code', width: 160 },
          { title: '描述', dataIndex: 'description', ellipsis: true },
          { title: '字典项', key: 'items', width: 90, render: (_, record) => record.items?.length ?? 0 },
          { title: '状态', dataIndex: 'status', width: 90, render: (status: number) => <StatusTag status={status} /> },
          { title: '更新时间', dataIndex: 'updated_at', width: 170, render: formatTime },
          { title: '操作', key: 'actions', fixed: 'right', width: 210, render: (_, record) => <Space size="small">
            <Button type="link" size="small" onClick={() => setDetailCode(record.code)}>详情</Button>
            <Auth code="sys:dict:edit"><Button type="link" size="small" onClick={() => setEditing({ open: true, code: record.code })}>编辑</Button></Auth>
            <Auth code="sys:dict:delete"><Popconfirm title={`删除字典“${record.name}”？`} description="字典及其字典项将不再出现在列表中。"
              onConfirm={() => remove.mutate(record.id)} okButtonProps={{ loading: remove.isPending }}><Button type="link" size="small" danger disabled={remove.isPending}>删除</Button></Popconfirm></Auth>
          </Space> },
        ]} />}
    </Card>
    <DictForm open={editing.open} editingCode={editing.code} onClose={() => setEditing({ open: false, code: null })} />
    <DictDetails code={detailCode} onClose={() => setDetailCode(null)} />
  </>;
}
