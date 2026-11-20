import { Alert, App, Form, Input, Modal, Select } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchAppUser, reviewAppUser } from '../../api/app';
import type { AppUserInfo, AppUserStatus, ReviewAppUserReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { invalidateAppData } from '../../utils/queryInvalidation';
import { queryKeys } from '../../utils/queryKeys';
import { ApiError } from '../../utils/response';
import { appUserStatus, appUserStatusOptions } from './appOptions';

export type AppUserReviewIntent = 'review' | 'enable' | 'disable';
interface Props { user: AppUserInfo; intent?: AppUserReviewIntent; disabled?: boolean; onClose: () => void }

export function AppUserReviewDialog({ user, intent = 'review', disabled, onClose }: Props) {
  type ReviewForm = Pick<ReviewAppUserReq, 'status' | 'reason'>;
  const [form] = Form.useForm<ReviewForm>();
  const { message } = App.useApp();
  const client = useQueryClient();
  const values: ReviewForm = { status: intent === 'disable' ? 'disabled' : 'active', reason: user.review_reason };
  const selectedStatus: AppUserStatus | undefined = Form.useWatch('status', form);
  const exitsDevices = selectedStatus !== undefined && selectedStatus !== 'active';
  const save = useMutation({
    mutationFn: (decision: ReviewForm) => reviewAppUser(user.id, {
      ...decision, expected_status: user.status, expected_reason: user.review_reason,
    }),
    onError: (error) => {
      if (error instanceof ApiError && error.code === 409) {
        void client.invalidateQueries({ queryKey: [queryKeys.appUsers] });
      }
    },
    onSuccess: async () => {
      await invalidateAppData(client);
      void message.success('APP用户状态已更新');
      onClose();
    },
  });
  const conflict = save.error instanceof ApiError && save.error.code === 409;
  const title = intent === 'disable' ? '停用APP用户' : intent === 'enable' ? '启用APP用户' : '审核APP用户';
  const options = appUserStatusOptions.filter(option => intent === 'review'
    ? option.value === 'active' || option.value === 'rejected' : option.value === values.status);
  return <Modal open title={title} onCancel={() => { if (!save.isPending) onClose(); }} onOk={() => form.submit()}
    okText={exitsDevices ? '确认并退出旧设备' : intent === 'enable' ? '确认启用' : '保存审核结果'}
    confirmLoading={save.isPending} okButtonProps={{ danger: exitsDevices, disabled: disabled || conflict }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} keyboard={!save.isPending} mask={{ closable: !save.isPending }}>
    {conflict && <LatestReview id={user.id} />}
    <Form name="app-user-review" form={form} layout="vertical" disabled={disabled || save.isPending || conflict}
      initialValues={values}
      onFinish={(values) => { if (!disabled && !save.isPending && !conflict) save.mutate(values); }}>
      <Form.Item label="账号"><Input value={user.account} readOnly /></Form.Item>
      <Form.Item name="status" label="账号状态" rules={[{ required: true, message: '请选择账号状态' }]}>
        <Select options={options} />
      </Form.Item>
      <Form.Item name="reason" label={intent === 'disable' ? '停用说明' : intent === 'enable' ? '启用说明' : '审核说明'} rules={[{ max: 255, message: '最多255个字符' }]}>
        <Input.TextArea maxLength={255} rows={3} showCount placeholder="填写审核或状态调整说明" />
      </Form.Item>
      {exitsDevices && <Alert type="warning" showIcon title={`将账号设为${appUserStatus[selectedStatus].label}`}
        description="保存后，该用户已登录的设备将全部退出，需要审核启用后重新登录。" />}
      {disabled && <Alert type="info" showIcon title="用户资料正在刷新或暂不可用，请稍后再保存。" />}
    </Form>
  </Modal>;
}

function LatestReview({ id }: { id: number }) {
  const latest = useQuery({
    queryKey: [queryKeys.appUser, id], queryFn: () => fetchAppUser(id), refetchOnMount: 'always',
  });
  return <Alert type="warning" showIcon title="审核冲突，此次修改未保存"
    description={<>
      <QueryState isPending={latest.isFetching || latest.isPending} error={latest.error} onRetry={() => latest.refetch()} />
      {latest.isSuccess && !latest.isFetching && <p>最新状态：{appUserStatus[latest.data.status].label}。审核说明：{latest.data.review_reason || '无'}。</p>}
      请核对最新决定，关闭弹窗后可重新操作。
    </>} />;
}
