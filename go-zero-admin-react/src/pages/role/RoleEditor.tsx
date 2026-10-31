import { App, Form, Modal } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createRole, fetchRole, updateRole } from '../../api/role';
import type { CreateRoleReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { invalidateRoleData } from '../../utils/queryInvalidation';
import { RoleFormFields } from './RoleFormFields';

interface Props { id?: number; onClose: () => void }
const defaults = { description: '', data_scope: 'SELF', sort: 0, status: 1 };

export function RoleEditor({ id, onClose }: Props) {
  const [form] = Form.useForm<CreateRoleReq>();
  const { message } = App.useApp();
  const client = useQueryClient();
  const role = useQuery({ queryKey: ['role', id], queryFn: () => fetchRole(id!), enabled: id !== undefined });
  useFormDraft(form, { entityKey: id ?? 'new', values: id === undefined ? defaults : role.data });
  const save = useMutation({
    mutationFn: (values: CreateRoleReq) => id === undefined ? createRole({ ...values, menu_ids: [] })
      : updateRole(id, { ...values, menu_ids: role.data!.menu_ids ?? [] }),
    onSuccess: async () => {
      await invalidateRoleData(client);
      message.success(id === undefined ? '角色已创建' : '角色已更新');
      onClose();
    },
  });
  const unavailable = id !== undefined && (!role.data || role.isError || role.isFetching);
  return <Modal open width={680} title={id === undefined ? '新增角色' : '编辑角色'} onCancel={onClose} onOk={() => form.submit()}
    confirmLoading={save.isPending} okButtonProps={{ disabled: unavailable }} cancelButtonProps={{ disabled: save.isPending }}
    closable={!save.isPending} mask={{ closable: !save.isPending }}>
    {id !== undefined && <QueryState isPending={role.isPending} error={role.error} onRetry={() => void role.refetch()} />}
    <Form name="role-editor" className="editor-grid" form={form} layout="vertical" disabled={unavailable || save.isPending} onFinish={(values) => save.mutate(values)}
      initialValues={defaults}>
      <RoleFormFields editing={id !== undefined} />
    </Form>
  </Modal>;
}
