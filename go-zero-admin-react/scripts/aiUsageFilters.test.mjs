import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('用量筛选使用上海日期并保留范围和身份边界', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const state = await server.ssrLoadModule('/src/pages/ai/usageState.ts');
    await t.test('跨上海午夜及跨月的快捷范围包含首尾日期', () => {
      assert.equal(typeof state.usageDateRange, 'function');
      assert.deepEqual(state.usageDateRange(1, new Date('2026-11-10T16:01:00Z')), { from_day: '2026-11-11', to_day: '2026-11-11' });
      assert.deepEqual(state.usageDateRange(7, new Date('2026-11-01T00:00:00Z')), { from_day: '2026-10-26', to_day: '2026-11-01' });
      assert.deepEqual(state.usageDateRange(30, new Date('2024-03-01T00:00:00Z')), { from_day: '2024-02-01', to_day: '2024-03-01' });
    });
    await t.test('日期缺失、非法、倒序及超过31日均拒绝，31日可查', () => {
      assert.equal(typeof state.usageDateRangeError, 'function');
      assert.equal(state.usageDateRangeError('2026-10-01', '2026-10-31'), undefined);
      assert.equal(state.usageDateRangeError('2024-02-29', '2024-02-29'), undefined);
      for (const [from, to] of [['', '2026-11-01'], ['2026-02-30', '2026-03-01'], ['2026-11-02', '2026-11-01'], ['2026-10-01', '2026-11-01']]) {
        assert.equal(typeof state.usageDateRangeError(from, to), 'string');
      }
    });
    await t.test('提交回第一页，收敛空白条件，本人请求不能携带用户筛选', () => {
      assert.equal(typeof state.usageFilterParams, 'function');
      const values = { from_day: '2026-11-05', to_day: '2026-11-11', source: 'chat', operation: 'chat', status: 'failed', usage_state: 'partial', provider: ' deepseek ', model: ' ', user_id: 9, page: 4, page_size: 99 };
      const actual = state.usageFilterParams(values, false);
      assert.deepEqual(actual, { from_day: '2026-11-05', to_day: '2026-11-11', source: 'chat', operation: 'chat', status: 'failed', usage_state: 'partial', provider: 'deepseek', page: 1, page_size: 20 });
      assert.equal(state.usageFilterParams(values, true).user_id, 9);
      assert.equal(values.page, 4, '不得修改表单对象');
      assert.throws(() => state.usageFilterParams({ from_day: '2026-10-01', to_day: '2026-11-11' }, true), /31/);
    });
    await t.test('重置恢复近7日并清除全部高级条件', () => {
      assert.equal(typeof state.usageFilterDefaults, 'function');
      assert.deepEqual(state.usageFilterDefaults(new Date('2026-11-10T16:01:00Z')), { from_day: '2026-11-05', to_day: '2026-11-11', page: 1, page_size: 20 });
    });
  } finally { await server.close(); }
});
