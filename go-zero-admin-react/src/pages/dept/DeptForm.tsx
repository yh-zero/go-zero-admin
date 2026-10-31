import { Alert, App, Form, Input, InputNumber, Modal, Select, TreeSelect } from 'antd';
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

export default function DeptForm({ open, dept, departments, departmentsPending,
  departmentsError, onRetryDepartments, onCancel, onSaved }: DeptFormProps) {
  const [form] = Form.useForm<CreateDeptReq>();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const save = useMutation({
    mutationFn: (values: CreateDeptReq) => dept
      ? updateDept(dept.id, { ...values, parent_id: dept.parent_id })
      : createDept(values),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['depts'] });
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
        {dept && <Alert className="field-full" type="info" showIcon title="上级部门暂不支持调整。" />}
        <Form.Item name="parent_id" label="上级部门" rules={[{ required: true }]}>
          <TreeSelect treeData={[{ id: 0, name: '顶级部门', children: buildTree(departments) }]}
            fieldNames={{ label: 'name', value: 'id', children: 'children' }}
            disabled={Boolean(dept)} treeDefaultExpandAll showSearch treeNodeFilterProp="name" />
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
