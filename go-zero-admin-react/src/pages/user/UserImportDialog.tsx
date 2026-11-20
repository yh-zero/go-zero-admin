import { useState } from 'react';
import { Alert, Button, Collapse, Flex, Modal, Space, Table, Tag, Typography, Upload } from 'antd';
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { importUsers } from '../../api/user';
import type { DeptInfo, UserImportRow } from '../../api/types';
import { queryKeys } from '../../utils/queryKeys';
import { mergeUserImportResults, readUserImportCsv } from './userImportState';

const templateUrl = 'data:text/csv;charset=utf-8,' + encodeURIComponent('\uFEFFusername,dept_id,real_name\r\n');
const rowStatus = {
  ready: { color: 'blue', label: '可导入' },
  invalid: { color: 'error', label: '预检未通过' },
  created: { color: 'success', label: '已创建' },
  failed: { color: 'error', label: '创建失败' },
} as const;

interface Props { departments: DeptInfo[]; onClose: () => void }

export function UserImportDialog({ departments, onClose }: Props) {
  const client = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [csv, setCsv] = useState('');
  const [fileError, setFileError] = useState('');
  const preview = useMutation({
    mutationFn: async () => {
      if (!file) return;
      const content = await readUserImportCsv(file).catch((error: Error) => {
        setFileError(error.message);
        throw error;
      });
      const response = await importUsers({ csv: content });
      setCsv(content);
      return response;
    },
  });
  const previewRows = preview.data?.rows ?? [];
  const readyRows = previewRows.filter(row => row.status === 'ready');
  const confirmed = useMutation({
    mutationFn: () => importUsers({ csv, confirm: true, row_numbers: readyRows.map(row => row.row) }),
    onSettled: async () => {
      setCsv('');
      // 超时或断线时，已执行的行也可能增加用户列表和部门成员统计。
      await Promise.all([queryKeys.users, queryKeys.depts].map(async key => {
        await client.cancelQueries({ queryKey: [key] });
        await client.invalidateQueries({ queryKey: [key] });
      }));
    },
    onError: () => preview.reset(),
  });
  const busy = preview.isPending || confirmed.isPending;
  const result = confirmed.data;
  const rows = result ? mergeUserImportResults(previewRows, result.rows) : previewRows;

  return <Modal open title="批量导入用户" width={760} onCancel={onClose}
    closable={!busy} keyboard={!busy} mask={{ closable: !busy }}
    footer={<Space>
      <Button onClick={onClose} disabled={busy}>{result ? '完成' : '关闭'}</Button>
      {!result && (!preview.data
        ? <Button type="primary" disabled={!file} loading={preview.isPending} onClick={() => {
          setFileError('');
          confirmed.reset();
          preview.mutate();
        }}>预检文件</Button>
        : <Button type="primary" disabled={readyRows.length === 0 || !csv}
          loading={confirmed.isPending} onClick={() => confirmed.mutate()}>确认导入 {readyRows.length} 人</Button>)}
    </Space>}>
    <Flex vertical gap="middle">
      <Typography.Text type="secondary">
        支持 UTF-8 CSV，最多 100 条、64 KB。填写用户名、部门编号和可选的真实姓名，部门编号 0 表示未分配。
        新账号默认停用。导入后，请由管理员在用户列表中启用账号、按需分配角色，然后重置密码。
      </Typography.Text>
      <Space wrap>
        <Button icon={<DownloadOutlined />} href={templateUrl} download="user-import-template.csv">下载模板</Button>
        <Upload accept=".csv,text/csv" maxCount={1} showUploadList={false} disabled={busy} beforeUpload={(selected) => {
          setFile(selected);
          setCsv('');
          setFileError('');
          preview.reset();
          confirmed.reset();
          return Upload.LIST_IGNORE;
        }}>
          <Button icon={<UploadOutlined />} disabled={busy}>选择 CSV</Button>
        </Upload>
        {file && <Typography.Text>{file.name}</Typography.Text>}
      </Space>
      {departments.length > 0 && <Collapse size="small" items={[{
        key: 'departments',
        label: '查看部门编号',
        children: <Table<DeptInfo> rowKey="id" size="small" dataSource={departments}
          pagination={{ pageSize: 5, showSizeChanger: false }}
          columns={[{ title: '部门', dataIndex: 'name' }, { title: '编号', dataIndex: 'id', width: 120 }]} />,
      }]} />}
      {fileError && <Alert type="error" showIcon title={fileError} />}
      {preview.isError && !fileError && <Alert type="error" showIcon title="预检未完成，请检查 CSV 或重新预检。" />}
      {confirmed.isError && <Alert type="warning" showIcon title="未收到完整导入结果"
        description="部分账号可能已创建。请重新预检，系统会标出已存在的账号。" />}
      {result
        ? <Alert showIcon type={result.created === rows.length ? 'success' : 'warning'}
          title={`导入结果：已创建 ${result.created} 人，其余 ${rows.length - result.created} 人请查看对应行。`} />
        : preview.data && <Alert showIcon type={readyRows.length ? 'info' : 'warning'}
          title={`预检结果：${readyRows.length} 行可导入，${rows.length - readyRows.length} 行未通过。`}
          description="确认时将再次校验通过预检的行。已有账号会保留原资料。" />}
      {rows.length > 0 && <Table<UserImportRow> rowKey="row" size="small" dataSource={rows}
        loading={confirmed.isPending} scroll={{ x: 560 }} pagination={{ pageSize: 10, showSizeChanger: false }}
        columns={[
          { title: 'CSV 行号', dataIndex: 'row', width: 95 },
          { title: '用户名', dataIndex: 'username', width: 150, render: (value: string) => value || '—' },
          { title: '部门编号', dataIndex: 'dept_id', width: 90 },
          { title: '结果', dataIndex: 'status', width: 120, render: (status: UserImportRow['status']) =>
            <Tag color={rowStatus[status].color}>{rowStatus[status].label}</Tag> },
          { title: '说明', dataIndex: 'message', render: (value: string) => value || '—' },
        ]} />}
    </Flex>
  </Modal>;
}
