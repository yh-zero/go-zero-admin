import { useState } from 'react';
import { Button, Card, Form, Input, Space, Table, Tag, Typography } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { fetchDepartmentDirectory } from '../../api/dept';
import type { DepartmentDirectoryMember, GetDepartmentDirectoryReq } from '../../api/types';
import { QueryState } from '../../components/QueryState';
import { useTablePagination } from '../../hooks/useTablePagination';
import { useAuthStore } from '../../stores/auth';
import { queryKeys } from '../../utils/queryKeys';

export default function DirectoryPage() {
  const departmentVersion = useAuthStore(state => state.departmentVersion);
  return <DepartmentDirectory key={departmentVersion} departmentVersion={departmentVersion} />;
}

function DepartmentDirectory({ departmentVersion }: { departmentVersion: number }) {
  const [form] = Form.useForm<Pick<GetDepartmentDirectoryReq, 'keyword'>>();
  const [params, setParams] = useState<GetDepartmentDirectoryReq>({ page: 1, page_size: 20 });
  const query = useQuery({
    queryKey: [queryKeys.directory, departmentVersion, params],
    queryFn: ({ signal }) => fetchDepartmentDirectory(params, signal),
  });
  const data = query.isSuccess ? query.data : undefined;
  const pagination = useTablePagination(params, setParams, query.data?.total ?? 0);

  return <div className="page-stack">
    <Card className="filter-card">
      <Form form={form} name="directory-filter" layout="inline" className="filter-form"
        onFinish={({ keyword }) => setParams(current => ({ ...current, page: 1, keyword: keyword?.trim() || undefined }))}>
        <Form.Item name="keyword" label="姓名"><Input allowClear maxLength={64} placeholder="搜索姓名" /></Form.Item>
        <Form.Item><Space>
          <Button type="primary" htmlType="submit">查询</Button>
          <Button onClick={() => {
            form.resetFields();
            setParams(current => ({ page: 1, page_size: current.page_size }));
          }}>重置</Button>
          <Button aria-label="刷新通讯录" icon={<ReloadOutlined />} loading={query.isFetching} onClick={() => void query.refetch()} />
        </Space></Form.Item>
      </Form>
    </Card>
    <Card className="data-card" title={data?.dept_name ?? '当前部门'}
      extra={<Typography.Text type="secondary">在顶部切换部门，查看该部门的直属成员</Typography.Text>}>
      <QueryState isPending={query.isPending} error={query.error} onRetry={() => query.refetch()} />
      {data && <Table<DepartmentDirectoryMember> rowKey="id" dataSource={data.list}
        loading={query.isFetching} pagination={pagination} locale={{ emptyText: '暂无符合条件的成员' }} columns={[
          { title: '姓名', dataIndex: 'name', render: (name: string) => <span className="table-primary">{name}</span> },
          { title: '部门', key: 'department', render: () => data.dept_name },
          { title: '负责人', dataIndex: 'is_leader', render: (leader: boolean) => leader ? <Tag color="blue">负责人</Tag> : '—' },
        ]} />}
    </Card>
  </div>;
}
