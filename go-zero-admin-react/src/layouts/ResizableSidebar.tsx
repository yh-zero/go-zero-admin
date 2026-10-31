import type { ReactNode } from 'react';
import { Button, Layout } from 'antd';
import { LeftOutlined, RightOutlined } from '@ant-design/icons';
import { SIDEBAR_MAX_WIDTH, SIDEBAR_MIN_WIDTH } from '../stores/layout';
import { useSidebarResize } from './useSidebarResize';

interface Props {
  collapsed: boolean;
  onToggle: () => void;
  children: ReactNode;
}

export function ResizableSidebar({ collapsed, onToggle, children }: Props) {
  const { width, dragging, handleProps } = useSidebarResize(collapsed);

  return <Layout.Sider width={width} collapsedWidth={76} collapsed={collapsed}
    className={`app-sider${dragging ? ' app-sider-resizing' : ''}`}>
    <div className="sidebar-navigation" onDoubleClick={event => {
      if (event.target === event.currentTarget) onToggle();
    }}>{children}</div>
    <Button type="text" className="sidebar-collapse" aria-label={collapsed ? '展开侧栏' : '折叠侧栏'}
      icon={collapsed ? <RightOutlined /> : <LeftOutlined />} onClick={onToggle}>{!collapsed && '收起侧栏'}</Button>
    {!collapsed && <div className="sidebar-resize-handle" role="separator" tabIndex={0}
      aria-label="调整侧栏宽度" aria-orientation="vertical" aria-valuemin={SIDEBAR_MIN_WIDTH}
      aria-valuemax={SIDEBAR_MAX_WIDTH} aria-valuenow={width} aria-valuetext={`${width} 像素`}
      title="拖动调整宽度，双击恢复默认；方向键微调"
      {...handleProps} />}
  </Layout.Sider>;
}
