import { Form, Input, Select } from 'antd';
import type { DeptInfo } from '../../api/types';

interface Props { editing: boolean; depts: DeptInfo[]; currentDeptId?: number; currentDeptName?: string }

export function UserFormFields({ editing, depts, currentDeptId = 0, currentDeptName }: Props) {
  const options = depts.map((dept) => ({ label: dept.name, value: dept.id }));
  if (currentDeptId > 0 && !depts.some((dept) => dept.id === currentDeptId)) {
    options.push({ label: currentDeptName || `部门 ${currentDeptId}`, value: currentDeptId });
  }
  if (!editing || currentDeptId === 0) options.unshift({ label: '未分配', value: 0 });
  return <>
    <Form.Item name="username" label="用户名" rules={[{ required: true, whitespace: true }, { max: 64 }]}>
      <Input disabled={editing} autoComplete="off" />
    </Form.Item>
    {!editing && <Form.Item name="password" label="初始密码" rules={[
      { required: true }, { min: 8, message: '请至少输入 8 个字符' },
      { validator: (_, value: string) => !value || new TextEncoder().encode(value).length <= 72
        ? Promise.resolve() : Promise.reject(new Error('密码最多 72 字节')) },
    ]}><Input.Password autoComplete="new-password" /></Form.Item>}
    <Form.Item name="nickname" label="昵称" rules={[{ max: 64 }]}><Input /></Form.Item>
    <Form.Item name="real_name" label="真实姓名" rules={[{ max: 64 }]}><Input /></Form.Item>
    <Form.Item name="email" label="邮箱" rules={[{ type: 'email' }, { max: 128 }]}><Input /></Form.Item>
    <Form.Item name="phone" label="手机号" rules={[{ max: 20 }]}><Input maxLength={20} /></Form.Item>
    <Form.Item name="dept_id" label="部门"><Select options={options} showSearch optionFilterProp="label" /></Form.Item>
    <Form.Item name="gender" label="性别"><Select options={[
      { value: 0, label: '未知' }, { value: 1, label: '男' }, { value: 2, label: '女' },
    ]} /></Form.Item>
    <Form.Item name="status" label="状态" rules={[{ required: true }]}><Select options={[
      { value: 1, label: '启用' }, { value: 2, label: '停用' },
    ]} /></Form.Item>
    <Form.Item name="remark" label="备注" rules={[{ max: 512 }]}><Input.TextArea rows={3} /></Form.Item>
    <Form.Item name="avatar_file_id" hidden><Input type="number" /></Form.Item>
  </>;
}
