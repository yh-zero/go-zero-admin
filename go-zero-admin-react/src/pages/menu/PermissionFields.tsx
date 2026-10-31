import { Col, Form, Input, InputNumber, Row, Select, TreeSelect } from 'antd';
import type { MenuInfo } from '../../api/types';
import { buildTree } from '../../utils/tree';
import { statusOptions } from '../../utils/formOptions';

interface Props { resources: MenuInfo[]; lockParent: boolean }

export function PermissionFields({ resources, lockParent }: Props) {
  const navigation = resources.filter(item => item.type === 1 || item.type === 2);
  const options = navigation.map(item => ({ ...item, selectable: item.type === 2 }));
  return <Row gutter={16}>
    <Col span={24}><Form.Item name="parent_id" label="所属页面" extra={lockParent ? '操作权限将添加到当前页面。' : '可重新选择所属页面，目录不作为操作权限的归属。'}
      rules={[{ required: true, message: '请选择所属页面' }, { validator: (_, value) => resources.some(item => item.type === 2 && item.id === value)
        ? Promise.resolve() : Promise.reject(new Error('请选择有效的页面')) }]}>
      <TreeSelect treeData={buildTree(options)} fieldNames={{ label: 'title', value: 'id', children: 'children' }}
        treeDefaultExpandAll showSearch treeNodeFilterProp="title" disabled={lockParent} /></Form.Item></Col>
    <Col span={24}><Form.Item name="title" label="权限名称" rules={[{ required: true, whitespace: true, message: '请输入权限名称' }]}>
      <Input maxLength={64} placeholder="例如：新增用户" /></Form.Item></Col>
    <Col span={24}><Form.Item name="perm_code" label="权限码" extra="填写对应操作使用的权限码，例如 sys:user:add。"
      rules={[{ required: true, whitespace: true, message: '请输入权限码' }, { pattern: /^\S+$/, message: '权限码不能包含空格' }]}>
      <Input maxLength={128} placeholder="sys:user:add" /></Form.Item></Col>
    <Col xs={24} sm={12}><Form.Item name="sort" label="排序"><InputNumber min={0} max={2147483647} precision={0} /></Form.Item></Col>
    <Col xs={24} sm={12}><Form.Item name="status" label="状态"><Select options={statusOptions} /></Form.Item></Col>
  </Row>;
}
