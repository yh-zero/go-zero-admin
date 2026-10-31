import { useState } from 'react';
import { App, Button, Card, Form, Popconfirm, Select, Space, Table } from 'antd';
import type { TableColumnsType } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deleteDept, fetchDeptTree } from '../../api/dept';
import type { DeptInfo, GetDeptTreeReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { StatusTag } from '../../components/StatusTag';
import { buildTree } from '../../utils/tree';
import { formatTime } from '../../utils/format';
import { statusOptions } from '../../utils/formOptions';
import DeptForm from './DeptForm';

export default function DeptList() {
  const [params, setParams] = useState<GetDeptTreeReq>({});
  const [editing, setEditing] = useState<DeptInfo>();
  const [formOpen, setFormOpen] = useState(false);
  const [filterForm] = Form.useForm<GetDeptTreeReq>();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ['depts', params], queryFn: () => fetchDeptTree(params) });
  const allDepts = useQuery({ queryKey: ['depts', {}], queryFn: () => fetchDeptTree(), enabled: formOpen });
  const remove = useMutation({
    mutationFn: deleteDept,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['depts'] });
      message.success('部门已删除');
    },
  });
  const onCreate = () => { setEditing(undefined); setFormOpen(true); };
  const onEdit = (dept: DeptInfo) => { setEditing(dept); setFormOpen(true); };
  const columns: TableColumnsType<DeptInfo> = [
    { title: '部门名称', dataIndex: 'name', key: 'name', render: (value: string) => <span className="table-primary">{value}</span> },
    { title: '负责人', dataIndex: 'leader', key: 'leader' },
    { title: '联系电话', dataIndex: 'phone', key: 'phone' },
    { title: '邮箱', dataIndex: 'email', key: 'email' },
    { title: '排序', dataIndex: 'sort', key: 'sort', width: 80 },
    { title: '状态', dataIndex: 'status', key: 'status', render: (status: number) => <StatusTag status={status} /> },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: formatTime },
    { title: '操作', key: 'actions', width: 150, render: (_, dept) => (
      <Space>
        <Auth code="sys:dept:edit"><Button type="link" onClick={() => onEdit(dept)}>编辑</Button></Auth>
        <Auth code="sys:dept:delete">
          <Popconfirm title={`确认删除部门“${dept.name}”？`} description="存在子部门或用户的部门无法删除。"
            onConfirm={() => remove.mutate(dept.id)} okButtonProps={{ loading: remove.isPending }}>
            <Button type="link" danger loading={remove.isPending && remove.variables === dept.id}>删除</Button>
          </Popconfirm>
        </Auth>
      </Space>
    ) },
  ];

  return (
    <div className="page-stack">
      <PageHeader title="部门管理" description="维护组织层级与部门联系信息" extra={
        <Auth code="sys:dept:add"><Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>新增部门</Button></Auth>
      } />
      <Card className="filter-card">
        <Form name="dept-filter" className="filter-form" form={filterForm} layout="inline" onFinish={setParams}>
          <Form.Item name="status" label="状态"><Select allowClear placeholder="全部状态" className="filter-select"
            options={statusOptions} /></Form.Item>
          <Form.Item><Space><Button type="primary" htmlType="submit">查询</Button>
            <Button onClick={() => { filterForm.resetFields(); setParams({}); }}>重置</Button>
            <Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>刷新</Button>
          </Space></Form.Item>
        </Form>
      </Card>
      <Card className="data-card" title="部门结构">
        <QueryState isPending={query.isPending} error={query.error} onRetry={() => void query.refetch()} />
        {!query.isPending && !query.error && <Table rowKey="id" columns={columns}
          dataSource={buildTree(query.data?.list ?? [])} pagination={false} scroll={{ x: 950 }} />}
      </Card>
      <DeptForm open={formOpen} dept={editing} departments={allDepts.data?.list ?? []}
        departmentsPending={allDepts.isPending} departmentsError={allDepts.error}
        onRetryDepartments={() => void allDepts.refetch()}
        onCancel={() => setFormOpen(false)} onSaved={() => setFormOpen(false)} />
    </div>
  );
}
