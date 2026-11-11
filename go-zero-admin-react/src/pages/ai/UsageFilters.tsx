import { useState } from 'react';
import { Button, Form, Input, InputNumber, Select, Space } from 'antd';
import { DownOutlined, UpOutlined } from '@ant-design/icons';
import type { AiUsageAdminQuery } from '../../api/types';
import { usageDateRange, usageDateRangeError, usageFilterDefaults, usageFilterParams } from './usageState';
import styles from './usage.module.css';

interface Props { admin: boolean; params: AiUsageAdminQuery; onChange: (value: AiUsageAdminQuery) => void }

const sources = { chat: '对话', analysis: '流程分析', kb_search: '独立检索', kb_ingest: '文档摄取', evaluation: '质量验收' };
const statuses = { reserved: '预留中', running: '请求中', succeeded: '已完成', failed: '请求失败', cancelled: '已取消', unknown: '待核对', not_sent: '未发送' };
const usageStates = { complete: '用量完整', partial: '部分用量', unknown: '未取得用量', invalid: '用量无效' };

export function UsageFilters({ admin, params, onChange }: Props) {
  const [form] = Form.useForm<AiUsageAdminQuery>();
  const [expanded, setExpanded] = useState(false);
  const advancedCount = [params.operation, params.status, params.usage_state, params.provider, params.model, admin ? params.user_id : undefined]
    .filter(value => value !== undefined && value !== null && value !== '').length;
  const validateRange = () => {
    const error = usageDateRangeError(form.getFieldValue('from_day'), form.getFieldValue('to_day'));
    return error ? Promise.reject(new Error(error)) : Promise.resolve();
  };
  function reset() {
    const defaults = usageFilterDefaults();
    form.resetFields();
    form.setFieldsValue({ from_day: defaults.from_day, to_day: defaults.to_day, source: undefined, operation: undefined, status: undefined,
      usage_state: undefined, provider: undefined, model: undefined, user_id: undefined });
    onChange(defaults);
  }
  function quickRange(days: 1 | 7 | 30) {
    form.setFieldsValue(usageDateRange(days));
    form.submit();
  }

  return <Form form={form} className={styles.filterForm} layout="vertical" initialValues={params}
    onFinish={values => onChange(usageFilterParams(values, admin))}>
    <div className={styles.filterPrimary}>
      <div className={styles.dateRange}>
        <Form.Item name="from_day" label="开始日期" rules={[{ required: true, message: '请选择开始日期' }]}><Input type="date" aria-label="开始日期" /></Form.Item>
        <Form.Item name="to_day" label="结束日期" dependencies={['from_day']} rules={[{ validator: validateRange }]}><Input type="date" aria-label="结束日期" /></Form.Item>
      </div>
      <Form.Item name="source" label="来源"><Select allowClear className={styles.select} placeholder="全部来源" aria-label="用量来源"
        options={Object.entries(sources).map(([value, label]) => ({ value, label }))} /></Form.Item>
      <div className={styles.filterActions}><Button type="primary" htmlType="submit" aria-label="查询用量">查询</Button><Button onClick={reset} aria-label="重置筛选">重置</Button>
        <Button type="link" aria-expanded={expanded} onClick={() => setExpanded(value => !value)} icon={expanded ? <UpOutlined /> : <DownOutlined />}>
          {expanded ? '收起筛选' : '更多筛选'}{advancedCount > 0 ? `（${advancedCount}）` : ''}
        </Button>
      </div>
    </div>
    <div className={styles.quickRanges}><Space wrap size="small"><Button size="small" type="text" onClick={() => quickRange(1)}>今天</Button>
      <Button size="small" type="text" onClick={() => quickRange(7)}>最近7天</Button><Button size="small" type="text" onClick={() => quickRange(30)}>最近30天</Button>
      <span>上海时区 · 最多31日</span></Space></div>
    <div className={styles.advancedFilters} hidden={!expanded}>
      <Form.Item name="operation" label="调用类型"><Select allowClear placeholder="全部类型" aria-label="调用类型" options={[{ value: 'chat', label: '对话' }, { value: 'embedding', label: '向量化' }]} /></Form.Item>
      <Form.Item name="status" label="请求状态"><Select allowClear placeholder="全部状态" aria-label="请求状态" options={Object.entries(statuses).map(([value, label]) => ({ value, label }))} /></Form.Item>
      <Form.Item name="usage_state" label="计量状态"><Select allowClear placeholder="全部计量状态" aria-label="计量状态" options={Object.entries(usageStates).map(([value, label]) => ({ value, label }))} /></Form.Item>
      <Form.Item name="provider" label="供应商"><Input maxLength={32} aria-label="供应商" /></Form.Item>
      <Form.Item name="model" label="模型"><Input maxLength={128} aria-label="模型" /></Form.Item>
      {admin && <Form.Item name="user_id" label="后台用户 ID"><InputNumber min={0} precision={0} aria-label="后台用户 ID" /></Form.Item>}
    </div>
  </Form>;
}
