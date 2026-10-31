import { useState } from 'react';
import { App, Button, Card, Form, InputNumber, Segmented, Select, Space } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deleteMenu, fetchMenus, fetchMenuTree } from '../../api/menu';
import type { GetMenuListReq, GetMenuTreeReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { buildTree } from '../../utils/tree';
import { invalidateMenuData } from '../../utils/queryInvalidation';
import { statusOptions } from '../../utils/formOptions';
import { MenuDetails } from './MenuDetails';
import { menuTypeOptions } from './menuOptions';
import { MenuForm } from './MenuForm';
import { MenuTable } from './MenuTable';

export default function MenuPage() {
  const { message } = App.useApp();
  const [mode, setMode] = useState<'list' | 'tree'>('list');
  const [filters, setFilters] = useState<GetMenuListReq>({ page: 1, page_size: 20 });
  const [treeStatus, setTreeStatus] = useState<number>();
  const [editing, setEditing] = useState<{ open: boolean; id: number | null }>({ open: false, id: null });
  const [detailId, setDetailId] = useState<number | null>(null);
  const [filterForm] = Form.useForm<GetMenuListReq & GetMenuTreeReq>();
  const queryClient = useQueryClient();
  const list = usePagedQuery({ params: filters, setParams: setFilters, queryKey: ['menus', 'list', filters], queryFn: () => fetchMenus(filters), enabled: mode === 'list' });
  const tree = useQuery({ queryKey: ['menus', 'tree', filters.type, treeStatus], queryFn: () => fetchMenuTree({ type: filters.type, status: treeStatus }), enabled: mode === 'tree' });
  const current = mode === 'list' ? list : tree;
  const remove = useMutation({ mutationFn: deleteMenu, onSuccess: async () => {
    await invalidateMenuData(queryClient);
    message.success('菜单已删除');
  } });
  return <div className="page-stack">
    <PageHeader title="菜单管理" description="维护导航目录、页面和按钮权限，支持列表与树形查看。"
      extra={<Auth code="sys:menu:add"><Button type="primary" icon={<PlusOutlined />} onClick={() => setEditing({ open: true, id: null })}>新增菜单</Button></Auth>} />
    <Card className="filter-card">
      <Form name="menu-filter" className="filter-form" form={filterForm} layout="inline" onFinish={({ status, ...values }) => {
        setTreeStatus(status); setFilters({ ...values, page: 1, page_size: filters.page_size });
      }}>
        <Form.Item name="type" label="类型"><Select className="filter-select" allowClear placeholder="全部类型" options={menuTypeOptions} /></Form.Item>
        {mode === 'list' && <Form.Item name="parent_id" label="父菜单 ID"><InputNumber min={1} precision={0} placeholder="全部" /></Form.Item>}
        {mode === 'tree' && <Form.Item name="status" label="状态"><Select className="filter-select" allowClear placeholder="全部" options={statusOptions} /></Form.Item>}
        <Form.Item><Space><Button htmlType="submit" type="primary">查询</Button><Button onClick={() => { filterForm.resetFields(); setTreeStatus(undefined); setFilters({ page: 1, page_size: 20 }); }}>重置</Button>
          <Button icon={<ReloadOutlined />} loading={current.isFetching} onClick={() => { void current.refetch(); }}>刷新</Button>
        </Space></Form.Item>
      </Form>
    </Card>
    <Card className="data-card" title={mode === 'list' ? '菜单列表' : '菜单结构'} extra={
      <Segmented value={mode} options={[{ label: '分页列表', value: 'list' }, { label: '菜单树', value: 'tree' }]} onChange={(value) => setMode(value as 'list' | 'tree')} />
    }>
      <QueryState isPending={current.isPending} error={current.error} onRetry={() => { void current.refetch(); }} />
      {!current.isPending && !current.error && <MenuTable rows={mode === 'list' ? list.data?.list ?? [] : buildTree(tree.data?.list ?? [])}
        pagination={mode === 'tree' ? false : { current: filters.page, pageSize: filters.page_size, total: list.data?.total ?? 0, showSizeChanger: true,
          onChange: (page, pageSize) => setFilters({ ...filters, page, page_size: pageSize }) }}
        deleting={remove.isPending} onDetails={setDetailId} onEdit={(id) => setEditing({ open: true, id })} onDelete={(id) => remove.mutate(id)} />}
    </Card>
    <MenuForm open={editing.open} editingId={editing.id} onClose={() => setEditing({ open: false, id: null })} />
    <MenuDetails id={detailId} onClose={() => setDetailId(null)} />
  </div>;
}
