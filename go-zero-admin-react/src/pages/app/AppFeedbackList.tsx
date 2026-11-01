import { useRef, useState } from 'react';
import { App, Avatar, Button, Card, Descriptions, Empty, Form, Input, InputNumber, Modal, Select, Space, Table, Tag } from 'antd';
import { EditOutlined, EyeOutlined, MessageOutlined, ReloadOutlined, SearchOutlined, UndoOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { fetchAppFeedback, updateAppFeedback } from '../../api/app';
import type { AppFeedbackInfo, GetAppFeedbackListReq, UpdateAppFeedbackReq } from '../../api/types';
import { Auth } from '../../components/Auth';
import { PageHeader } from '../../components/PageHeader';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { usePagedQuery } from '../../hooks/usePagedQuery';
import { useAuthStore } from '../../stores/auth';
import { formatTime } from '../../utils/format';
import { invalidateAppData } from '../../utils/queryInvalidation';
import './app.css';

const statusOptions = [{ value: 'pending', label: '待处理' }, { value: 'resolved', label: '已处理' }];
interface FeedbackDialogProps { feedback: AppFeedbackInfo; readOnly: boolean; onClose: () => void }

function FeedbackDialog({ feedback, readOnly, onClose }: FeedbackDialogProps) {
  const [form] = Form.useForm<UpdateAppFeedbackReq>();
  const client = useQueryClient();
  const { message } = App.useApp();
  const submitting = useRef(false);
  const canReply = useAuthStore(state => state.permissionsReady && state.permCodes.includes('sys:app:feedback:reply'));
  const editable = !readOnly && canReply;
  useFormDraft(form, { entityKey: feedback.id, values: { reply: feedback.reply, status: feedback.status } });
  const save = useMutation({ mutationFn: (values: UpdateAppFeedbackReq) => updateAppFeedback(feedback.id, values), onSuccess: async () => {
    await invalidateAppData(client); void message.success('反馈处理结果已保存'); onClose();
  }, onSettled: () => { submitting.current = false; } });
  return <Modal open className="app-form-dialog app-feedback-dialog" title={<span className="app-section-title"><span className="app-section-icon app-section-icon--cyan"><MessageOutlined /></span>{editable ? '处理反馈' : '反馈详情'}</span>} width={680}
    onCancel={() => { if (!save.isPending) onClose(); }} onOk={() => form.submit()}
    okText="保存处理结果" confirmLoading={save.isPending} cancelButtonProps={{ disabled: save.isPending }}
    footer={editable ? undefined : <Button onClick={onClose}>关闭</Button>}
    closable={!save.isPending} mask={{ closable: !save.isPending }} keyboard={!save.isPending}>
    <div className="app-feedback-thread"><div className="app-feedback-author"><Avatar shape="square" size={38} className="app-avatar--cyan" icon={<UserOutlined />} />
      <div className="app-identity-copy"><strong className="app-primary-text">{feedback.nickname || feedback.account}</strong>
        <span className="app-secondary-text">{feedback.account} · #{feedback.user_id}</span></div>
      <Tag color={feedback.status === 'resolved' ? 'success' : 'gold'}>{feedback.status === 'resolved' ? '已处理' : '待处理'}</Tag></div>
      <div className="app-feedback-message"><span className="app-secondary-text">用户反馈</span><p className="app-plain-text">{feedback.content}</p>
        <time className="app-secondary-text">{formatTime(feedback.created_at)}</time></div></div>
    {editable ? <Form name="app-feedback-reply" form={form} layout="vertical" className="app-feedback-editor" disabled={save.isPending}
      onFinish={values => {
        if (submitting.current || !editable) return;
        const changes: UpdateAppFeedbackReq = Object.fromEntries(Object.entries(values)
          .filter(([key]) => form.isFieldTouched(key as keyof UpdateAppFeedbackReq)));
        if (!Object.keys(changes).length) { onClose(); return; }
        submitting.current = true; save.mutate(changes);
      }}>
      <h3 className="app-profile-section-title"><EditOutlined />回复与处理</h3>
      <Form.Item name="reply" label="回复内容" rules={[{ max: 2000, message: '回复最多2000个字符' }]}>
        <Input.TextArea rows={5} maxLength={2000} showCount allowClear placeholder="填写给用户的回复，可清空原回复" />
      </Form.Item>
      <Form.Item name="status" label="处理状态" rules={[{ required: true, message: '请选择处理状态' }]}><Select options={statusOptions} /></Form.Item>
    </Form> : <Descriptions size="small" column={1} className="app-feedback-editor" items={[
      { key: 'reply', label: '回复内容', children: <span className="app-plain-text">{feedback.reply || '尚未回复'}</span> },
      { key: 'status', label: '处理状态', children: <Tag color={feedback.status === 'resolved' ? 'success' : 'gold'}>{feedback.status === 'resolved' ? '已处理' : '待处理'}</Tag> },
      { key: 'updated_at', label: '最近更新', children: formatTime(feedback.updated_at) },
    ]} />}
  </Modal>;
}

export default function AppFeedbackList() {
  const [form] = Form.useForm<GetAppFeedbackListReq>();
  const [params, setParams] = useState<GetAppFeedbackListReq>({ page: 1, page_size: 20 });
  const [selected, setSelected] = useState<{ feedback: AppFeedbackInfo; readOnly: boolean } | null>(null);
  const feedback = usePagedQuery({ params, setParams, queryKey: ['app-feedback', params], queryFn: () => fetchAppFeedback(params) });
  return <div className="page-stack app-page">
    <div className="app-page-heading"><span className="app-heading-icon app-heading-icon--cyan"><MessageOutlined /></span>
      <PageHeader title="意见反馈" description="倾听用户的声音，及时回应每一条建议" /></div>
    <Card className="filter-card app-filter-card"><Form name="app-feedback-filter" form={form} layout="inline" className="filter-form"
      onFinish={values => setParams({ ...values, page: 1, page_size: params.page_size })}>
      <Form.Item name="user_id" label="用户 ID" rules={[{ type: 'integer', min: 1, max: Number.MAX_SAFE_INTEGER, message: '请输入有效用户 ID' }]}>
        <InputNumber min={1} max={Number.MAX_SAFE_INTEGER} precision={0} placeholder="全部用户" />
      </Form.Item>
      <Form.Item name="status" label="处理状态"><Select className="filter-select" allowClear placeholder="全部状态" options={statusOptions} /></Form.Item>
      <Form.Item><Space><Button type="primary" icon={<SearchOutlined />} htmlType="submit">查询</Button>
        <Button icon={<UndoOutlined />} onClick={() => { form.resetFields(); setParams({ page: 1, page_size: params.page_size }); }}>重置</Button>
        <Button icon={<ReloadOutlined />} title="刷新" aria-label="刷新APP反馈" loading={feedback.isFetching} onClick={() => void feedback.refetch()} />
      </Space></Form.Item>
    </Form></Card>
    <Card className="data-card app-data-card" title={<span className="app-section-title"><span className="app-section-icon app-section-icon--cyan"><MessageOutlined /></span>
      用户反馈{feedback.isSuccess && <span className="app-count">{feedback.data.total}</span>}</span>} extra={<span className="app-card-hint">每一条建议都值得回应</span>}>
      <QueryState isPending={feedback.isPending} error={feedback.error} onRetry={() => feedback.refetch()} />
      {!feedback.isError && <Table<AppFeedbackInfo> rowKey="id" dataSource={feedback.data?.list ?? []} loading={feedback.isFetching} scroll={{ x: 1100 }}
        locale={{ emptyText: feedback.isPending ? null : <Empty className="app-empty" image={<MessageOutlined />} description={<>
          <strong>{params.user_id || params.status ? '暂无匹配反馈' : '暂无用户反馈'}</strong><p>{params.user_id || params.status ? '调整筛选条件后再试' : '收到的用户建议会在这里展示'}</p></>} /> }}
        pagination={{ current: params.page, pageSize: params.page_size, total: feedback.data?.total ?? 0, showSizeChanger: true,
          showTotal: total => `共 ${total} 条反馈`, onChange: (page, page_size) => setParams({ ...params, page, page_size }) }} columns={[
          { title: '反馈用户', key: 'user', width: 265, render: (_, item) => <div className="app-table-identity">
            <Avatar shape="square" size={36} className="app-avatar--cyan">{(item.nickname || item.account).slice(0, 1)}</Avatar>
            <div className="app-identity-copy"><span className="app-primary-text">{item.nickname || item.account}</span>
              <span className="app-secondary-text">{item.account} · #{item.user_id}</span></div>
          </div> },
          { title: '反馈内容', dataIndex: 'content', ellipsis: true },
          { title: '回复', dataIndex: 'reply', ellipsis: true, render: value => <span className={value ? 'app-reply-preview' : 'app-secondary-text'}>{value || '尚未回复'}</span> },
          { title: '状态', dataIndex: 'status', width: 100, render: status => <Tag color={status === 'resolved' ? 'success' : 'gold'}>{status === 'resolved' ? '已处理' : '待处理'}</Tag> },
          { title: '提交时间', dataIndex: 'created_at', width: 170, render: value => <span className="app-date-text">{formatTime(value)}</span> },
          { title: '操作', key: 'actions', width: 155, render: (_, item) => <Space className="app-table-actions">
            <Button type="link" icon={<EyeOutlined />} disabled={feedback.isFetching} onClick={() => setSelected({ feedback: item, readOnly: true })}>详情</Button>
            <Auth code="sys:app:feedback:reply"><Button type="link" icon={<EditOutlined />} disabled={feedback.isFetching} onClick={() => setSelected({ feedback: item, readOnly: false })}>处理</Button></Auth>
          </Space> },
        ]} />}
    </Card>
    {selected && <FeedbackDialog key={selected.feedback.id} {...selected} onClose={() => setSelected(null)} />}
  </div>;
}
