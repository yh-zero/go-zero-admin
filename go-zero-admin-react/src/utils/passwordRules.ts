import type { Rule } from 'antd/es/form';

export const newPasswordRules: Rule[] = [
  { required: true, message: '请输入密码' },
  { min: 8, message: '密码至少需要 8 个字符' },
  {
    validator: (_, value: string) => !value || new TextEncoder().encode(value).length <= 72
      ? Promise.resolve() : Promise.reject(new Error('密码最多 72 字节')),
  },
];

export function createPasswordConfirmationRules(passwordField: string): Rule[] {
  return [
    { required: true, message: '请再次输入新密码' },
    ({ getFieldValue }) => ({
      validator: (_, value: string) => !value || value === getFieldValue(passwordField)
        ? Promise.resolve() : Promise.reject(new Error('两次输入的密码不一致')),
    }),
  ];
}
