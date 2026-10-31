import { useEffect, useState } from 'react';
import { Button, Select, Space, Tooltip } from 'antd';
import { ApartmentOutlined, ReloadOutlined } from '@ant-design/icons';
import { useIsMutating, useQuery } from '@tanstack/react-query';
import { fetchMyDepts } from '../api/dept';
import { useAuthStore } from '../stores/auth';
import { switchDepartment } from '../utils/request';

export function DepartmentSwitcher() {
  const selectedDeptId = useAuthStore(state => state.selectedDeptId);
  const isLoggingOut = useAuthStore(state => state.isLoggingOut);
  const [switching, setSwitching] = useState(false);
  const mutations = useIsMutating();
  const query = useQuery({ queryKey: ['my-depts'], queryFn: fetchMyDepts });
  useEffect(() => {
    if (selectedDeptId !== null && query.isSuccess && !query.data.some(dept => dept.id === selectedDeptId)) {
      void switchDepartment(null).catch(() => undefined);
    }
  }, [query.data, query.isSuccess, selectedDeptId]);
  if (query.isError) return <Tooltip title="部门列表加载失败，点击重试">
    <Button type="text" aria-label="重新加载可选部门" icon={<ReloadOutlined />} onClick={() => void query.refetch()} />
  </Tooltip>;
  const onChange = (value: number | 'default') => {
    const nextId = value === 'default' ? null : value;
    if (nextId !== null && !query.data?.some(dept => dept.id === nextId)) return;
    setSwitching(true);
    void switchDepartment(nextId).catch(() => undefined).finally(() => setSwitching(false));
  };
  return <Space className="department-switcher">
    <Select aria-label="切换部门" value={selectedDeptId ?? 'default'} onChange={onChange} prefix={<ApartmentOutlined />}
      loading={query.isFetching || switching} disabled={query.isPending || switching || isLoggingOut || mutations > 0}
      popupMatchSelectWidth={false} showSearch optionFilterProp="label" options={[
        { value: 'default', label: '默认部门' }, ...(query.data ?? []).map(dept => ({ value: dept.id, label: dept.name })),
      ]} />
  </Space>;
}
