import { App, Form, Modal } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createUser, fetchUser, updateUser } from '../../api/user';
import type { CreateUserReq, DeptInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { UserFormFields } from './UserFormFields';

interface Props { id?: number; depts: DeptInfo[]; onClose: () => void }
const defaults = { nickname: '', real_name: '', email: '', phone: '', dept_id: 0, avatar_file_id: 0, gender: 0, status: 1, remark: '' };

export function UserEditor({ id, depts, onClose }: Props) {
  const [form] = Form.useForm<CreateUserReq>();
  const { message } = App.useApp();
  const client = useQueryClient();
  const user = useQuery({ queryKey: ['user', id], queryFn: () => fetchUser(id!), enabled: id !== undefined });
  useFormDraft(form, { entityKey: id ?? 'new', values: id === undefined ? defaults : user.data });
  const save = useMutation({
    mutationFn: (values: CreateUserReq) => id === undefined ? createUser(values) : updateUser(id, {
      nickname: values.nickname, real_name: values.real_name, email: values.email, phone: values.phone,
      dept_id: values.dept_id, avatar_file_id: values.avatar_file_id, gender: values.gender,
      status: values.status, remark: values.remark,
    }),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: ['users'] }),
        client.invalidateQueries({ queryKey: ['user', id] }),
      ]);
      message.success(id === undefined ? '用户已创建' : '用户已更新');
      onClose();
    },
  });
  const unavailable = id !== undefined && (!user.data || user.isError || user.isFetching);
  return <Modal open title={id === undefined ? '新增用户' : '编辑用户'} onCancel={onClose}
    onOk={() => form.submit()} confirmLoading={save.isPending} okButtonProps={{ disabled: unavailable }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} maskClosable={!save.isPending}>
    {id !== undefined && <QueryState isPending={user.isPending} error={user.error} onRetry={() => void user.refetch()} />}
    <Form name="user-editor" form={form} layout="vertical" disabled={unavailable || save.isPending} onFinish={(values) => save.mutate(values)}
      initialValues={defaults}>
      <UserFormFields editing={id !== undefined} depts={depts} currentDeptId={user.data?.dept_id} currentDeptName={user.data?.dept_name} />
    </Form>
  </Modal>;
}
