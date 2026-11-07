import { useEffect, useState } from 'react';
import { Button, Card, Drawer, Empty, Input, Space, Tag } from 'antd';
import { AppstoreOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import type { AiCapabilities, AiToolInfo } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { AIWorkspaceAccess } from './AIWorkspaceAccess';
import { useAiWorkspaceGuard } from './useAiWorkspaceGuard';
import { aiWorkspaceKeys, toolListOptions } from './workspaceQueries';
import type { AiWorkspaceScope } from './workspaceQueries';
import { formatTaskValue } from './taskState';
import { filterToolsByPurpose, getToolAvailability, toolGuides } from './toolGuides';
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
  const list = filterToolsByPurpose(tools.data?.list ?? [], query);
  const selected = tools.data?.list.find(tool => tool.name === selectedName);
  const canCreateWorkflow = guard.can('sys:ai:workflow:list') && guard.can('sys:ai:workflow:create');

  useEffect(() => () => { void client.cancelQueries({ queryKey: aiWorkspaceKeys.tools(scope) }); }, [client, scope]);

  return <Card className={styles.workspace} title={<div><Space wrap><span>工具与能力</span><Tag>工具目录</Tag></Space><p className={styles.subtitle}>按想完成的工作选择能力，在搭建流程时加入使用</p></div>}
    extra={<Button icon={<ReloadOutlined />} onClick={() => { if (guard.isCurrent()) void tools.refetch(); }}>刷新</Button>}>
    <div className={styles.toolbar}><Input allowClear prefix={<SearchOutlined />} aria-label="搜索工具名称或说明" placeholder="搜索用途，例如：分析资料、整理 JSON" value={search} onChange={event => setSearch(event.target.value)} />
      <span className={styles.muted}>{tools.data ? `${list.length} / ${tools.data.list.length} 个工具` : ''}</span></div>
    <QueryState isPending={tools.isPending} error={tools.error} onRetry={() => tools.refetch()} />
    {!tools.isPending && !tools.error && !list.length && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={query ? '没有找到对应工具' : '暂无注册工具'} />}
    {!tools.error && <div className={styles.grid}>{list.map(tool => <ToolCard key={tool.name} tool={tool} capabilities={tools.data?.capabilities} canUse={canCreateWorkflow} onView={() => {
      if (guard.isCurrent()) setSelectedName(tool.name);
    }} />)}</div>}
    <Drawer open={!!selected && !tools.error} title="工具详情" size={520} className={styles.drawer} onClose={() => setSelectedName(null)}
      footer={<Space wrap className={styles.drawerActions}><Button onClick={() => setSelectedName(null)}>关闭</Button>
        {selected && canCreateWorkflow && <Link to={`/ai/workflows/new?tool=${encodeURIComponent(selected.name)}`}><Button type="primary" icon={<PlusOutlined />}>用此工具创建工作流</Button></Link>}
      </Space>}>
      {selected && <ToolDetailContent tool={selected} capabilities={tools.data?.capabilities} />}
    </Drawer>
    <p className={styles.catalogNote}>能力也可在搭建流程时直接选择。配置状态仅说明服务端配置，实际运行结果请在运行记录中查看。</p>
  </Card>;
}

export function ToolCard({ tool, capabilities, canUse, onView, onSelect }: { tool: AiToolInfo; capabilities?: AiCapabilities; canUse: boolean; onView: () => void; onSelect?: () => void }) {
  const guide = toolGuides[tool.name];
  const availability = getToolAvailability(tool.name, capabilities);
  return <article className={styles.toolCard}>
    <div className={styles.cardHeading}><div className={styles.symbol}><AppstoreOutlined /></div><div><h3 className={styles.toolName}>{guide?.title ?? tool.name}</h3><Tag color="blue">{guide?.category ?? '其他能力'}</Tag></div></div>
    <p className={styles.description}>{guide?.purpose || tool.description || '暂无说明'}</p>
    <div className={styles.availability}><Tag color={availability.color}>{availability.label}</Tag><span>{availability.description}</span></div>
    <div className={styles.cardActions}>
      {canUse && (onSelect ? <Button type={tool.name === 'kb_analyze' ? 'primary' : 'default'} icon={<PlusOutlined />} onClick={onSelect}>加入当前流程</Button>
        : <Link to={`/ai/workflows/new?tool=${encodeURIComponent(tool.name)}`}><Button type={tool.name === 'kb_analyze' ? 'primary' : 'default'} icon={<PlusOutlined />}>用于工作流</Button></Link>)}
      <Button type="link" onClick={onView}>看输入输出</Button>
    </div>
  </article>;
}

export function ToolDetailContent({ tool, capabilities }: { tool: AiToolInfo; capabilities?: AiCapabilities }) {
  const guide = toolGuides[tool.name];
  const availability = getToolAvailability(tool.name, capabilities);
  return <div className={styles.detailContent}><div className={styles.symbol}><AppstoreOutlined /></div><h3 className={styles.toolName}>{guide?.title ?? tool.name}</h3>
    <p className={styles.description}>{guide?.purpose || tool.description || '暂无说明'}</p>
    <div className={styles.availability}><Tag color={availability.color}>{availability.label}</Tag><span>{availability.description}</span></div>
    {guide && <><h3>使用示例</h3>
      <h4>示例输入</h4><pre className={styles.code}>{formatTaskValue(guide.input)}</pre>
      <h4>示例输出</h4><pre className={styles.code}>{guide.output}</pre>
      {guide.note && <p className={styles.muted}>{guide.note}</p>}</>}
    <details className={styles.schema}><summary>参数 Schema · 高级信息</summary>
      <p className={styles.muted}>工具标识：{tool.name}。参数按服务端注册定义展示。</p>
      <pre className={styles.code}>{tool.parameters_schema ? formatTaskValue(tool.parameters_schema) : '未提供参数 Schema'}</pre>
    </details>
  </div>;
}
