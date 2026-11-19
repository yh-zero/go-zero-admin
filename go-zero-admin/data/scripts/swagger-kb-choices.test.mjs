import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

test('知识库轻量选项的公开 Swagger 描述每个元素的完整签名', async () => {
  const swagger = JSON.parse(await readFile(new URL('../api/generated/admin.json', import.meta.url), 'utf8'));
  const schema = swagger.paths['/v1/sys/ai/kbs'].get.responses['200'].schema;
  const choices = schema.properties.choices;
  assert.equal(schema.required?.includes('choices') ?? false, false);
  assert.equal(choices.type, 'array');
  assert.equal(choices.items?.type, 'object');
  assert.deepEqual([...choices.items.required].sort(), ['id', 'name']);
  assert.deepEqual(Object.keys(choices.items.properties).sort(), ['id', 'name']);
  assert.equal(choices.items.properties.id.type, 'integer');
  assert.equal(choices.items.properties.name.type, 'string');
});
