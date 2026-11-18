import { Alert, Drawer, Typography } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import type { AiSearchKbHit } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useAuthStore } from '../../stores/auth';
import { knowledgeDocumentOptions } from './knowledgeDocumentQueries';
import { AskDocumentButton } from './AskDocumentButton';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import type { AiWorkspaceScope } from './workspaceQueries';
import styles from './chat.module.css';

interface Props { source: AiSearchKbHit; onClose: () => void; askDisabled?: boolean }

function uniqueExcerptStart(content: string, snippet: string) {
  if (!snippet.trim()) return -1;
  const first = content.indexOf(snippet);
  // 从下一字符继续检查，也识别相互重叠的重复片段。
  if (first < 0 || content.indexOf(snippet, first + 1) >= 0) return -1;
  // 分块会裁剪段落空白。另一处仅空白不同的条款也可能是来源，不足以确定位置。
  const compactContent = content.replace(/\p{White_Space}+/gu, '');
  const compactSnippet = snippet.replace(/\p{White_Space}+/gu, '');
  const compactStart = compactContent.indexOf(compactSnippet);
  return compactSnippet && compactContent.indexOf(compactSnippet, compactStart + 1) < 0 ? first : -1;
}

export function ChatSourcePreview({ source, onClose, askDisabled }: Props) {
  const auth = useAuthStore();
  const scope: AiWorkspaceScope = [auth.sessionId, auth.sessionVersion, auth.departmentVersion];
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:kb:list');
  const canRead = guard.can('sys:ai:kb:list');
  const detail = useQuery({ ...knowledgeDocumentOptions(scope, source.kb_id, source.doc_id), enabled: canRead });
  const result = canRead && !detail.isFetching && !detail.error ? detail.data : undefined;
  const current = result?.doc.id === source.doc_id && result.doc.kb_id === source.kb_id ? result : undefined;
  const content = current?.content;
  const start = useMemo(() => content ? uniqueExcerptStart(content, source.snippet) : -1, [content, source.snippet]);
  const snapshotVersion = source.doc_version;
  const [opened, setOpened] = useState(false);
  const highlight = useRef<HTMLElement>(null);
  useEffect(() => {
    if (opened && start >= 0) highlight.current?.scrollIntoView({ block: 'start' });
  }, [opened, start, current]);

  return <Drawer open title="当前原文" placement="right" size={720} rootClassName={styles.sourceDrawer} mask={{ closable: false }} afterOpenChange={setOpened} onClose={onClose}
    extra={current && <Typography.Text type="secondary">{snapshotVersion && snapshotVersion !== current.doc.version
      ? `引用第 ${snapshotVersion} 版 · 当前第 ${current.doc.version} 版`
      : `当前第 ${current.doc.version} 版${snapshotVersion ? '' : ' · 历史版本未记录'}`}</Typography.Text>}>
    {!canRead ? <Alert type="warning" showIcon title="暂无查看原文权限" /> : <>
      <QueryState isPending={detail.isPending || detail.isFetching} error={detail.error} onRetry={() => { if (guard.isCurrent()) void detail.refetch(); }} />
      {result && !current && <Alert type="warning" showIcon title="当前原文与引用的资料不匹配" />}
    </>}
    {current && <div className={styles.sourceHeading}><Typography.Title level={4}>{current.doc.title}</Typography.Title><AskDocumentButton scope={scope} doc={current.doc} disabled={askDisabled} /></div>}
    {start < 0 && (current || !canRead || detail.error || result) && <section aria-label="历史引用摘录">
      {current && <Typography.Paragraph type="secondary">当前原文中无法唯一定位这段引用。</Typography.Paragraph>}
      <Typography.Text strong>当时保存的摘录</Typography.Text>
      <blockquote className={styles.historicalExcerpt}>{source.snippet}</blockquote>
    </section>}
    {current && <div className={styles.sourceDocument} aria-label="当前原文全文">{start < 0 ? current.content : <>
      {current.content.slice(0, start)}<mark ref={highlight}>{source.snippet}</mark>{current.content.slice(start + source.snippet.length)}
    </>}</div>}
  </Drawer>;
}
