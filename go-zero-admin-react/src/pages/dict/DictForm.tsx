import { App, Col, Form, Input, Modal, Row, Select } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createDict, fetchDictByCode, updateDict } from '../../api/dict';
import type { CreateDictReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useFormDraft } from '../../hooks/useFormDraft';
import { statusOptions } from '../../utils/formOptions';
import { DictItemsFields } from './DictItemsFields';

interface DictFormProps { open: boolean; editingCode: string | null; onClose: () => void }
const defaults: CreateDictReq = { name: '', code: '', items: [], description: '', status: 1 };

export function DictForm({ open, editingCode, onClose }: DictFormProps) {
  const [form] = Form.useForm<CreateDictReq>();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const detail = useQuery({ queryKey: ['dicts', 'detail', editingCode], queryFn: () => fetchDictByCode(editingCode!), enabled: open && editingCode !== null, refetchOnWindowFocus: false });
  useFormDraft(form, { open, entityKey: editingCode ?? 'new', values: editingCode === null ? defaults : detail.data });
  const save = useMutation({
    mutationFn: (values: CreateDictReq) => {
      const data: CreateDictReq = { ...defaults, ...values, items: (values.items ?? []).map((item) => ({ ...item, sort: item.sort ?? 0 })) };
      if (editingCode === null) return createDict(data);
      if (!detail.data) return Promise.reject(new Error('字典详情尚未读取完成'));
      return updateDict(detail.data.id, data);
    },
    onSuccess: async () => {
      onClose();
      await queryClient.invalidateQueries({ predicate: (query) => query.queryKey[0] === 'dicts' && query.queryKey[1] !== 'detail' });
      await queryClient.invalidateQueries({ queryKey: ['dicts', 'detail'], refetchType: 'none' });
      message.success(editingCode === null ? '字典已创建' : '字典已更新');
    },
  });
  const loading = editingCode !== null && detail.isFetching;
  const blocked = editingCode !== null && (detail.isPending || !!detail.error || !detail.data);
  return <Modal title={editingCode === null ? '新增字典' : '编辑字典'} open={open} width={800}
    onCancel={() => { if (!save.isPending) onClose(); }} onOk={() => form.submit()}
    confirmLoading={save.isPending} okButtonProps={{ disabled: blocked || loading }}
    cancelButtonProps={{ disabled: save.isPending }} closable={!save.isPending} mask={{ closable: !save.isPending }} destroyOnHidden>
    {editingCode !== null && <QueryState isPending={loading} error={detail.error} onRetry={() => { void detail.refetch(); }} />}
    <Form name="dict-editor" form={form} layout="vertical" initialValues={defaults} disabled={save.isPending || blocked || loading}
      onFinish={(values) => { if (!blocked && !loading && !save.isPending) save.mutate(values); }}>
      <Row gutter={16}>
        <Col xs={24} sm={12}><Form.Item name="name" label="字典名称" rules={[{ required: true, whitespace: true, message: '请输入字典名称' }]}><Input maxLength={64} /></Form.Item></Col>
        <Col xs={24} sm={12}><Form.Item name="code" label="字典编码" rules={[{ required: true, whitespace: true, message: '请输入字典编码' }]}><Input maxLength={64} /></Form.Item></Col>
        <Col xs={24} sm={12}><Form.Item name="status" label="状态"><Select options={statusOptions} /></Form.Item></Col>
        <Col span={24}><Form.Item name="description" label="描述"><Input.TextArea maxLength={512} showCount rows={2} /></Form.Item></Col>
      </Row>
      <DictItemsFields />
    </Form>
  </Modal>;
}
