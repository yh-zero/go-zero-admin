import { App, Form, Modal } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createMenu, fetchMenu, updateMenu } from '../../api/menu';
import type { CreateMenuReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { MenuFields } from './MenuFields';

interface MenuFormProps { open: boolean; editingId: number | null; onClose: () => void }

const defaults: CreateMenuReq = {
  parent_id: 0, title: '', name: '', type: 2, path: '', component: '', perm_code: '',
  icon: '', sort: 0, visible: true, status: 1,
};

export function MenuForm({ open, editingId, onClose }: MenuFormProps) {
  const [form] = Form.useForm<CreateMenuReq>();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const detail = useQuery({
    queryKey: ['menus', 'detail', editingId], queryFn: () => fetchMenu(editingId!),
    enabled: open && editingId !== null, refetchOnWindowFocus: false,
  });
  useFormDraft(form, { open, entityKey: editingId ?? 'new', values: editingId === null ? defaults : detail.data });
  const save = useMutation({
    mutationFn: (values: CreateMenuReq) => {
      const data = { ...defaults, ...values, parent_id: values.parent_id ?? 0, sort: values.sort ?? 0 };
      return editingId === null ? createMenu(data) : updateMenu(editingId, data);
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['menus'] }),
        queryClient.invalidateQueries({ queryKey: ['my-menus'] }),
        queryClient.invalidateQueries({ queryKey: ['perm-codes'] }),
      ]);
      message.success(editingId === null ? '菜单已创建' : '菜单已更新');
      onClose();
    },
  });
  const loadingDetail = editingId !== null && detail.isFetching;
  const blocked = editingId !== null && (detail.isPending || !!detail.error || !detail.data);
  return <Modal title={editingId === null ? '新增菜单' : '编辑菜单'} open={open} width={760}
    onCancel={() => { if (!save.isPending) onClose(); }} onOk={() => form.submit()}
    confirmLoading={save.isPending} okButtonProps={{ disabled: blocked || loadingDetail }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} maskClosable={!save.isPending} destroyOnHidden>
    {editingId !== null && <QueryState isPending={loadingDetail} error={detail.error} onRetry={() => { void detail.refetch(); }} />}
    <Form name="menu-editor" form={form} layout="vertical" initialValues={defaults}
      disabled={save.isPending || blocked || loadingDetail}
      onFinish={(values) => { if (!blocked && !loadingDetail && !save.isPending) save.mutate(values); }}><MenuFields /></Form>
  </Modal>;
}
