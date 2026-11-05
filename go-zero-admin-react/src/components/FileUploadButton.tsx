import { useState } from 'react';
import { App, Avatar, Button, Space, Upload } from 'antd';
import type { UploadProps } from 'antd';
import { DeleteOutlined, UploadOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { uploadFile } from '../api/file';
import type { FileInfo, UploadFileReq } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { queryKeys } from '../utils/queryKeys';

interface Props {
  scene?: UploadFileReq['scene']; visibility?: UploadFileReq['visibility']; disabled?: boolean;
  value?: number; previewUrl?: string; onChange?: (id: number) => void; onUploaded?: (file: FileInfo) => void;
  onUploadingChange?: (uploading: boolean) => void;
}

export function FileUploadButton({ scene = 'common', visibility = 'private', disabled, value, previewUrl, onChange, onUploaded, onUploadingChange }: Props) {
  const { message } = App.useApp();
  const client = useQueryClient();
  const avatar = scene === 'avatar';
  const [reading, setReading] = useState(false);
  const finishUpload = () => { setReading(false); onUploadingChange?.(false); };
  const mutation = useMutation({ mutationFn: uploadFile, onSuccess: async (file) => {
    onChange?.(file.id);
    onUploaded?.(file);
    await client.invalidateQueries({ queryKey: [queryKeys.files] });
    void message.success(avatar ? '头像已上传，保存后生效' : '文件已上传');
  }, onSettled: finishUpload });
  const beforeUpload: UploadProps['beforeUpload'] = (file) => {
    if (!file.size || file.size > 10 * 1024 * 1024) {
      void message.warning('请选择大小在 1 字节至 10 MiB 之间的文件');
      return Upload.LIST_IGNORE;
    }
    if (avatar && !/\.(png|jpe?g|gif)$/i.test(file.name)) {
      void message.warning('头像支持 PNG、JPEG 或 GIF 图片');
      return Upload.LIST_IGNORE;
    }
    return true;
  };
  const customRequest: UploadProps['customRequest'] = ({ file, onSuccess, onError }) => {
    if (!(file instanceof File)) return;
    setReading(true); onUploadingChange?.(true);
    const { sessionVersion, departmentVersion } = useAuthStore.getState();
    const reader = new FileReader();
    reader.onerror = () => { finishUpload(); void message.error('读取文件失败，请重新选择'); onError?.(new Error('读取文件失败')); };
    reader.onabort = finishUpload;
    reader.onload = () => {
      const current = useAuthStore.getState();
      if (sessionVersion !== current.sessionVersion || departmentVersion !== current.departmentVersion) {
        finishUpload(); onError?.(new Error('会话或部门已切换')); return;
      }
      const content = String(reader.result).split(',')[1];
      if (!content) { finishUpload(); onError?.(new Error('文件内容为空')); return; }
      setReading(false);
      mutation.mutate({ name: file.name, content_base64: content, scene, visibility }, {
        onSuccess: (uploaded) => onSuccess?.(uploaded),
        onError: (error) => onError?.(error),
      });
    };
    reader.readAsDataURL(file);
    return { abort: () => { if (reader.readyState === FileReader.LOADING) reader.abort(); } };
  };
  const imageUrl = value && mutation.data?.id === value ? mutation.data.url : value ? previewUrl : undefined;
  return <Space wrap>
    {avatar && <Avatar size={48} src={imageUrl || undefined} icon={<UserOutlined />} />}
    <Upload accept={avatar ? 'image/png,image/jpeg,image/gif' : undefined} maxCount={1} multiple={false}
      showUploadList={false} disabled={disabled || reading || mutation.isPending} beforeUpload={beforeUpload} customRequest={customRequest}>
      <Button icon={<UploadOutlined />} loading={reading || mutation.isPending} disabled={disabled}>{avatar ? '上传头像' : '上传文件'}</Button>
    </Upload>
    {avatar && !!value && <Button icon={<DeleteOutlined />} disabled={disabled || reading || mutation.isPending} onClick={() => onChange?.(0)}>移除头像</Button>}
  </Space>;
}
