import { useMemo } from 'react';
import { App, Form, Modal } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createRole, fetchRole, updateRole } from '../../api/role';
import type { CreateRoleReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { invalidateRoleData } from '../../utils/queryInvalidation';
import { RoleFormFields } from './RoleFormFields';
import { queryKeys } from '../../utils/queryKeys';

interface Props { id?: number; onClose: () => void }
const defaults = { description: '', data_scope: 'SELF', sort: 0, status: 1 };

export function RoleEditor({ id, onClose }: Props) {
  const [form] = Form.useForm<CreateRoleReq>();
  const { message } = App.useApp();
  const client = useQueryClient();
  const role = useQuery({ queryKey: [queryKeys.role, id], queryFn: () => fetchRole(id!), enabled: id !== undefined });
  const editorValues = useMemo(() => {
    if (!role.data) return undefined;
    const { name, code, description, data_scope, sort, status } = role.data;
    return { name, code, description, data_scope, sort, status };
  }, [role.data]);
  useFormDraft(form, { entityKey: id ?? 'new', values: id === undefined ? defaults : editorValues });
  const save = useMutation({
    mutationFn: ({ name, code, description, data_scope, sort, status }: CreateRoleReq) => {
      const values = { name, code, description, data_scope, sort, status };
      return id === undefined ? createRole(values) : updateRole(id, values);
    },
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
