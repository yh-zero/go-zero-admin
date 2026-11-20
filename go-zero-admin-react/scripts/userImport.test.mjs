import assert from 'node:assert/strict';
import test from 'node:test';
import { createServer } from 'vite';

test('用户导入保留同一份 CSV 的预检结果及确认范围', async t => {
  const server = await createServer({
    configFile: false, envFile: false, logLevel: 'error',
    server: { middlewareMode: true, watch: null, ws: false },
  });
  try {
    const { readUserImportCsv, mergeUserImportResults } =
      await server.ssrLoadModule('/src/pages/user/userImportState.ts');

    await t.test('有效 UTF-8 CSV 保留引号、换行和中文', async () => {
      const content = 'username,dept_id,real_name\r\nexample,31,"李,小明"\r\n';
      assert.equal(await readUserImportCsv(new File(['\uFEFF', content], 'users.CSV')), content);
    });

    await t.test('拒绝空、超限、非 CSV 和无效 UTF-8 文件', async () => {
      for (const file of [
        new File([], 'users.csv'), new File(['x'.repeat(65537)], 'users.csv'),
        new File(['username'], 'users.xlsx'), new File([Uint8Array.of(0xff)], 'users.csv'),
      ]) {
        await assert.rejects(readUserImportCsv(file));
      }
    });

    await t.test('确认明细替换对应行且保留未选择的预检失败行', () => {
      const preview = [
        { row: 2, username: 'example', dept_id: 31, status: 'ready', message: '' },
        { row: 3, username: 'duplicate', dept_id: 31, status: 'invalid', message: '已存在' },
        { row: 4, username: 'other', dept_id: 31, status: 'ready', message: '' },
      ];
      const results = [
        { row: 4, username: 'other', dept_id: 31, status: 'failed', message: '已存在' },
        { row: 2, username: 'example', dept_id: 31, status: 'created', message: '' },
      ];
      assert.deepEqual(mergeUserImportResults(preview, results), [
        { row: 2, username: 'example', dept_id: 31, status: 'created', message: '' },
        { row: 3, username: 'duplicate', dept_id: 31, status: 'invalid', message: '已存在' },
        { row: 4, username: 'other', dept_id: 31, status: 'failed', message: '已存在' },
      ]);
    });
  } finally { await server.close(); }
});
