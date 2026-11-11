import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('工具目录仅在已有可达流程入口时下沉，原菜单授权保持', async () => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { toMenuItems } = await server.ssrLoadModule('/src/layouts/navigation.tsx');
    const { getAccessiblePagePaths } = await server.ssrLoadModule('/src/router/menuRoutes.ts');
    const node = (id, path, visible = 1) => ({ id, parent_id: 0, path, visible, status: 1, type: 2, name: `node${id}`, title: path, sort: id, children: [] });
    const menus = [node(1, '/ai/workflows'), node(2, '/ai/tools')];
    assert.deepEqual(toMenuItems(menus, true).map(item => item.key), ['/ai/workflows']);
    assert.deepEqual(toMenuItems(menus, false).map(item => item.key), ['/ai/workflows', '/ai/tools']);
    assert.deepEqual(toMenuItems([node(2, '/ai/tools')], true).map(item => item.key), ['/ai/tools']);
    assert.deepEqual(toMenuItems([node(1, '/ai/workflows', 0), node(2, '/ai/tools')], true).map(item => item.key), ['/ai/tools']);
    assert.ok(getAccessiblePagePaths(menus).has('/ai/tools'));
    for (const parent of [{ ...node(4, '/hidden', 0), type: 1 }, { ...node(4, '/disabled'), type: 1, status: 0 }]) {
      const nested = [parent, { ...node(1, '/ai/workflows'), parent_id: 4 }, node(2, '/ai/tools')];
      assert.deepEqual(toMenuItems(nested, true).map(item => item.key), ['/ai/tools']);
    }
  } finally { await server.close(); }
});
