import { useMemo, useState } from 'react';
import { Alert, App, Avatar, Button, Form, Modal, Space } from 'antd';
import { DeleteOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { setAppUserAvatar } from '../../api/app';
import type { AppUserInfo } from '../../api/types';
import { Auth } from '../../components/Auth';
import { FileUploadButton } from '../../components/FileUploadButton';
import { useFormDraft } from '../../hooks/useFormDraft';
import { useAuthStore } from '../../stores/auth';
import { invalidateAppData } from '../../utils/queryInvalidation';

interface Props { user: AppUserInfo; disabled?: boolean; onClose: () => void }
interface ExistingAvatarProps {
  value?: number; previewUrl: string; disabled: boolean; onChange?: (id: number) => void;
}
type AvatarForm = Pick<AppUserInfo, 'avatar_file_id'>;

function ExistingAvatar({ value, previewUrl, disabled, onChange }: ExistingAvatarProps) {
  return <Space><Avatar size={48} src={value ? previewUrl || undefined : undefined} icon={<UserOutlined />} />
    {!!value && <Button icon={<DeleteOutlined />} disabled={disabled} onClick={() => onChange?.(0)}>移除头像</Button>}
  </Space>;
}

export function AppUserAvatarDialog({ user, disabled, onClose }: Props) {
  const [form] = Form.useForm<AvatarForm>();
  const [uploading, setUploading] = useState(false);
  const { message } = App.useApp();
  const client = useQueryClient();
  const canUpload = useAuthStore(state => state.permissionsReady && state.permCodes.includes('sys:file:upload'));
  const values = useMemo(() => ({ avatar_file_id: user.avatar_file_id }), [user.avatar_file_id]);
  useFormDraft(form, { entityKey: user.id, values });
  const save = useMutation({
    mutationFn: (data: AvatarForm) => setAppUserAvatar(user.id, data.avatar_file_id ?? 0),
    onSuccess: async () => {
      await invalidateAppData(client);
      void message.success('APP用户头像已更新');
      onClose();
    },
  });
  const busy = uploading || save.isPending;
  return <Modal open title="设置APP用户头像" onCancel={onClose} onOk={() => form.submit()}
    confirmLoading={save.isPending} okButtonProps={{ disabled: disabled || uploading }}
    cancelButtonProps={{ disabled: busy }} closable={!busy} keyboard={!busy} mask={{ closable: !busy }}>
    <Form name="app-user-avatar" form={form} layout="vertical" disabled={disabled || busy}
      onFinish={(data) => { if (!disabled && !busy) save.mutate(data); }}>
      <Form.Item label="账号"><span>{user.account}</span></Form.Item>
      <Auth code="sys:file:upload"><Form.Item name="avatar_file_id" label="头像" extra="支持 PNG、JPEG、GIF，最大10 MiB；保存后生效。">
        <FileUploadButton scene="avatar" visibility="public" previewUrl={user.avatar_url}
          disabled={disabled || save.isPending} onUploadingChange={setUploading} />
      </Form.Item></Auth>
      {!canUpload && <Form.Item name="avatar_file_id" label="头像" extra="没有文件上传权限，可移除当前头像。">
        <ExistingAvatar previewUrl={user.avatar_url} disabled={!!disabled || busy} />
      </Form.Item>}
      {disabled && <Alert type="info" showIcon title="用户资料正在刷新或暂不可用，请稍后再保存。" />}
    </Form>
  </Modal>;
}
