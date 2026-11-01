import { useDeferredValue, useEffect, useRef, useState } from 'react';
import { App, Form, Input, InputNumber, Modal, Pagination, Select, Spin, Typography } from 'antd';
import { CalendarOutlined, CrownOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchAppMembershipPlans, fetchAppUsers, grantAppMembership } from '../../api/app';
import type { GrantAppMembershipReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { invalidateAppData } from '../../utils/queryInvalidation';

interface UserChoice { value: number; label: string }
interface UserSelectProps { id?: string; value?: UserChoice; onChange?: (value?: UserChoice) => void; disabled?: boolean }
interface Props { initialUserId?: number; onClose: () => void }
type GrantFormValues = Omit<GrantAppMembershipReq, 'request_id' | 'user_id'> & { user?: UserChoice };

function AppUserSelect({ id, value, onChange, disabled }: UserSelectProps) {
  const [account, setAccount] = useState('');
  const [page, setPage] = useState(1);
  const search = useDeferredValue(account);
  const searchValid = new TextEncoder().encode(search).length <= 128;
  const users = useQuery({ queryKey: ['app-user-options', search, page],
    queryFn: () => fetchAppUsers({ account: search || undefined, page, page_size: 20 }), enabled: searchValid });
  const options = (users.data?.list ?? []).map(user => ({ value: user.id,
    label: `${user.nickname ? user.nickname + ' · ' : ''}${user.account}（#${user.id}）` }));
  const selected = options.find(option => option.value === value?.value);
  useEffect(() => {
    if (selected && value && selected.label !== value.label) onChange?.(selected);
  }, [selected, value, onChange]);
  return <>
    <Select<UserChoice> id={id} labelInValue allowClear value={value} onChange={onChange} disabled={disabled}
      placeholder="搜索并选择 APP 用户" options={options} loading={users.isFetching}
      showSearch={{ filterOption: false, onSearch: text => { setAccount(text.trim()); setPage(1); } }}
      notFoundContent={!searchValid ? '账号搜索最多 128 字节' : users.isFetching ? <Spin size="small" /> : '没有匹配用户'}
      popupRender={menu => <>{menu}<div onMouseDown={event => event.preventDefault()}>
        <Pagination size="small" current={page} pageSize={20} total={users.data?.total ?? 0} hideOnSinglePage
          showSizeChanger={false} onChange={setPage} />
      </div></>} />
    {users.isError && <QueryState error={users.error} onRetry={() => users.refetch()} />}
  </>;
}

export function AppMembershipGrantDialog({ initialUserId, onClose }: Props) {
  const [form] = Form.useForm<GrantFormValues>();
  const client = useQueryClient();
  const { message } = App.useApp();
  const plans = useQuery({ queryKey: ['app-membership-plans'], queryFn: fetchAppMembershipPlans });
  const plansUnavailable = !plans.isSuccess || plans.data.length === 0;
  const submission = useRef<{ fingerprint: string; requestId: string } | null>(null);
  const submitting = useRef(false);
  const save = useMutation({ mutationFn: grantAppMembership, onSuccess: async () => {
    await invalidateAppData(client);
    void message.success('会员已开通');
    onClose();
  }, onSettled: () => { submitting.current = false; } });
  const submit = (values: GrantFormValues) => {
    if (submitting.current || !values.user || plansUnavailable) return;
    const business = { user_id: values.user.value, plan_code: values.plan_code.trim(),
      benefits: values.benefits ?? '', duration_days: values.duration_days };
    const fingerprint = JSON.stringify(business);
    if (submission.current?.fingerprint !== fingerprint) {
      submission.current = { fingerprint, requestId: crypto.randomUUID() };
    }
    submitting.current = true;
    save.mutate({ ...business, request_id: submission.current.requestId });
  };
  return <Modal open title="开通会员" className="app-form-dialog" width={600} onCancel={onClose} onOk={() => form.submit()}
    confirmLoading={save.isPending} okText="确认开通" okButtonProps={{ disabled: plansUnavailable }} cancelButtonProps={{ disabled: save.isPending }}
    closable={!save.isPending} keyboard={!save.isPending} mask={{ closable: !save.isPending }}>
    <div className="app-form-intro"><span className="app-form-intro-icon" aria-hidden="true"><CrownOutlined /></span>
      <div className="app-form-intro-copy"><strong>配置会员权益</strong><p>为指定 APP 用户选择套餐，设置本次会员的有效期与权益。</p></div>
    </div>
    <QueryState isPending={plans.isPending} error={plans.error} onRetry={() => plans.refetch()} />
    {plans.isSuccess && plans.data.length === 0 && <Typography.Paragraph type="secondary">暂无启用的会员套餐，请联系管理员配置。</Typography.Paragraph>}
    <Form<GrantFormValues> form={form} name="app-membership-grant" layout="vertical" disabled={save.isPending || plansUnavailable}
      initialValues={{ duration_days: 30, benefits: '', user: initialUserId ? { value: initialUserId, label: `用户 #${initialUserId}` } : undefined }}
      onFinish={submit}>
      <div className="app-section-title"><span className="app-section-icon" aria-hidden="true"><UserOutlined /></span><span>开通对象</span></div>
      <Form.Item name="user" label="APP 用户" rules={[{ required: true, message: '请选择 APP 用户' },
        { validator: (_, value?: UserChoice) => !value || Number.isSafeInteger(value.value) && value.value > 0
          ? Promise.resolve() : Promise.reject(new Error('请选择有效用户')) }]}>
        <AppUserSelect disabled={save.isPending} />
      </Form.Item>
      <Form.Item name="plan_code" label="会员套餐" extra="从已启用的套餐中选择，有效天数及本次权益可单独填写。"
        rules={[{ required: true, whitespace: true, message: '请填写套餐标识' },
          { validator: (_, value?: string) => !value || new TextEncoder().encode(value.trim()).length <= 32
            ? Promise.resolve() : Promise.reject(new Error('套餐标识最多 32 字节')) }]}>
        <Select placeholder="请选择会员套餐" loading={plans.isFetching} showSearch={{ optionFilterProp: 'label' }}
          options={plans.isSuccess ? plans.data : []} />
      </Form.Item>
      <div className="app-section-title"><span className="app-section-icon" aria-hidden="true"><CalendarOutlined /></span><span>有效期与权益</span></div>
      <Form.Item name="duration_days" label="有效天数" rules={[{ required: true, message: '请填写有效天数' },
        { type: 'integer', min: 1, max: 3650, message: '请输入 1 至 3650 的整数天数' }]}>
        <InputNumber min={1} max={3650} precision={0} suffix="天" />
      </Form.Item>
      <Form.Item name="benefits" label="会员权益" rules={[
        { validator: (_, value?: string) => !value || new TextEncoder().encode(value).length <= 512
          ? Promise.resolve() : Promise.reject(new Error('会员权益最多 512 字节')) }]}>
        <Input.TextArea rows={3} placeholder="填写本次授予的权益，可留空" maxLength={512} />
      </Form.Item>
      <Typography.Text type="secondary" className="app-secondary-text">有效期从开通成功时起计算。提交失败后可以重试，不会重复开通。</Typography.Text>
    </Form>
  </Modal>;
}
