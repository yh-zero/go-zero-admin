import { Form, Input, InputNumber, Select } from 'antd';
import { dataScopeOptions } from './roleOptions';

interface Props { editing: boolean }

export function RoleFormFields({ editing }: Props) {
  return <>
    <Form.Item name="name" label="角色名称" rules={[{ required: true, whitespace: true }, { max: 64 }]}><Input /></Form.Item>
    <Form.Item name="code" label="角色编码" extra={editing ? '角色编码作为权限标识，创建后保持不变。' : undefined}
      rules={[{ required: true, whitespace: true }, { pattern: /^[a-zA-Z][a-zA-Z0-9_:-]*$/, message: '请以字母开头，仅使用字母、数字、下划线、冒号或短横线' }, { max: 64 }]}>
      <Input disabled={editing} />
    </Form.Item>
    <Form.Item name="description" label="说明" rules={[{ max: 512 }]}><Input.TextArea rows={3} /></Form.Item>
    <Form.Item name="data_scope" label="数据范围" rules={[{ required: true }]}><Select options={dataScopeOptions} /></Form.Item>
    <Form.Item name="sort" label="排序" rules={[{ required: true }]}><InputNumber precision={0} min={0} max={2147483647} /></Form.Item>
    <Form.Item name="status" label="状态" rules={[{ required: true }]}><Select options={[
      { value: 1, label: '启用' }, { value: 2, label: '停用' },
    ]} /></Form.Item>
  </>;
}
