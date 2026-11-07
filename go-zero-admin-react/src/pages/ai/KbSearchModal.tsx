import { useRef } from 'react';
import { Alert, Card, Empty, Form, Input, Modal, Space, Typography } from 'antd';
import { useMutation } from '@tanstack/react-query';
import type { AiKbInfo, AiSearchKbReq, AiSearchKbResp } from '../../api/types';
import { KB_SEARCH_MAX_BYTES, kbSearchOptions, kbSearchQueryBytes, validateKbSearchQuery } from './knowledgeSearch';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';
import './knowledgeSearch.css';

interface KbSearchModalProps { scope: AiWorkspaceScope; kb: AiKbInfo; onCancel: () => void }

export default function KbSearchModal({ scope, kb, onCancel }: KbSearchModalProps) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list');
  const submitting = useRef(false);
  const [form] = Form.useForm<AiSearchKbReq>();
  const query = Form.useWatch('query', form) ?? '';
  const search = useMutation(kbSearchOptions(scope, kb.id));
  const busy = guard.writing || search.isPending;

  return <Modal open title={`召回测试：${kb.name}`} width={800} okText="开始召回" cancelText="关闭" confirmLoading={busy}
    closable={!busy} keyboard={!busy} mask={{ closable: !busy }} cancelButtonProps={{ disabled: busy }} okButtonProps={{ disabled: busy }}
    onCancel={() => { if (!submitting.current && guard.isCurrent()) onCancel(); }} onOk={() => { void submit(); }} destroyOnHidden>
    <Space orientation="vertical" size="middle" className="kb-search-content">
      <Alert type="info" showIcon title="直接查看检索片段，不生成聊天答案"
        description="片段并非完整原文。当前没有最低相似度限制，有候选片段不代表资料中有答案；相似度也不是正确率或答案置信度。" />
      <Form form={form} layout="vertical" disabled={busy} preserve={false}>
        <Form.Item name="query" label="查询内容" extra={`${kbSearchQueryBytes(query)} / ${KB_SEARCH_MAX_BYTES} 字节（UTF-8）`}
          rules={[{ validator: (_, value: string | undefined) => {
            const error = validateKbSearchQuery(value ?? '');
            return error ? Promise.reject(new Error(error)) : Promise.resolve();
          } }]}>
          <Input.TextArea rows={3} placeholder="输入想在此知识库检索的问题或关键词" />
        </Form.Item>
      </Form>
      {busy && <Alert type="info" showIcon title="正在召回，请稍候" />}
      {guard.isCurrent() && !busy && search.isError && <Alert type="error" showIcon title="召回失败"
        description="本次未能完成检索。请根据请求错误提示处理后重新测试；失败不代表没有相关片段。" />}
      {guard.isCurrent() && !busy && search.isSuccess && <>
        <Typography.Paragraph className="kb-search-query">本次查询：{search.variables.query}</Typography.Paragraph>
        <KbSearchResults result={search.data} />
      </>}
    </Space>
  </Modal>;

  async function submit() {
    if (!guard.beginWrite('sys:ai:kb:list')) return;
    // 同步保护验证期间的双击与关闭，React提交disabled之前也生效。
    submitting.current = true;
    search.reset();
    try {
      const values = await form.validateFields();
      if (guard.isCurrent()) await search.mutateAsync(values);
    } catch { /* 字段由Form提示；请求错误由统一请求层反馈，弹窗保留查询。 */ }
    finally { submitting.current = false; guard.finishWrite(); }
  }
}

export function KbSearchResults({ result }: { result: AiSearchKbResp }) {
  return <Space orientation="vertical" size="middle" className="kb-search-content">
    <Typography.Text type="secondary">召回 {result.list.length} 个片段 · 服务端检索耗时 {result.elapsed_ms} ms</Typography.Text>
    {!result.list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="未召回片段，可调整查询后重试" />}
    {result.list.map((hit, rank) => <Card key={hit.chunk_id} size="small" title={`${rank + 1}. ${hit.doc_title}`}>
      <Space wrap size="middle">
        <Typography.Text type="secondary">文档 ID：{hit.doc_id}</Typography.Text>
        <Typography.Text type="secondary">分块 ID：{hit.chunk_id}</Typography.Text>
        <Typography.Text>相似度：{hit.score}</Typography.Text>
      </Space>
      <Typography.Paragraph className="kb-search-snippet">{hit.snippet}</Typography.Paragraph>
    </Card>)}
  </Space>;
}
