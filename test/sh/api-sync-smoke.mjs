// Local HTTP regression. Temporarily changes one API description and restores it.
import assert from 'node:assert/strict';
import fs from 'node:fs';

const base = process.env.GOZERO_TEST_URL || 'http://127.0.0.1:7001';
if (!/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(base)) throw Error('Only local development APIs are supported');
const token = process.env.GOZERO_TEST_TOKEN ||
  (process.argv[2] && JSON.parse(fs.readFileSync(process.argv[2], 'utf8')).accessToken);
if (!token) throw Error('Provide an admin session file or GOZERO_TEST_TOKEN');
const results = [];
async function request(method, path, body, expectFailure = false) {
  const response = await fetch(base + '/v1/sys' + path, {
    method, headers: { Authorization: 'Bearer ' + token, 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(15000),
  });
  const data = await response.json();
  assert.equal(response.status, 200, path + ': HTTP ' + response.status);
  if (expectFailure) {
    assert.notEqual(data.code, 200, path + ': expected business rejection');
    assert.notEqual(data.code, 100003, path + ': session expired');
  } else assert.equal(data.code, 200, path + ': ' + data.message);
  return data.result;
}
const catalog = async () => (await request('GET', '/api/getAllApiList')).apiList;
const normalized = (rows) => rows.map(({ ID, path, method, apiGroup, description }) => ({ ID, path, method, apiGroup, description })).sort((a, b) => a.ID - b.ID);
let original;
try {
  const before = await catalog();
  const preview = await request('GET', '/api/previewSync');
  assert.match(preview.version, /^[a-f0-9]{64}$/);
  assert.ok(Array.isArray(preview.added) && Array.isArray(preview.changed) && Array.isArray(preview.obsolete));
  assert.ok([...preview.added, ...preview.changed].every(({ path }) => !['/v1/sys/me', '/v1/sys/logout', '/v1/sys/changePassword'].includes(path)));
  await request('POST', '/api/applySync', { version: preview.version, keys: [] }, true);
  await request('POST', '/api/applySync', { version: preview.version, keys: ['GET /v1/sys/not-a-real-interface'] }, true);
  assert.deepEqual(normalized(await catalog()), normalized(before));
  results.push({ label: 'Preview contract, personal endpoints excluded, invalid selection causes no writes', status: 'PASS' });

  original = before.find(({ path, method }) => path === '/v1/sys/api/previewSync' && method === 'GET');
  assert.ok(original, 'Run database migrations first');
  const policies = await request('GET', '/casbin/getPathByAuthorityId?authorityId=1');
  await request('PUT', '/api/updateApi', { ...original, description: 'Temporary API sync regression ' + Date.now() });
  const changed = await request('GET', '/api/previewSync');
  const key = 'GET /v1/sys/api/previewSync';
  assert.ok(changed.changed.some((item) => item.key === key));
  await request('POST', '/api/applySync', { version: preview.version, keys: [key] }, true);
  const result = await request('POST', '/api/applySync', { version: changed.version, keys: [key] });
  assert.equal(result.updated, 1); assert.equal(result.added, 0);
  const after = await catalog();
  assert.equal(after.find(({ path, method }) => path === original.path && method === original.method).ID, original.ID);
  assert.deepEqual(normalized(after.filter(({ ID }) => ID !== original.ID)), normalized(before.filter(({ ID }) => ID !== original.ID)));
  assert.deepEqual(await request('GET', '/casbin/getPathByAuthorityId?authorityId=1'), policies);
  await request('POST', '/api/applySync', { version: changed.version, keys: [key] }, true);
  results.push({ label: 'Stale version rejected, selected metadata applied once, IDs/other resources/grants preserved', status: 'PASS' });
} catch (error) {
  results.push({ label: 'API sync HTTP regression', status: 'FAIL', detail: error.message });
} finally {
  if (original) {
    try { await request('PUT', '/api/updateApi', original); results.push({ label: 'Original API description restored', status: 'PASS' }); }
    catch (error) { results.push({ label: 'Restore original API description', status: 'FAIL', detail: error.message }); }
  }
  fs.mkdirSync('test/reports', { recursive: true });
  fs.writeFileSync('test/reports/latest-api-sync-smoke.json', JSON.stringify({ time: new Date().toISOString(), base, results }, null, 2));
  for (const item of results) console.log(item.status + ' ' + item.label + (item.detail ? ': ' + item.detail : ''));
  if (results.some(({ status }) => status === 'FAIL')) process.exitCode = 1;
}
