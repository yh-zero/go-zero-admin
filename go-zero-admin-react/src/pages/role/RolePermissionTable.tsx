import { Card, Checkbox, Empty, Table, Tag, Tooltip, Typography } from 'antd';
import type { MenuInfo } from '../../api/types';
import { MenuExpandToggle, MenuIcon } from '../../components/MenuIcon';
import type { RolePermissionGroups, RolePermissionRow } from './rolePermissions';
import styles from './RolePermissions.module.css';

interface SelectionProps {
  checkedIds: Set<number>;
  disabled: boolean;
  onToggle: (id: number, checked: boolean) => void;
}

function ResourceFlags({ menu }: { menu: MenuInfo }) {
  return <>{menu.type !== 3 && !menu.visible && <Tag>隐藏</Tag>}{menu.status !== 1 && <Tag color="warning">停用</Tag>}</>;
}

function OperationPermissions({ items, checkedIds, disabled, onToggle }: SelectionProps & { items: MenuInfo[] }) {
  return <div className={styles.operations}>{items.map(menu => <Tooltip key={menu.id}
    title={`权限码：${menu.perm_code || '未设置'} · ID ${menu.id}`}>
    <Checkbox className={styles.operation} checked={checkedIds.has(menu.id)} disabled={disabled}
      aria-label={`操作权限：${menu.title}（ID ${menu.id}）`} onChange={event => onToggle(menu.id, event.target.checked)}>
      <span className={styles.operationLabel}>{menu.title || menu.name || `资源 ${menu.id}`}<ResourceFlags menu={menu} /></span>
    </Checkbox>
  </Tooltip>)}</div>;
}

export function RolePermissionTable({ groups, ...selection }: SelectionProps & { groups: RolePermissionGroups }) {
  return <div className={styles.groups}>
    <Table<RolePermissionRow> className={styles.table} rowKey="id" size="middle" pagination={false}
      dataSource={groups.rows} scroll={{ x: 800 }} expandable={{ defaultExpandAllRows: true, indentSize: 20,
        expandIcon: ({ expanded, expandable, record, onExpand }) => <MenuExpandToggle title={record.title} expanded={expanded}
          expandable={expandable} onToggle={event => onExpand(record, event)} />,
      }}
      rowClassName={menu => menu.type === 1 ? styles.directoryRow : styles.pageRow}
      locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无目录或页面资源" /> }} columns={[
        { title: '目录 / 页面', key: 'resource', width: 320, render: (_, menu) => <div className={styles.resource}>
          <MenuIcon menu={menu} variant="badge" className={styles.resourceIcon} />
          <div className={styles.resourceContent}>
            <div className={styles.resourceName}>
              <span className="table-primary">{menu.title || menu.name || `资源 ${menu.id}`}</span>
              <Tag color={menu.type === 2 ? 'blue' : undefined}>{menu.type === 1 ? '目录' : '页面'}</Tag>
              <ResourceFlags menu={menu} />
            </div>
            <Typography.Text type="secondary" className={styles.resourceMeta}>{menu.path || menu.name || `ID ${menu.id}`}</Typography.Text>
          </div>
        </div> },
        { title: '入口权限', key: 'entry', width: 130, render: (_, menu) => <Checkbox className={styles.entry} checked={selection.checkedIds.has(menu.id)}
          disabled={selection.disabled} aria-label={`${menu.type === 1 ? '目录' : '页面'}入口：${menu.title}（ID ${menu.id}）`}
          onChange={event => selection.onToggle(menu.id, event.target.checked)}>{menu.type === 1 ? '目录入口' : '页面入口'}</Checkbox> },
        { title: '操作权限', key: 'operations', render: (_, menu) => menu.type === 2
          ? menu.operations.length > 0 ? <OperationPermissions items={menu.operations} {...selection} />
            : <Typography.Text type="secondary">暂无操作权限</Typography.Text>
          : <Typography.Text type="secondary">—</Typography.Text> },
      ]} />
    {groups.otherOperations.length > 0 && <Card className={styles.otherGroup} size="small" title={`其他操作权限（${groups.otherOperations.length}）`}>
      <Typography.Paragraph type="secondary" className={styles.sectionHint}>以下操作未归属有效页面，可独立配置授权。</Typography.Paragraph>
      <OperationPermissions items={groups.otherOperations} {...selection} />
    </Card>}
    {groups.otherResources.length > 0 && <Card className={styles.otherGroup} size="small" title={`其他资源（${groups.otherResources.length}）`}>
      <Typography.Paragraph type="secondary" className={styles.sectionHint}>以下资源类型暂不识别，现有授权仍可查看和保留。</Typography.Paragraph>
      <OperationPermissions items={groups.otherResources} {...selection} />
    </Card>}
  </div>;
}
