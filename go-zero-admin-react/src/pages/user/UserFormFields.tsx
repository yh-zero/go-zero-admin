import { Form, Input, Select } from 'antd';
import type { DeptInfo, RoleInfo } from '../../api/types';
import { newPasswordRules } from '../../utils/passwordRules';
import { genderOptions, statusOptions } from '../../utils/formOptions';
import { FileUploadButton } from '../../components/FileUploadButton';

interface Props { editing: boolean; depts: DeptInfo[]; currentDeptId?: number; currentDeptName?: string; canAssignRoles: boolean; roles: RoleInfo[]; rolesLoading: boolean; currentRoleIds: number[]; avatarUrl?: string; disabled?: boolean; onUploadingChange: (uploading: boolean) => void }

export function UserFormFields({ editing, depts, currentDeptId = 0, currentDeptName, canAssignRoles, roles, rolesLoading, currentRoleIds, avatarUrl, disabled, onUploadingChange }: Props) {
  const options = depts.map((dept) => ({ label: dept.name, value: dept.id }));
  if (currentDeptId > 0 && !depts.some((dept) => dept.id === currentDeptId)) {
    options.push({ label: currentDeptName || `部门 ${currentDeptId}`, value: currentDeptId });
  }
  options.unshift({ label: '未分配', value: 0 });
  return <>
    {editing && <Form.Item name="avatar_file_id" label="头像" className="field-full" extra="上传公开头像后保存用户，或移除现有头像。">
      <FileUploadButton scene="avatar" visibility="public" previewUrl={avatarUrl} disabled={disabled} onUploadingChange={onUploadingChange} />
    </Form.Item>}
    <Form.Item name="username" label="用户名" rules={[{ required: true, whitespace: true }, { max: 64 }]}>
      <Input disabled={editing} autoComplete="off" />
    </Form.Item>
    {!editing && <Form.Item name="password" label="初始密码" rules={newPasswordRules}><Input.Password autoComplete="new-password" /></Form.Item>}
    <Form.Item name="nickname" label="昵称" rules={[{ max: 64 }]}><Input /></Form.Item>
    <Form.Item name="real_name" label="真实姓名" rules={[{ max: 64 }]}><Input /></Form.Item>
    <Form.Item name="email" label="邮箱" rules={[{ type: 'email' }, { max: 128 }]}><Input /></Form.Item>
    <Form.Item name="phone" label="手机号" rules={[{ max: 20 }]}><Input maxLength={20} /></Form.Item>
    <Form.Item name="dept_id" label="部门" extra={editing && !canAssignRoles ? '仅超级管理员可变更用户部门。' : undefined}>
      <Select options={options} showSearch optionFilterProp="label" disabled={editing && !canAssignRoles} />
    </Form.Item>
    <Form.Item name="gender" label="性别"><Select options={genderOptions} /></Form.Item>
    <Form.Item name="status" label="状态" rules={[{ required: true }]}><Select options={statusOptions} /></Form.Item>
    {canAssignRoles && <Form.Item name="role_ids" label="角色" className="field-full" extra="可选择多个角色；清空选择将移除该用户的全部角色。">
      <Select mode="multiple" allowClear showSearch optionFilterProp="label" loading={rolesLoading} placeholder="选择角色"
        options={roles.map(role => ({ value: role.id, label: role.name + (role.status === 1 ? '' : '（停用）'),
          disabled: role.status !== 1 && !currentRoleIds.includes(role.id) }))} />
    </Form.Item>}
    <Form.Item name="remark" label="备注" className="field-full" rules={[{ max: 512 }]}><Input.TextArea rows={3} /></Form.Item>
  </>;
}
