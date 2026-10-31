import { Col, Form, Input, InputNumber, Row, Select, Switch, TreeSelect } from 'antd';
import type { CreateMenuReq, MenuInfo } from '../../api/types';
import { buildTree } from '../../utils/tree';
import { statusOptions } from '../../utils/formOptions';
import { getDescendantIds, getPagePermissions } from './menuResources';

interface Props { resources: MenuInfo[]; editingId: number | null }

export function MenuFields({ resources, editingId }: Props) {
  const form = Form.useFormInstance<CreateMenuReq>();
  const type = Form.useWatch('type', form);
  const excludedIds = editingId === null ? new Set<number>() : getDescendantIds(resources, editingId);
  const parents = resources.filter(item => (item.type === 1 || item.type === 2) && !excludedIds.has(item.id));
  const hasPermissions = editingId !== null && getPagePermissions(resources, editingId).length > 0;
  const parentTree = [{ id: 0, title: '根目录', children: buildTree(parents) }];
  return <Row gutter={16}>
    <Col xs={24} sm={12}><Form.Item name="title" label="菜单名称" rules={[{ required: true, whitespace: true, message: '请输入菜单名称' }]}>
      <Input maxLength={64} /></Form.Item></Col>
    <Col xs={24} sm={12}><Form.Item name="type" label="类型" rules={[{ required: true }]}
      extra={hasPermissions ? '页面包含操作权限，需先调整权限归属才能改为目录。' : undefined}>
      <Select options={[{ value: 1, label: '目录', disabled: hasPermissions }, { value: 2, label: '页面' }]} /></Form.Item></Col>
    <Col xs={24} sm={24}><Form.Item name="parent_id" label="上级菜单" rules={[{ required: true, message: '请选择上级菜单' },
      { validator: (_, value) => value === 0 || parents.some(item => item.id === value) ? Promise.resolve() : Promise.reject(new Error('请选择有效的上级菜单')) }]}>
      <TreeSelect treeData={parentTree} fieldNames={{ label: 'title', value: 'id', children: 'children' }} treeDefaultExpandAll showSearch treeNodeFilterProp="title" /></Form.Item></Col>
    <Col xs={24} sm={12}><Form.Item name="name" label="页面标识" extra="与已注册页面一致，如 user、role、menu"
      rules={[{ required: type === 2, whitespace: true, message: '请输入页面标识' }]}>
      <Input maxLength={64} /></Form.Item></Col>
    <Col xs={24} sm={12}><Form.Item name="path" label="访问路径" extra="例如 /sys/menu"
      rules={[{ required: type === 2, whitespace: true, message: '请输入访问路径' }]}>
      <Input maxLength={255} /></Form.Item></Col>
    <Col xs={24} sm={12}><Form.Item name="component" label="组件标识"><Input maxLength={255} /></Form.Item></Col>
    <Col xs={24} sm={12}><Form.Item name="icon" label="图标标识"><Input maxLength={64} /></Form.Item></Col>
    <Col xs={24} sm={8}><Form.Item name="sort" label="排序"><InputNumber precision={0} min={0} max={2147483647} /></Form.Item></Col>
    <Col xs={24} sm={8}><Form.Item name="status" label="状态"><Select options={statusOptions} /></Form.Item></Col>
    <Col xs={24} sm={8}><Form.Item name="visible" label="侧栏显示" valuePropName="checked"><Switch /></Form.Item></Col>
  </Row>;
}
