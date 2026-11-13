import { useState } from 'react';
import { App, Button, Card, Form, Input, Select, Space } from 'antd';
import { ApartmentOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deleteMenu, fetchMenuTree } from '../../api/menu';
import { Auth } from '../../components/Auth';
import { QueryState } from '../../components/QueryState';
import { buildTree } from '../../utils/tree';
import { invalidateMenuData } from '../../utils/queryInvalidation';
import { statusOptions } from '../../utils/formOptions';
import { MenuDetails } from './MenuDetails';
import { MenuForm } from './MenuForm';
import type { MenuEditorState } from './MenuForm';
import { MenuTable } from './MenuTable';
import { OperationPermissions } from './OperationPermissions';
import { filterNavigationMenus, getUnassignedPermissions } from './menuResources';
import styles from './MenuPage.module.css';
import { queryKeys } from '../../utils/queryKeys';

interface MenuFilters { keyword?: string; status?: number }

export default function MenuPage() {
  const { message } = App.useApp();
  const [filters, setFilters] = useState<MenuFilters>({});
  const [editing, setEditing] = useState<MenuEditorState | null>(null);
  const [detailId, setDetailId] = useState<number | null>(null);
  const [pageId, setPageId] = useState<number | null>(null);
  const [filterForm] = Form.useForm<MenuFilters>();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: [queryKeys.menus, 'resources'], queryFn: () => fetchMenuTree() });
  const resources = query.data?.list ?? [];
  const ready = query.isSuccess && !query.isFetching;
  const unassignedCount = getUnassignedPermissions(resources).length;
  const remove = useMutation({ mutationFn: deleteMenu, onSuccess: async () => {
    await invalidateMenuData(queryClient);
    message.success('菜单已删除');
  } });
  const busy = query.isFetching || remove.isPending;
  return <div className={`page-stack ${pageId !== null ? styles.withPermissions : ''}`}>
    <Card className={`filter-card ${styles.filterCard}`}><Form name="menu-filter" className="filter-form" form={filterForm} layout="inline" onFinish={setFilters}>
      <Form.Item name="keyword" label="菜单名称"><Input allowClear prefix={<SearchOutlined />} placeholder="搜索菜单名称" /></Form.Item>
      <Form.Item name="status" label="状态"><Select className="filter-select" allowClear placeholder="全部状态" options={statusOptions} /></Form.Item>
      <Form.Item><Space><Button htmlType="submit" type="primary">查询</Button>
        <Button onClick={() => { filterForm.resetFields(); setFilters({}); }}>重置</Button>
        <Button icon={<ReloadOutlined />} loading={query.isFetching} onClick={() => { void query.refetch(); }}>刷新</Button>
      </Space></Form.Item>
    </Form></Card>
    <Card className={`data-card ${styles.menuCard}`} title={<div className={styles.structureHeading}>
      <span className={styles.structureHeadingIcon}><ApartmentOutlined /></span><span>菜单结构</span>
      {query.isSuccess && <span className={styles.structureCount}>{resources.filter(menu => menu.type === 1).length} 个目录 · {resources.filter(menu => menu.type === 2).length} 个页面</span>}
    </div>} extra={<Space wrap>
      {unassignedCount > 0 && <Button type="link" disabled={busy} onClick={() => setPageId(0)}>其他操作权限（{unassignedCount}）</Button>}
      <Auth code="sys:menu:add"><Button type="primary" icon={<PlusOutlined />} disabled={!ready}
        onClick={() => setEditing({ id: null, kind: 'navigation' })}>新增菜单</Button></Auth>
    </Space>}>
      <QueryState isPending={query.isPending} error={query.error} onRetry={() => { void query.refetch(); }} />
      {query.isSuccess && <MenuTable key={`${filters.keyword ?? ''}:${filters.status ?? ''}`}
        rows={buildTree(filterNavigationMenus(resources, filters.keyword ?? '', filters.status))} resources={resources}
        selectedId={pageId} busy={busy} onDetails={setDetailId}
        onEdit={id => setEditing({ id, kind: 'navigation' })} onDelete={id => remove.mutate(id)} onPermissions={setPageId} />}
      <div className={styles.tableHint}>操作权限在对应页面中管理，点击菜单名称可查看完整配置。</div>
    </Card>
    <OperationPermissions pageId={pageId} resources={resources} ready={ready} deleting={remove.isPending}
      onClose={() => setPageId(null)} onEdit={setEditing} onDelete={id => remove.mutate(id)} />
    {editing && <MenuForm key={`${editing.kind}:${editing.id ?? 'new'}:${editing.parentId ?? ''}`}
      editor={editing} resources={resources} resourcesReady={ready} onClose={() => setEditing(null)} />}
    <MenuDetails id={detailId} onClose={() => setDetailId(null)} />
  </div>;
}
