import type { UserImportRow } from '../../api/types';

export async function readUserImportCsv(file: File): Promise<string> {
  if (!file.name.toLowerCase().endsWith('.csv')) throw new Error('请选择 CSV 文件。');
  if (file.size === 0 || file.size > 65_536) throw new Error('CSV 文件必须大于 0 且不超过 64 KB。');
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(await file.arrayBuffer());
  } catch {
    throw new Error('无法读取文件，请重新选择 UTF-8 编码的 CSV。');
  }
}

export function mergeUserImportResults(preview: UserImportRow[], results: UserImportRow[]): UserImportRow[] {
  const confirmedRows = new Map(results.map(row => [row.row, row]));
  return preview.map(row => confirmedRows.get(row.row) ?? row);
}
