import { useState } from 'react';
import { App, Button, Card, Empty, Input, Popconfirm, Select, Space, Tree } from 'antd';
import type { TreeDataNode } from 'antd';
import { ApartmentOutlined, DeleteOutlined, EditOutlined, MailOutlined, PhoneOutlined, PlusOutlined, ReloadOutlined, SearchOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deleteDept, fetchDeptTree } from '../../api/dept';
import { fetchMyMenus } from '../../api/auth';
import type { DeptInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { getAccessiblePagePaths } from '../../router/menuRoutes';
import { buildTree } from '../../utils/tree';
import type { TreeNode } from '../../utils/tree';
import { statusOptions } from '../../utils/formOptions';
import DeptForm from './DeptForm';
import { DeptMembers } from './DeptMembers';
import './dept.css';
import { queryKeys } from '../../utils/queryKeys';

function filterDepartments(departments: DeptInfo[], keyword: string, status?: number) {
  const byId = new Map(departments.map(dept => [dept.id, dept]));
  const included = new Set<number>();
  for (const dept of departments) {
    if (!dept.name.toLocaleLowerCase().includes(keyword.trim().toLocaleLowerCase()) || status !== undefined && dept.status !== status) continue;
    let current: DeptInfo | undefined = dept;
    while (current && !included.has(current.id)) {
      included.add(current.id);
      current = byId.get(current.parent_id);
    }
  }
  return departments.filter(dept => included.has(dept.id));
}

function departmentNodes(departments: TreeNode<DeptInfo>[]): TreeDataNode[] {
  return departments.map(dept => ({
    key: dept.id,
    icon: <span className="dept-node-icon">{dept.children?.length ? <ApartmentOutlined /> : <TeamOutlined />}</span>,
    title: <span className="dept-tree-title"><span className={dept.status === 1 ? '' : 'dept-disabled-name'}>{dept.name}</span>
      <span className="dept-tree-count">{dept.members_visible ? `${dept.member_count} 人` : '—'}</span></span>,
    children: dept.children?.length ? departmentNodes(dept.children) : undefined,
  }));
}

function departmentTrail(departments: DeptInfo[], selected: DeptInfo) {
  const byId = new Map(departments.map(dept => [dept.id, dept]));
  const names: string[] = [];
  const seen = new Set<number>([selected.id]);
  let parent = byId.get(selected.parent_id);
  while (parent && !seen.has(parent.id)) {
    names.unshift(parent.name);
    seen.add(parent.id);
    parent = byId.get(parent.parent_id);
  }
  return names.join(' / ') || '组织架构';
}

export default function DeptList() {
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<number>();
  const [selectedId, setSelectedId] = useState<number>();
  const [editor, setEditor] = useState<{ dept?: DeptInfo; parentId?: number } | null>(null);
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: [queryKeys.depts, {}], queryFn: () => fetchDeptTree() });
  const menus = useQuery({ queryKey: [queryKeys.myMenus], queryFn: fetchMyMenus });
  const canReadUsers = menus.isSuccess && getAccessiblePagePaths(menus.data).has('/sys/user');
  const departments = query.data?.list ?? [];
  const visible = filterDepartments(departments, keyword, status);
  const selected = visible.find(dept => dept.id === selectedId) ?? visible[0];
  const remove = useMutation({
    mutationFn: deleteDept,
    onSuccess: async () => {
      await Promise.all(['depts', 'my-depts', 'profile', 'users'].map(key => queryClient.invalidateQueries({ queryKey: [key] })));
      message.success('部门已删除');
    },
  });
  const actions = selected && <Space className="dept-overview-actions" wrap>
    <Auth code="sys:dept:add"><Button icon={<PlusOutlined />} onClick={() => setEditor({ parentId: selected.id })}>新增子部门</Button></Auth>
    <Auth code="sys:dept:edit"><Button icon={<EditOutlined />} onClick={() => setEditor({ dept: selected })}>编辑</Button></Auth>
    <Auth code="sys:dept:delete"><Popconfirm title={`确认删除部门“${selected.name}”？`}
      description="存在子部门、成员或部门授权时，请先处理关联数据。"
      onConfirm={() => remove.mutateAsync(selected.id).catch(() => undefined)} okButtonProps={{ loading: remove.isPending }}>
      <Button danger type="text" icon={<DeleteOutlined />} disabled={remove.isPending}>删除</Button>
    </Popconfirm></Auth>
  </Space>;

  return <div className="page-stack dept-page">
    <PageHeader title="部门管理" description="维护组织架构、负责人和直属成员" extra={<Space>
      <Button icon={<ReloadOutlined />} onClick={() => void Promise.all([query.refetch(),
        queryClient.invalidateQueries({ queryKey: [queryKeys.users, 'dept-members'] })])}>刷新</Button>
      <Auth code="sys:dept:add"><Button type="primary" icon={<PlusOutlined />} onClick={() => setEditor({})}>新增部门</Button></Auth>
    </Space>} />
    <QueryState isPending={query.isPending} error={query.error} onRetry={() => void query.refetch()} />
    {query.isSuccess && <div className="dept-workspace">
      <Card className="data-card dept-organization" title={<div className="dept-section-heading">
        <span className="dept-section-icon"><ApartmentOutlined /></span><span>组织架构</span>
        <span className="dept-section-count">{departments.length}</span>
      </div>}>
        <div className="dept-tree-filters"><Input allowClear prefix={<SearchOutlined />} placeholder="搜索部门名称"
          value={keyword} onChange={event => setKeyword(event.target.value)} />
          <Select allowClear aria-label="部门状态" placeholder="全部状态" options={statusOptions} value={status} onChange={setStatus} /></div>
        {visible.length ? <Tree key={`${keyword}:${status ?? 'all'}`} blockNode showIcon showLine={{ showLeafIcon: false }} defaultExpandAll
          selectedKeys={selected ? [selected.id] : []} treeData={departmentNodes(buildTree(visible))}
          onSelect={keys => { if (keys[0] !== undefined) setSelectedId(Number(keys[0])); }} />
          : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="没有匹配的部门" />}
        <div className="dept-tree-caption"><TeamOutlined /><span>仅统计当前身份可查看的直属成员</span></div>
      </Card>
      <div className="dept-detail">
        {selected ? <>
          <Card className="data-card dept-overview">
            <div className="dept-overview-top">
              <div className="dept-overview-identity">
                <span className="dept-overview-icon"><ApartmentOutlined /></span>
                <div className="dept-overview-copy">
                  <div className="dept-breadcrumb">{departmentTrail(departments, selected)}</div>
                  <div className="dept-overview-title"><h2>{selected.name}</h2><StatusTag status={selected.status} /></div>
                </div>
              </div>
              {actions}
            </div>
            <div className="dept-overview-summary">
              <span className="dept-member-total"><TeamOutlined />
                {selected.members_visible ? <><strong>{selected.member_count}</strong><span>位直属成员</span></> : <span>无成员查看权限</span>}
              </span>
              <span className="dept-sort">排序 {selected.sort}</span>
            </div>
            <dl className="dept-contact-grid">
              <div><dt><UserOutlined />负责人</dt><dd>{selected.leader || '未设置'}</dd></div>
              <div><dt><PhoneOutlined />联系电话</dt><dd>{selected.phone || '—'}</dd></div>
              <div><dt><MailOutlined />邮箱</dt><dd>{selected.email || '—'}</dd></div>
            </dl>
          </Card>
          <DeptMembers key={selected.id} dept={selected} canReadUsers={canReadUsers && selected.members_visible} />
        </> : <Card className="data-card"><Empty description="选择部门查看详情与成员" /></Card>}
      </div>
    </div>}
    {editor && <DeptForm open dept={editor.dept} initialParentId={editor.parentId} departments={departments}
      canReadUsers={canReadUsers} departmentsPending={query.isPending} departmentsError={query.error}
      onRetryDepartments={() => void query.refetch()} onCancel={() => setEditor(null)} onSaved={() => setEditor(null)} />}
  </div>;
}
