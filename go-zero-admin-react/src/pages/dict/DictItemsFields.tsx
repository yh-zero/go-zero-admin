import { Button, Form, Input, InputNumber, Typography } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';

export function DictItemsFields() {
  return <Form.List name="items">{(fields, { add, remove }) => <>
    <Typography.Title level={5}>字典项</Typography.Title>
    {fields.map((field) => <div key={field.key} className="dict-item-row">
      <Form.Item name={[field.name, 'label']} label="显示文字" rules={[{ required: true, whitespace: true, message: '请输入显示文字' }]}><Input placeholder="例如 启用" /></Form.Item>
      <Form.Item name={[field.name, 'value']} label="值" rules={[{ required: true, whitespace: true, message: '请输入字典值' }]}><Input placeholder="例如 1" /></Form.Item>
      <Form.Item name={[field.name, 'sort']} label="排序"><InputNumber precision={0} /></Form.Item>
      <Button aria-label="删除字典项" danger icon={<DeleteOutlined />} onClick={() => remove(field.name)} />
    </div>)}
    <Form.Item><Button type="dashed" block icon={<PlusOutlined />} onClick={() => add({ label: '', value: '', sort: fields.length })}>添加字典项</Button></Form.Item>
  </>}</Form.List>;
}
