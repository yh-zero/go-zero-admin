import { App, Button, Form, Input } from 'antd';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { updateProfile } from '../../api/profile';
import type { ProfileInfo, UpdateProfileReq } from '../../api/types';
import { useFormDraft } from '../../hooks/useFormDraft';
export function ProfileEditor({ profile }: { profile: ProfileInfo }) {
  const [form] = Form.useForm<UpdateProfileReq>();
  const client = useQueryClient();
  const { message } = App.useApp();
  const resetDraft = useFormDraft(form, { entityKey: profile.id, values: profile });
  const mutation = useMutation({ mutationFn: updateProfile, onSuccess: async (_, values) => {
    await client.invalidateQueries({ queryKey: ['profile'] });
    const refreshed = client.getQueryState<ProfileInfo>(['profile']);
    resetDraft(refreshed?.status === 'success' && !refreshed.isInvalidated && refreshed.data
      ? refreshed.data : { ...profile, ...values });
    void message.success('个人资料已保存');
  } });
  return <>
    <Form name="profile-editor" form={form} layout="vertical" disabled={mutation.isPending}
      onFinish={(values) => { if (!mutation.isPending) mutation.mutate({ ...values, gender: profile.gender, avatar_url: profile.avatar_url }); }} className="profile-form">
      <Form.Item name="nickname" label="昵称" rules={[{ max: 64, message: '最多64个字符' }]}><Input maxLength={64} /></Form.Item>
      <Form.Item name="real_name" label="真实姓名" rules={[{ max: 64, message: '最多64个字符' }]}><Input maxLength={64} /></Form.Item>
      <Form.Item name="email" label="邮箱" rules={[{ type: 'email', message: '请输入有效邮箱' }, { max: 128, message: '最多128个字符' }]}><Input maxLength={128} /></Form.Item>
      <Form.Item name="phone" label="手机号"><Input maxLength={20} /></Form.Item>
      <Button type="primary" htmlType="submit" loading={mutation.isPending}>保存资料</Button>
    </Form>
  </>;
}
