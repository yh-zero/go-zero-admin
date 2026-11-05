import { useMemo } from 'react';
import { Alert, App, Form, Modal } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createMenu, fetchMenu, updateMenu } from '../../api/menu';
import type { CreateMenuReq, MenuInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { invalidateMenuData } from '../../utils/queryInvalidation';
import { MenuFields } from './MenuFields';
import { PermissionFields } from './PermissionFields';
import { queryKeys } from '../../utils/queryKeys';

export interface MenuEditorState { id: number | null; kind: 'navigation' | 'permission'; parentId?: number }
interface Props { editor: MenuEditorState; resources: MenuInfo[]; resourcesReady: boolean; onClose: () => void }

const defaults: CreateMenuReq = {
  parent_id: 0, title: '', name: '', type: 2, path: '', component: '', perm_code: '',
  icon: '', sort: 0, visible: true, status: 1,
};

function editableValues(menu: MenuInfo): CreateMenuReq {
  return { parent_id: menu.parent_id, title: menu.title, name: menu.name, type: menu.type,
    path: menu.path, component: menu.component, perm_code: menu.perm_code, icon: menu.icon,
    sort: menu.sort, visible: menu.visible, status: menu.status };
}

export function MenuForm({ editor, resources, resourcesReady, onClose }: Props) {
  const [form] = Form.useForm<CreateMenuReq>();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const permission = editor.kind === 'permission';
  const detail = useQuery({
    queryKey: [queryKeys.menus, 'detail', editor.id], queryFn: () => fetchMenu(editor.id!),
    enabled: editor.id !== null, refetchOnWindowFocus: false,
  });
  const initialValues = useMemo(() => editor.id === null
    ? { ...defaults, parent_id: editor.parentId ?? 0, type: permission ? 3 : 2, visible: !permission }
    : detail.data ? editableValues(detail.data) : undefined,
  [editor.id, editor.parentId, permission, detail.data]);
  useFormDraft(form, { entityKey: editor.id ?? 'new', values: initialValues });
  const save = useMutation({
    mutationFn: (values: CreateMenuReq) => {
      const data: CreateMenuReq = { ...defaults, ...initialValues, ...values,
        parent_id: values.parent_id ?? initialValues?.parent_id ?? 0, sort: values.sort ?? initialValues?.sort ?? 0 };
      if (permission) {
        data.type = 3;
        data.name ||= data.perm_code?.replace(/[^a-zA-Z0-9_-]/g, '_').slice(0, 64) ?? '';
      }
      return editor.id === null ? createMenu(data) : updateMenu(editor.id, data);
    },
    onSuccess: async () => {
      await invalidateMenuData(queryClient);
      message.success(permission ? '操作权限已保存' : editor.id === null ? '菜单已创建' : '菜单已更新');
      onClose();
    },
  });
  const wrongType = !!detail.data && (permission ? detail.data.type !== 3 : detail.data.type !== 1 && detail.data.type !== 2);
  const blocked = !resourcesReady || wrongType || editor.id !== null && (detail.isPending || detail.isFetching || !!detail.error || !detail.data);
  const title = permission ? editor.id === null ? '新增操作权限' : '编辑操作权限' : editor.id === null ? '新增菜单' : '编辑菜单';
  return <Modal title={title} open width={720}
    onCancel={() => { if (!save.isPending) onClose(); }} onOk={() => form.submit()}
    confirmLoading={save.isPending} okButtonProps={{ disabled: blocked }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} mask={{ closable: !save.isPending }} destroyOnHidden>
    {editor.id !== null && <QueryState isPending={detail.isFetching} error={detail.error} onRetry={() => { void detail.refetch(); }} />}
    {wrongType && <Alert type="warning" showIcon title="资源类型已变化，请关闭后重新打开。" />}
    {!resourcesReady && <Alert type="info" showIcon title="资源列表正在刷新或加载失败，暂时无法保存。" />}
    <Form name={permission ? 'permission-editor' : 'menu-editor'} form={form} layout="vertical"
      initialValues={initialValues} disabled={save.isPending || blocked}
      onFinish={values => { if (!blocked && !save.isPending) save.mutate(values); }}>
      {permission ? <PermissionFields resources={resources} lockParent={editor.id === null && editor.parentId !== undefined} />
        : <MenuFields resources={resources} editingId={editor.id} />}
    </Form>
  </Modal>;
}
