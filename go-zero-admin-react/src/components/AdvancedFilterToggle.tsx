import { Button } from 'antd';
import { DownOutlined, UpOutlined } from '@ant-design/icons';

interface AdvancedFilterToggleProps {
  open: boolean;
  activeCount: number;
  onChange: (open: boolean) => void;
}

export function AdvancedFilterToggle({ open, activeCount, onChange }: AdvancedFilterToggleProps) {
  return <Button
    type="link"
    aria-expanded={open}
    icon={open ? <UpOutlined /> : <DownOutlined />}
    onClick={() => onChange(!open)}
  >
    {open ? '收起筛选' : `更多筛选${activeCount ? `（${activeCount}）` : ''}`}
  </Button>;
}
