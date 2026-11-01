import { App, Form, Input, Modal, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchAppUser, updateAppUser } from '../../api/app';
import type { UpdateAppUserReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { invalidateAppData } from '../../utils/queryInvalidation';

interface Props { id: number; onClose: () => void }

export function AppUserEditDialog({ id, onClose }: Props) {
  const [form] = Form.useForm<UpdateAppUserReq>();
  const { message } = App.useApp();
  const client = useQueryClient();
  const user = useQuery({ queryKey: ['app-user', id], queryFn: () => fetchAppUser(id) });
  useFormDraft(form, { entityKey: id, values: user.data });
  const save = useMutation({ mutationFn: (values: UpdateAppUserReq) => updateAppUser(id, values),
  onSuccess: async () => {
    await invalidateAppData(client); void message.success('APP用户资料已保存'); onClose();
  } });
  const unavailable = !user.isSuccess || user.isFetching;
  return <Modal open className="app-form-dialog" title="编辑APP用户资料" width={580} confirmLoading={save.isPending}
    onOk={() => form.submit()} onCancel={() => { if (!save.isPending) onClose(); }}
    okButtonProps={{ disabled: unavailable }} cancelButtonProps={{ disabled: save.isPending }}
    closable={!save.isPending} mask={{ closable: !save.isPending }} keyboard={!save.isPending}>
    <QueryState isPending={user.isPending} error={user.error} onRetry={() => user.refetch()} />
    <Form name="app-user-editor" form={form} layout="vertical" disabled={unavailable || save.isPending}
      onFinish={values => {
        if (unavailable || save.isPending) return;
        const changes: UpdateAppUserReq = Object.fromEntries(Object.entries(values)
          .filter(([key]) => form.isFieldTouched(key as keyof UpdateAppUserReq)));
        if (!Object.keys(changes).length) { onClose(); return; }
        save.mutate(changes);
      }}>
      <Form.Item label="账号"><Input value={user.data?.account ?? ''} readOnly /></Form.Item>
      <Form.Item name="nickname" label="昵称" rules={[{ max: 64, message: '昵称最多64个字符' }]}><Input maxLength={64} allowClear /></Form.Item>
      <Form.Item name="remark" label="内部备注" extra="仅后台管理员可见，与用户审核说明分开保存。"
        rules={[{ max: 500, message: '备注最多500个字符' }]}>
        <Input.TextArea rows={4} maxLength={500} showCount allowClear placeholder="填写管理备注" />
      </Form.Item>
      <Typography.Text type="secondary">账号及审核状态由专用入口维护；清空昵称或备注后保存即可移除原内容。</Typography.Text>
    </Form>
  </Modal>;
}
