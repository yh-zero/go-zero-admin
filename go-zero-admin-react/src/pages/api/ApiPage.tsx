import { useState } from 'react';
import { App, Button, Card, Form, Input, Popconfirm, Select, Space, Table, Tag } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { deleteApiResource, fetchApiResources } from '../../api/resource';
import type { ApiInfo, GetApiListReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { formatTime } from '../../utils/format';

export default function ApiPage() {
  const { message } = App.useApp();
  const [filters, setFilters] = useState<GetApiListReq>({ page: 1, page_size: 20 });
  const [form] = Form.useForm<GetApiListReq>();
  const queryClient = useQueryClient();
  const query = usePagedQuery({ params: filters, setParams: setFilters, queryKey: ['api-resources', filters], queryFn: () => fetchApiResources(filters) });
  const remove = useMutation({ mutationFn: deleteApiResource, onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['api-resources'] });
    message.success('API 资源登记已删除');
  } });
  return <>
    <PageHeader title="API 资源" description="维护接口资源目录，支持按分组、方法和路径查询。" />
    <Card>
      <Form name="api-filter" form={form} layout="inline" onFinish={(values) => setFilters({ ...values, page: 1, page_size: filters.page_size })}>
        <Form.Item name="group" label="分组"><Input allowClear placeholder="例如 sys/user" /></Form.Item>
        <Form.Item name="method" label="方法"><Select allowClear placeholder="全部" style={{ minWidth: 120 }}
          options={['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'].map((method) => ({ label: method, value: method }))} /></Form.Item>
        <Form.Item name="path" label="路径"><Input allowClear placeholder="按路径搜索" /></Form.Item>
        <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button><Button onClick={() => { form.resetFields(); setFilters({ page: 1, page_size: 20 }); }}>重置</Button>
          <Button icon={<ReloadOutlined />} loading={query.isFetching} onClick={() => { void query.refetch(); }}>刷新</Button></Space></Form.Item>
      </Form>
      <QueryState isPending={query.isPending} error={query.error} onRetry={() => { void query.refetch(); }} />
      {!query.isPending && !query.error && <Table<ApiInfo> rowKey="id" dataSource={query.data?.list ?? []} scroll={{ x: 950 }}
        pagination={{ current: filters.page, pageSize: filters.page_size, total: query.data?.total ?? 0, showSizeChanger: true,
          onChange: (page, pageSize) => setFilters({ ...filters, page, page_size: pageSize }) }} columns={[
          { title: 'ID', dataIndex: 'id', width: 70 }, { title: '分组', dataIndex: 'group', width: 150 },
          { title: '方法', dataIndex: 'method', width: 110, render: (method: string) => <Tag>{method}</Tag> },
          { title: '路径', dataIndex: 'path', ellipsis: true }, { title: '说明', dataIndex: 'title', ellipsis: true },
          { title: '创建时间', dataIndex: 'created_at', width: 170, render: formatTime },
          { title: '操作', key: 'actions', width: 90, fixed: 'right', render: (_, record) => <Auth code="sys:api:delete">
            <Popconfirm title="删除 API 资源登记？" description="只删除目录登记，不会移除后端实际接口。" onConfirm={() => remove.mutate(record.id)} okButtonProps={{ loading: remove.isPending }}>
              <Button type="link" size="small" danger disabled={remove.isPending}>删除</Button>
            </Popconfirm>
          </Auth> },
        ]} />}
    </Card>
  </>;
}
