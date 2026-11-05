import { useState } from 'react';
import type { ReactNode } from 'react';
import { App, Button, Empty, Form, Input, InputNumber, Modal, Select, Space, TreeSelect, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createDept, updateDept } from '../../api/dept';
import { fetchUsers } from '../../api/user';
import type { CreateDeptReq, DeptInfo, UpdateDeptReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { buildTree } from '../../utils/tree';
import { statusOptions } from '../../utils/formOptions';
import { queryKeys } from '../../utils/queryKeys';

interface DeptFormProps {
  open: boolean;
  dept?: DeptInfo;
  initialParentId?: number;
  canReadUsers: boolean;
  departments: DeptInfo[];
  departmentsPending: boolean;
  departmentsError: Error | null;
  onRetryDepartments: () => void;
  onCancel: () => void;
  onSaved: () => void;
}
interface LeaderChoice { value: number; label: ReactNode }
type DeptFormValues = Omit<CreateDeptReq, 'leader' | 'leader_user_id'> & { leader_selection?: LeaderChoice };

function allowedParents(departments: DeptInfo[], currentId?: number) {
  const excluded = new Set<number>(currentId === undefined ? [] : [currentId]);
  const queue = currentId === undefined ? [] : [currentId];
  while (queue.length) {
    const parent = queue.shift();
    for (const item of departments) {
      if (item.parent_id === parent && !excluded.has(item.id)) { excluded.add(item.id); queue.push(item.id); }
    }
  }
  return departments.filter(item => !excluded.has(item.id));
}

interface LeaderSelectProps {
  id?: string;
  enabled: boolean;
  disabled?: boolean;
  value?: LeaderChoice;
  onChange?: (value?: LeaderChoice) => void;
}

function LeaderSelect({ id, enabled, disabled, value, onChange }: LeaderSelectProps) {
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const params = { username: search, status: 1, page, page_size: 20 };
  const users = useQuery({ queryKey: [queryKeys.users, 'dept-leader-options', params], queryFn: () => fetchUsers(params), enabled });
  return <Select id={id} labelInValue allowClear showSearch={{ filterOption: false, onSearch: keyword => { setSearch(keyword); setPage(1); } }}
    value={value} onChange={onChange} disabled={disabled || !enabled ? true : undefined} loading={users.isFetching}
    placeholder={enabled ? '输入用户名搜索并选择负责人' : '需要用户查询权限'}
    options={users.isError ? [] : users.data?.list.map(user => ({ value: user.id, label: user.id === value?.value ? value.label
      : `${user.real_name || user.nickname || user.username}（${user.username}）` }))}
    notFoundContent={users.isError ? <QueryState error={users.error} onRetry={() => void users.refetch()} />
      : users.isPending ? '正在加载…' : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="没有可选用户" />}
    popupRender={menu => <>{menu}<div className="dept-leader-paging" onMouseDown={event => event.preventDefault()}>
      <Typography.Text type="secondary">第 {page} 页</Typography.Text><Space size="small">
        <Button size="small" disabled={page <= 1 || users.isFetching} onClick={() => setPage(current => current - 1)}>上一页</Button>
        <Button size="small" disabled={!users.isSuccess || page * 20 >= users.data.total || users.isFetching}
          onClick={() => setPage(current => current + 1)}>下一页</Button>
      </Space></div></>} />;
}

export default function DeptForm({ open, dept, initialParentId = 0, canReadUsers, departments, departmentsPending,
  departmentsError, onRetryDepartments, onCancel, onSaved }: DeptFormProps) {
  const [form] = Form.useForm<DeptFormValues>();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const save = useMutation({
    mutationFn: (values: DeptFormValues) => {
      const { leader_selection, ...fields } = values;
      if (!dept) return createDept({ ...fields, leader_user_id: leader_selection?.value ?? 0 });
      const changes: UpdateDeptReq = Object.fromEntries(Object.entries(fields)
        .filter(([key]) => form.isFieldTouched(key as keyof DeptFormValues)));
      if (form.isFieldTouched('leader_selection')) changes.leader_user_id = leader_selection?.value ?? 0;
      return updateDept(dept.id, changes);
    },
    onSuccess: async () => {
      await Promise.all(['depts', 'my-depts', 'profile', 'users'].map(key => queryClient.invalidateQueries({ queryKey: [key] })));
      message.success(dept ? '部门已更新' : '部门已创建');
      onSaved();
    },
  });
  const initialValues: Partial<DeptFormValues> = dept ? { ...dept,
    leader_selection: dept.leader_user_id ? { value: dept.leader_user_id, label: dept.leader || `用户 #${dept.leader_user_id}` }
      : dept.leader ? { value: 0, label: `原负责人：${dept.leader}` } : undefined,
  } : { parent_id: initialParentId, sort: 0, status: 1 };
  useFormDraft(form, { open, entityKey: dept?.id ?? `new:${initialParentId}`, values: initialValues });

  return <Modal title={dept ? '编辑部门' : initialParentId ? '新增子部门' : '新增部门'} width={680} open={open}
    onCancel={() => { if (!save.isPending) onCancel(); }} onOk={() => form.submit()} confirmLoading={save.isPending}
    okButtonProps={{ disabled: departmentsPending || Boolean(departmentsError) }} cancelButtonProps={{ disabled: save.isPending }}
    closable={!save.isPending} mask={{ closable: !save.isPending }} keyboard={!save.isPending} destroyOnHidden>
    <QueryState isPending={departmentsPending} error={departmentsError} onRetry={onRetryDepartments} />
    <Form name="dept-editor" className="editor-grid" form={form} layout="vertical"
      disabled={save.isPending || departmentsPending || Boolean(departmentsError)} onFinish={values => save.mutate(values)}>
      <Form.Item name="parent_id" label="上级部门" rules={[{ required: true }]}
        extra={dept ? '移动后子部门随之调整，不能移到自身或下级部门。' : undefined}>
        <TreeSelect treeData={[{ id: 0, name: '顶级部门', children: buildTree(allowedParents(departments, dept?.id)) }]}
          fieldNames={{ label: 'name', value: 'id', children: 'children' }} treeDefaultExpandAll showSearch treeNodeFilterProp="name" />
      </Form.Item>
      <Form.Item name="name" label="部门名称" rules={[{ required: true, whitespace: true, message: '请输入部门名称' }]}><Input maxLength={64} /></Form.Item>
      <Form.Item name="leader_selection" label="负责人" extra={!canReadUsers ? '无用户查询权限，负责人保持不变。'
        : dept?.leader && !dept.leader_user_id ? `原负责人：${dept.leader}；选择用户后关联账号，不自动授权。` : '仅关联负责人账号，不自动授予管理权限。'}>
        <LeaderSelect enabled={open && canReadUsers} />
      </Form.Item>
      <Form.Item name="phone" label="联系电话"><Input maxLength={20} /></Form.Item>
      <Form.Item name="email" label="邮箱" rules={[{ type: 'email', message: '请输入有效邮箱' }]}><Input maxLength={128} /></Form.Item>
      <Form.Item name="sort" label="排序"><InputNumber min={0} precision={0} /></Form.Item>
      <Form.Item name="status" label="状态" rules={[{ required: true }]} extra="存在子部门、成员或部门授权时，需先处理关联数据才能停用。">
        <Select options={statusOptions} />
      </Form.Item>
    </Form>
  </Modal>;
}
