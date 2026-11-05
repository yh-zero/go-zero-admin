import { App, Form, Modal } from 'antd';
import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createUser, fetchUser, updateUser } from '../../api/user';
import { fetchRoleOptions } from '../../api/role';
import { fetchProfile } from '../../api/profile';
import type { CreateUserReq, DeptInfo, UpdateUserReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { invalidateUserData } from '../../utils/queryInvalidation';
import { UserFormFields } from './UserFormFields';
import { queryKeys } from '../../utils/queryKeys';

interface Props { id?: number; depts: DeptInfo[]; onClose: () => void }
const defaults = { nickname: '', real_name: '', email: '', phone: '', dept_id: 0, avatar_file_id: 0, gender: 0, status: 1, remark: '' };

export function UserEditor({ id, depts, onClose }: Props) {
  const [form] = Form.useForm<CreateUserReq>();
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const { message } = App.useApp();
  const client = useQueryClient();
  const user = useQuery({ queryKey: [queryKeys.user, id], queryFn: () => fetchUser(id!), enabled: id !== undefined });
  const profile = useQuery({ queryKey: [queryKeys.profile], queryFn: fetchProfile });
  const canAssignRoles = profile.data?.is_super_admin === true;
  const roles = useQuery({ queryKey: [queryKeys.roleOptions], queryFn: fetchRoleOptions, enabled: canAssignRoles });
  useFormDraft(form, { entityKey: id ?? 'new', values: id === undefined ? defaults : user.data });
  const save = useMutation({
    mutationFn: (values: CreateUserReq) => {
      if (id === undefined) return createUser({ ...values, role_ids: canAssignRoles ? values.role_ids ?? [] : undefined });
      const patch: UpdateUserReq = {
      nickname: values.nickname, real_name: values.real_name, email: values.email, phone: values.phone,
      dept_id: values.dept_id, avatar_file_id: values.avatar_file_id, gender: values.gender,
      status: values.status, remark: values.remark, role_ids: canAssignRoles ? values.role_ids ?? [] : undefined,
      };
      return updateUser(id, Object.fromEntries(Object.entries(patch).filter(([key]) => form.isFieldTouched(key as keyof CreateUserReq))));
    },
    onSuccess: async () => {
      await invalidateUserData(client);
      message.success(id === undefined ? '用户已创建' : '用户已更新');
      onClose();
    },
  });
  const unavailable = profile.isPending || profile.isError || (canAssignRoles && (roles.isPending || roles.isError))
    || (id !== undefined && (!user.data || user.isError || user.isFetching));
  return <Modal open width={680} title={id === undefined ? '新增用户' : '编辑用户'} onCancel={onClose}
    onOk={() => form.submit()} confirmLoading={save.isPending} okButtonProps={{ disabled: unavailable || uploadingAvatar }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} mask={{ closable: !save.isPending }}>
    {id !== undefined && <QueryState isPending={user.isPending} error={user.error} onRetry={() => void user.refetch()} />}
    {canAssignRoles && <QueryState isPending={roles.isPending} error={roles.error} onRetry={() => void roles.refetch()} />}
    {profile.isError && <QueryState isPending={false} error={profile.error} onRetry={() => void profile.refetch()} />}
    <Form name="user-editor" className="editor-grid" form={form} layout="vertical" disabled={unavailable || save.isPending} onFinish={(values) => { if (!uploadingAvatar) save.mutate(values); }}
      initialValues={defaults}>
      <UserFormFields editing={id !== undefined} depts={depts} currentDeptId={user.data?.dept_id} currentDeptName={user.data?.dept_name}
        avatarUrl={user.data?.avatar_url} disabled={unavailable || save.isPending} onUploadingChange={setUploadingAvatar}
        canAssignRoles={canAssignRoles} roles={roles.data ?? []} rolesLoading={roles.isFetching} currentRoleIds={user.data?.role_ids ?? []} />
    </Form>
  </Modal>;
}
