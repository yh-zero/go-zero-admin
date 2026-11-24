import { useState } from 'react';
import { App, Avatar, Button, Space, Upload } from 'antd';
import type { UploadProps } from 'antd';
import { DeleteOutlined, UploadOutlined, UserOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { uploadMultipart } from '../api/file';
import type { FileInfo, UploadFileReq } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { queryKeys } from '../utils/queryKeys';

interface Props {
  scene?: UploadFileReq['scene']; visibility?: UploadFileReq['visibility']; disabled?: boolean;
  value?: number; previewUrl?: string; onChange?: (id: number) => void; onUploaded?: (file: FileInfo) => void;
  onUploadingChange?: (uploading: boolean) => void;
}
type UploadInput = { file: File; scene: UploadFileReq['scene']; visibility: UploadFileReq['visibility'] };

export function FileUploadButton({ scene = 'common', visibility = 'private', disabled, value, previewUrl, onChange, onUploaded, onUploadingChange }: Props) {
  const { message } = App.useApp();
  const client = useQueryClient();
  const avatar = scene === 'avatar';
  const [uploading, setUploading] = useState(false);
  const finishUpload = () => { setUploading(false); onUploadingChange?.(false); };
  const mutation = useMutation({
    mutationFn: ({ file, scene: uploadScene, visibility: uploadVisibility }: UploadInput) => uploadMultipart(file, uploadScene, uploadVisibility),
    onSuccess: async (file) => {
      onChange?.(file.id);
      onUploaded?.(file);
      await Promise.all([
        client.invalidateQueries({ queryKey: [queryKeys.files] }),
        client.invalidateQueries({ queryKey: [queryKeys.fileStats] }),
      ]);
      void message.success(avatar ? '头像已上传，保存后生效' : '文件已上传');
    },
    onSettled: finishUpload,
  });
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
    const { sessionVersion, departmentVersion } = useAuthStore.getState();
    setUploading(true); onUploadingChange?.(true);
    if (sessionVersion !== useAuthStore.getState().sessionVersion || departmentVersion !== useAuthStore.getState().departmentVersion) {
      finishUpload(); onError?.(new Error('会话或部门已切换')); return;
    }
    mutation.mutate({ file, scene, visibility }, {
      onSuccess: (uploaded) => onSuccess?.(uploaded),
      onError: (error) => onError?.(error),
    });
    return { abort: () => { finishUpload(); mutation.reset(); } };
  };
  const imageUrl = value && mutation.data?.id === value ? mutation.data.url : value ? previewUrl : undefined;
  return <Space wrap>
    {avatar && <Avatar size={48} src={imageUrl || undefined} icon={<UserOutlined />} />}
    <Upload accept={avatar ? 'image/png,image/jpeg,image/gif' : undefined} maxCount={1} multiple={false}
      showUploadList={false} disabled={disabled || uploading} beforeUpload={beforeUpload} customRequest={customRequest}>
      <Button icon={<UploadOutlined />} loading={uploading} disabled={disabled}>{avatar ? '上传头像' : '上传文件'}</Button>
    </Upload>
    {avatar && !!value && <Button icon={<DeleteOutlined />} disabled={disabled || uploading} onClick={() => onChange?.(0)}>移除头像</Button>}
  </Space>;
}
