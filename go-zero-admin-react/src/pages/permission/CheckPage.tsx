import { useState } from 'react';
import { Alert, Button, Card, Descriptions, Form, Input, InputNumber, Select, Space, Tag, Typography } from 'antd';
import { useMutation } from '@tanstack/react-query';
import { checkPermission } from '../../api/permission';
import type { PermissionCheckReq, PermissionCheckResp } from '../../api/types';
import { PageHeader } from '../../components/PageHeader';
import { formatTime } from '../../utils/format';

export default function PermissionCheckPage() {
  const [form] = Form.useForm<PermissionCheckReq>();
  const [result, setResult] = useState<PermissionCheckResp | null>(null);
  const check = useMutation({ mutationFn: checkPermission, onSuccess: setResult });
  return <div className="page-stack"><PageHeader title="权限诊断" description="按当前策略快照模拟用户、部门、接口与方法的授权结果。" />
    <Card className="filter-card" title="模拟请求"><Form form={form} layout="inline" initialValues={{ user_id: 1, dept_id: 1, method: 'GET', path: '/v1/sys/user/list' }} onFinish={values => check.mutate(values)}>
      <Form.Item name="user_id" label="用户 ID" rules={[{ required: true, message: '请输入用户 ID' }]}><InputNumber min={1} precision={0} /></Form.Item>
      <Form.Item name="dept_id" label="部门 ID"><InputNumber min={0} precision={0} /></Form.Item>
      <Form.Item name="method" label="方法" rules={[{ required: true }]}><Select className="filter-select" options={['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].map(value => ({ value, label: value }))} /></Form.Item>
      <Form.Item name="path" label="接口路径" rules={[{ required: true, message: '请输入接口路径' }]}><Input maxLength={255} placeholder="/v1/sys/user/list" /></Form.Item>
      <Form.Item><Button type="primary" htmlType="submit" loading={check.isPending}>检查权限</Button></Form.Item>
    </Form></Card>
    {check.isError && <Alert type="error" showIcon title="权限诊断失败" description={check.error.message} />}
    {result && <Card title={result.allowed ? '允许访问' : '拒绝访问'}>
      <Alert type={result.allowed ? 'success' : 'warning'} showIcon title={result.allowed ? '当前策略允许该请求' : '当前策略拒绝该请求'} />
      <Descriptions column={1} style={{ marginTop: 20 }} items={[
        { key: 'role', label: '命中角色', children: result.role || '未命中' },
        { key: 'domain', label: '策略域', children: result.domain || '未命中' },
        { key: 'scope', label: '数据范围', children: result.data_scopes.length ? <Space wrap>{result.data_scopes.map(scope => <Tag key={scope}>{scope}</Tag>)}</Space> : '不适用' },
        { key: 'revision', label: '策略版本', children: result.policy_revision },
        { key: 'generated', label: '快照生成时间', children: formatTime(result.generated_at) },
        { key: 'loaded', label: 'API 加载时间', children: formatTime(result.loaded_at) },
      ]} />
      <Typography.Paragraph type="secondary" style={{ marginTop: 16 }}>诊断结果只反映当前 API 进程加载的策略快照；版本或加载时间变化后请重新检查。</Typography.Paragraph>
    </Card>}
  </div>;
}
