import { Form, Select } from 'antd';
import { useQuery } from '@tanstack/react-query';
import { fetchApiResources } from '../../api/resource';
import { queryKeys } from '../../utils/queryKeys';

export function MenuApiSelect() {
  const query = useQuery({ queryKey: [queryKeys.apiResources, { page: 1, page_size: 100 }], queryFn: () => fetchApiResources({ page: 1, page_size: 100 }) });
  const options = (query.data?.list ?? []).map(item => ({ value: item.id, label: `${item.method} ${item.path}${item.title ? `（${item.title}）` : ''}` }));
  return <Form.Item name="api_ids" label="关联 API" extra="授权时同时生成对应的 Casbin 接口策略；未关联的菜单不会隐式获得接口权限。">
    <Select mode="multiple" allowClear showSearch optionFilterProp="label" options={options} loading={query.isPending} placeholder="选择需要授权的接口" />
  </Form.Item>;
}
