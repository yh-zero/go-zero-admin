import { Col, Form, Input, InputNumber, Row, Select, Switch } from 'antd';
import { menuTypeOptions } from './menuOptions';

export function MenuFields() {
  return <>
    <Row gutter={16}>
      <Col xs={24} sm={12}><Form.Item name="title" label="菜单标题" rules={[{ required: true, whitespace: true, message: '请输入菜单标题' }]}><Input maxLength={64} /></Form.Item></Col>
      <Col xs={24} sm={12}><Form.Item name="type" label="类型" rules={[{ required: true }]}><Select options={menuTypeOptions} /></Form.Item></Col>
      <Col xs={24} sm={12}><Form.Item name="parent_id" label="父菜单 ID" extra="0 表示根节点"><InputNumber min={0} precision={0} /></Form.Item></Col>
      <Col xs={24} sm={12}><Form.Item name="name" label="页面名称" extra="与页面注册名称一致，如 menu、dict、api"><Input maxLength={64} /></Form.Item></Col>
      <Col xs={24} sm={12}><Form.Item name="path" label="访问路径" extra="例如 /sys/menu；按钮可以留空"><Input maxLength={255} /></Form.Item></Col>
      <Col xs={24} sm={12}><Form.Item name="component" label="组件标识" extra="保留已有标识；页面按 name 注册"><Input maxLength={255} /></Form.Item></Col>
      <Col xs={24} sm={12}><Form.Item name="perm_code" label="权限码" extra="按钮权限，例如 sys:menu:add"><Input maxLength={128} /></Form.Item></Col>
      <Col xs={24} sm={12}><Form.Item name="icon" label="图标标识"><Input maxLength={64} /></Form.Item></Col>
      <Col xs={24} sm={8}><Form.Item name="sort" label="排序"><InputNumber precision={0} /></Form.Item></Col>
      <Col xs={24} sm={8}><Form.Item name="status" label="状态"><Select options={[{ value: 1, label: '启用' }, { value: 2, label: '停用' }]} /></Form.Item></Col>
      <Col xs={24} sm={8}><Form.Item name="visible" label="侧栏显示" valuePropName="checked"><Switch /></Form.Item></Col>
    </Row>
  </>;
}
