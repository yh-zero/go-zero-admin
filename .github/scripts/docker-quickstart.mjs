import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

const { chromium } = createRequire(import.meta.url)(process.env.PLAYWRIGHT_MODULE || 'playwright-core');
const project = `gozero-smoke-${randomUUID()}`;
const webPort = process.env.GOZERO_DEMO_PORT || '18080';
const origin = `http://127.0.0.1:${webPort}`;
const captchaPath = '/v1/sys/auth/captcha';
const loginPath = '/v1/sys/auth/login';

function compose(args, override) {
  const files = ['-f', 'compose.yaml'];
  if (override) files.push('-f', override);
  execFileSync('docker', ['compose', '-p', project, ...files, ...args], {
    stdio: 'inherit', env: { ...process.env, GOZERO_DEMO_PORT: webPort },
  });
}

async function writeLocalOverride(directory) {
  const original = await readFile('go-zero-admin/docker/etc/admin-rpc.yaml', 'utf8');
  const local = original.replace(/^(Captcha:\r?\n  Enabled: )true\b/m, '$1false');
  assert(local !== original, '找不到默认启用的 Captcha.Enabled');
  const config = join(directory, 'admin.yaml');
  await writeFile(config, local);
  const override = join(directory, 'compose.json');
  await writeFile(override, JSON.stringify({ services: { 'admin-rpc': { volumes: [
    { type: 'bind', source: config, target: '/app/etc/admin.yaml', read_only: true },
  ] } } }));
  return override;
}

async function checkStack(captchaEnabled) {
  // 替换 RPC 容器后，API 和反向代理可能仍在重新连接上游。
  const deadline = Date.now() + 60_000;
  for (;;) {
    try {
      for (const path of ['/ready', captchaPath]) {
        const response = await fetch(origin + path, { signal: AbortSignal.timeout(5000) });
        assert.equal(response.status, 200, `${path} HTTP 状态异常`);
        const { code, result } = await response.json();
        assert.equal(code, 200, `${path} 业务状态异常`);
        if (path === '/ready') {
          assert.equal(result.ready, true, 'API 尚未就绪');
        } else {
          assert.equal(result.enabled, captchaEnabled, '验证码开关未生效');
          assert.equal(Boolean(result.captcha_id), captchaEnabled, '验证码 ID 与开关不符');
          assert(captchaEnabled ? result.image_base64.startsWith('data:image/') : result.image_base64 === '', '验证码图片与开关不符');
        }
      }
      return;
    } catch (error) {
      if (Date.now() >= deadline) throw error;
      await delay(2000);
    }
  }
}

async function checkRequest(page, method, path, action, pageSize) {
  const [response] = await Promise.all([
    page.waitForResponse(response => {
      const url = new URL(response.url());
      return url.origin === origin && url.pathname === path && response.request().method() === method &&
        (!pageSize || url.searchParams.get('page_size') === String(pageSize));
    }),
    action(),
  ]);
  assert.equal(response.status(), 200, `${path} HTTP 状态异常`);
  const { code, result } = await response.json();
  assert.equal(code, 200, `${path} 业务状态异常`);
  return result;
}

async function checkUserList(page) {
  await page.goto(origin);
  await page.waitForURL(`${origin}/login`);
  await page.getByPlaceholder('请输入用户名').fill('admin');
  await page.getByPlaceholder('请输入密码').fill('Admin@123456');
  await checkRequest(page, 'POST', loginPath, () => page.getByRole('button', { name: '登录工作台' }).click());
  await page.waitForURL(`${origin}/dashboard`);
  const users = await checkRequest(page, 'GET', '/v1/sys/user/list', () =>
    page.getByRole('link', { name: /用户管理.*账号与成员信息/ }).click(), 20);
  await page.waitForURL(`${origin}/sys/user`);
  assert(users.list.some(user => user.username === 'admin'), '用户列表接口未返回初始化管理员');
  await page.getByRole('table').getByRole('button', { name: 'admin', exact: true }).waitFor();
}

async function readLoginAudit(page) {
  const audit = await checkRequest(page, 'GET', '/v1/sys/audit/list', () => page.goto(`${origin}/sys/audit`), 20);
  const record = audit.list.find(row => row.user_agent === project && row.path === loginPath && row.result === 1);
  assert(record?.id > 0 && record.created_at > 0 && record.username === 'admin', '未找到本轮浏览器成功登录的审计记录');
  return { id: record.id, createdAt: record.created_at };
}

const directory = await mkdtemp(join(tmpdir(), 'gozero-docker-smoke-'));
let browser;
let started = false;
try {
  const override = await writeLocalOverride(directory);
  console.log(`本轮 Compose 项目 ${project}，入口 ${origin}`);
  browser = await chromium.launch({ executablePath: process.env.PLAYWRIGHT_BROWSER_PATH || undefined });
  const page = await browser.newPage({ userAgent: project, viewport: { width: 1440, height: 900 } });
  started = true;
  compose(['up', '-d', '--build', '--wait']);
  await checkStack(true);
  await page.goto(`${origin}/login`);
  await page.getByRole('img', { name: '登录验证码' }).waitFor();
  await page.goto('about:blank');
  console.log('空卷启动成功，登录页显示默认图形验证码');
  compose(['up', '-d', '--no-deps', '--wait', 'admin-rpc'], override);
  await checkStack(false);
  await checkUserList(page);
  const audit = await readLoginAudit(page);
  console.log(`真实登录与用户列表通过，已生成审计记录 ${audit.id}`);

  await page.goto('about:blank');
  compose(['down']);
  compose(['up', '-d', '--wait']);
  await checkStack(true);
  assert.deepEqual(await readLoginAudit(page), audit, '重启后未找到同一条登录审计记录');
  await checkRequest(page, 'POST', '/v1/sys/auth/logout', () => page.getByRole('button', { name: '退出登录' }).click());
  await page.waitForURL(`${origin}/login`);
  console.log('保留卷重启后验证码已恢复，原会话可读取同一条审计记录，登出成功');
} finally {
  try {
    if (started) compose(['down', '--volumes', '--remove-orphans', '--rmi', 'local']);
  } finally {
    await browser?.close();
    await rm(directory, { recursive: true, force: true });
  }
}
