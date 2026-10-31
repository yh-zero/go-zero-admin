export const statusOptions = [
  { value: 1, label: '启用' },
  { value: 2, label: '停用' },
];

export const genderOptions = [
  { value: 0, label: '未知' },
  { value: 1, label: '男' },
  { value: 2, label: '女' },
];

export function formatGender(gender: number): string {
  return genderOptions.find((option) => option.value === gender)?.label ?? '未知';
}
