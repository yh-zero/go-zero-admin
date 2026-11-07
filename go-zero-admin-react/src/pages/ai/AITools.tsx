import { useEffect, useState } from 'react';
import { Button, Card, Drawer, Empty, Input, Space, Tag } from 'antd';
import { AppstoreOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import type { AiToolInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, toolListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { formatTaskValue } from './taskState';
import { toolGuides } from './toolGuides';
import styles from './tools.module.css';

export default function AITools() {
  return <AIWorkspaceAccess code="sys:ai:tool:list" title="工具目录">{scope => <AIToolSession scope={scope} />}</AIWorkspaceAccess>;
}

export function AIToolSession({ scope }: { scope: AiWorkspaceScope }) {
  const guard = useAiWorkspaceGuard(scope, 'sys:ai:tool:list');
  const client = useQueryClient();
  const [search, setSearch] = useState('');
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const tools = useQuery({ ...toolListOptions(scope), enabled: guard.can('sys:ai:tool:list') });
  const query = search.trim().toLocaleLowerCase();
  const list = (tools.data?.list ?? []).filter(tool => `${tool.name}\n${tool.description}\n${toolGuides[tool.name]?.title ?? ''}\n${toolGuides[tool.name]?.purpose ?? ''}`.toLocaleLowerCase().includes(query));
  const selected = tools.data?.list.find(tool => tool.name === selectedName);
  const canCreateWorkflow = guard.can('sys:ai:workflow:list') && guard.can('sys:ai:workflow:create');

  useEffect(() => () => { void client.cancelQueries({ queryKey: aiWorkspaceKeys.tools(scope) }); }, [client, scope]);

  return <Card className={styles.workspace} title={<div><Space wrap><span>工具目录</span><Tag>只读目录</Tag></Space><p className={styles.subtitle}>查看可用工具及参数，在工作流中选择使用</p></div>}
    extra={<Button icon={<ReloadOutlined />} onClick={() => { if (guard.isCurrent()) void tools.refetch(); }}>刷新</Button>}>
    <div className={styles.toolbar}><Input allowClear prefix={<SearchOutlined />} aria-label="搜索工具名称或说明" placeholder="搜索工具名称、用途" value={search} onChange={event => setSearch(event.target.value)} />
      <span className={styles.muted}>{tools.data ? `${list.length} / ${tools.data.list.length} 个工具` : ''}</span></div>
    <QueryState isPending={tools.isPending} error={tools.error} onRetry={() => tools.refetch()} />
    {!tools.isPending && !tools.error && !list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={query ? '没有找到对应工具' : '暂无注册工具'} />}
    {!tools.error && <div className={styles.grid}>{list.map(tool => <ToolCard key={tool.name} tool={tool} canUse={canCreateWorkflow} onView={() => {
      if (guard.isCurrent()) setSelectedName(tool.name);
    }} />)}</div>}
    <Drawer open={!!selected && !tools.error} title="工具详情" size={520} className={styles.drawer} onClose={() => setSelectedName(null)}
      footer={<Space wrap className={styles.drawerActions}><Button onClick={() => setSelectedName(null)}>关闭</Button>
        {selected && canCreateWorkflow && <Link to={`/ai/workflows/new?tool=${encodeURIComponent(selected.name)}`}><Button type="primary" icon={<PlusOutlined />}>用此工具创建工作流</Button></Link>}
      </Space>}>
      {selected && <ToolDetailContent tool={selected} />}
    </Drawer>
  </Card>;
}

export function ToolCard({ tool, canUse, onView }: { tool: AiToolInfo; canUse: boolean; onView: () => void }) {
  const guide = toolGuides[tool.name];
  return <article className={styles.toolCard}>
    <div className={styles.symbol}><AppstoreOutlined /></div>
    <h3 className={styles.toolName}>{guide?.title ?? tool.name}</h3><Tag>{tool.name}</Tag><p className={styles.description}>{guide?.purpose ?? tool.description ?? '暂无说明'}</p>
    <div className={styles.cardActions}><Button type="link" onClick={onView}>查看参数</Button>
      {canUse && <Link to={`/ai/workflows/new?tool=${encodeURIComponent(tool.name)}`}><Button icon={<PlusOutlined />}>用于工作流</Button></Link>}
    </div>
  </article>;
}

export function ToolDetailContent({ tool }: { tool: AiToolInfo }) {
  const guide = toolGuides[tool.name];
  return <div className={styles.detailContent}><div className={styles.symbol}><AppstoreOutlined /></div><h3 className={styles.toolName}>{tool.name}</h3>
    <p className={styles.description}>{tool.description || '暂无说明'}</p>
    {guide && <><h3>{guide.title} · 使用示例</h3><p>{guide.purpose}</p>
      <h4>示例输入</h4><pre className={styles.code}>{formatTaskValue(guide.input)}</pre>
      <h4>示例输出</h4><pre className={styles.code}>{guide.output}</pre>
      {guide.note && <p className={styles.muted}>{guide.note}</p>}</>}
    <h3>参数 Schema</h3>
    <pre className={styles.code}>{tool.parameters_schema ? formatTaskValue(tool.parameters_schema) : '未提供参数 Schema'}</pre>
    <p className={styles.muted}>参数按服务端注册定义展示。请选择工作流步骤来使用工具。</p>
  </div>;
}
