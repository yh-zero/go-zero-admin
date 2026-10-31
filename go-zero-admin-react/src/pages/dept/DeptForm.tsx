import { App, Form, Input, InputNumber, Modal, Select, TreeSelect } from 'antd';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createDept, updateDept } from '../../api/dept';
import type { CreateDeptReq, DeptInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { buildTree } from '../../utils/tree';
import { statusOptions } from '../../utils/formOptions';

interface DeptFormProps {
  open: boolean;
  dept?: DeptInfo;
  departments: DeptInfo[];
  departmentsPending: boolean;
  departmentsError: Error | null;
  onRetryDepartments: () => void;
  onCancel: () => void;
  onSaved: () => void;
}
const defaults = { parent_id: 0, sort: 0, status: 1 };

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

export default function DeptForm({ open, dept, departments, departmentsPending,
  departmentsError, onRetryDepartments, onCancel, onSaved }: DeptFormProps) {
  const [form] = Form.useForm<CreateDeptReq>();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const save = useMutation({
    mutationFn: (values: CreateDeptReq) => dept
      ? updateDept(dept.id, Object.fromEntries(Object.entries(values).filter(([key]) => form.isFieldTouched(key as keyof CreateDeptReq))))
      : createDept(values),
    onSuccess: async () => {
      await Promise.all(['depts', 'my-depts', 'profile'].map(key => queryClient.invalidateQueries({ queryKey: [key] })));
      message.success(dept ? '部门已更新' : '部门已创建');
      onSaved();
    },
  });

  useFormDraft(form, { open, entityKey: dept?.id ?? 'new', values: dept ?? defaults });

  return (
    <Modal title={dept ? '编辑部门' : '新增部门'} width={680} open={open} onCancel={() => { if (!save.isPending) onCancel(); }}
      onOk={() => form.submit()} confirmLoading={save.isPending}
      okButtonProps={{ disabled: departmentsPending || Boolean(departmentsError) }}
      cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending}
      mask={{ closable: !save.isPending }} keyboard={!save.isPending} destroyOnHidden>
      <QueryState isPending={departmentsPending} error={departmentsError} onRetry={onRetryDepartments} />
      <Form name="dept-editor" className="editor-grid" form={form} layout="vertical" disabled={save.isPending || departmentsPending || Boolean(departmentsError)}
        onFinish={(values) => save.mutate(values)}>
        <Form.Item name="parent_id" label="上级部门" rules={[{ required: true }]} extra={dept ? '移动后，子部门会随之调整；不能移到自身或下级部门。' : undefined}>
          <TreeSelect treeData={[{ id: 0, name: '顶级部门', children: buildTree(allowedParents(departments, dept?.id)) }]}
            fieldNames={{ label: 'name', value: 'id', children: 'children' }}
            treeDefaultExpandAll showSearch treeNodeFilterProp="name" />
        </Form.Item>
        <Form.Item name="name" label="部门名称" rules={[{ required: true, whitespace: true, message: '请输入部门名称' }]}>
          <Input maxLength={64} />
        </Form.Item>
        <Form.Item name="leader" label="负责人"><Input maxLength={64} /></Form.Item>
        <Form.Item name="phone" label="联系电话"><Input maxLength={20} /></Form.Item>
        <Form.Item name="email" label="邮箱" rules={[{ type: 'email', message: '请输入有效邮箱' }]}>
          <Input maxLength={128} />
        </Form.Item>
        <Form.Item name="sort" label="排序"><InputNumber min={0} precision={0} /></Form.Item>
        <Form.Item name="status" label="状态" rules={[{ required: true }]}>
          <Select options={statusOptions} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
