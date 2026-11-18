import { Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import type { AiKbDocInfo } from '../../api/types';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';

export function AskDocumentButton({ scope, doc, disabled }: { scope: AiWorkspaceScope; doc: AiKbDocInfo; disabled?: boolean }) {
  const navigate = useNavigate();
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list');
  const canAsk = () => guard.can('sys:ai:conversation:list') && guard.can('sys:ai:message:send');
  if (!canAsk() || doc.status !== 'ready') return null;
  return <Button type="primary" disabled={disabled} onClick={() => {
    if (canAsk()) navigate('/ai/chat', { state: { knowledge: { id: doc.kb_id, doc_id: doc.id, scope: JSON.stringify(scope) } } });
  }}>问这篇资料</Button>;
}
