// 在前端开发服务或预览页面运行。新建浏览器上下文，所有业务请求只访问内存样本。
// oxlint-disable-next-line no-unused-expressions -- Playwright CLI 执行此函数。
async page => {
  const origin = new URL(page.url()).origin;
  const context = await page.context().browser().newContext();
  const menu = (id, title, name) => ({ id, title, name, path: '/sys/' + name,
    component: name, type: 2, parent_id: 0, visible: true, status: 1, sort: id });
  const menus = [menu(1, '角色管理', 'role'), menu(2, '文件管理', 'file'), menu(3, '审计日志', 'audit'),
    { ...menu(4, 'APP用户', 'app_user'), path: '/app/users' }];
  const role = { id: 19, name: '授权回归角色', code: 'role-test', status: 1, menu_ids: [], data_scope: 'SELF' };
  const appUser = { id: 24, account: 'review@example.test', identity_kind: 'email', nickname: '审核样本',
    status: 'pending', review_reason: '', remark: '', avatar_url: '', created_at: 1 };
  const records = [
    { id: 1, username: '创建账号的管理员', module: 'sys/user', action: 'post' },
    { id: 2, username: '登录的管理员', module: 'sys/auth', action: 'login' },
    { id: 3, username: '历史操作人', module: 'sys/user', action: 'legacy-action' },
  ];
  const file = { id: 43, original_name: '清理回归样本.txt', size: 12, mime: 'text/plain',
    visibility: 'private', ref_count: 0, uploader_id: 9, created_at: 1, cleanup_pending: false };
  const writes = [], queries = [], fileQueries = [], reviews = [], unexpected = [];
  let detailReadFails = false, deleteAttempts = 0;
  const check = (condition, message) => { if (!condition) throw new Error(message); };
  const equal = (a, b) => JSON.stringify(a) === JSON.stringify(b);
  try {
    await context.addInitScript(() => localStorage.setItem('go-zero-auth', JSON.stringify({ version: 0, state: {
      token: 'admin-test-access', refreshToken: 'admin-test-refresh', sessionId: 'admin-test-session',
      expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200,
    } })));
    await context.route('**/*', route => {
      const request = route.request(), url = new URL(request.url()), path = url.pathname;
      if (url.origin === origin && !path.startsWith('/v1/')) return route.continue();
      const send = (result, code = 200, msg = '') => route.fulfill({ json: { code, result, msg } });
      if (url.origin === origin && request.method() === 'GET') {
        switch (path) {
          case '/v1/sys/dept/my': return send({ list: [] });
          case '/v1/sys/menu/my': return send({ list: menus });
          case '/v1/sys/menu/my/perm-codes': return send({ codes: ['sys:role:menus', 'sys:file:delete', 'sys:app:user:review'] });
          case '/v1/sys/profile/': return send({ user: { id: 9, username: 'fixture' } });
          case '/v1/sys/role/list': return send({ list: [role], total: 1 });
          case '/v1/sys/role/19': return detailReadFails ? send({}, 503, '暂时无法读取角色') : send({ role });
          case '/v1/sys/menu/tree': return send({ list: menus });
          case '/v1/sys/app/users': {
            const filter = url.searchParams.get('status');
            const list = !filter || filter === appUser.status ? [appUser] : [];
            return send({ total: list.length, list });
          }
          case '/v1/sys/app/users/24': return send({ user: appUser });
          case '/v1/sys/audit/list': {
            const query = url.searchParams;
            queries.push(query);
            const list = records.filter(row => (!query.has('action') || row.action === query.get('action'))
              && (!query.has('module') || row.module === query.get('module')));
            return send({ total: list.length, list });
          }
          case '/v1/sys/file/list': {
            const includePending = url.searchParams.get('include_cleanup_pending') === 'true';
            fileQueries.push(includePending);
            const list = deleteAttempts < 2 && (!file.cleanup_pending || includePending) ? [file] : [];
            return send({ total: list.length, list });
          }
        }
      }
      if (url.origin === origin && path === '/v1/sys/role/19/menus' && request.method() === 'POST') {
        const body = request.postDataJSON();
        writes.push(body);
        if (!equal(body.expected_menu_ids, role.menu_ids)) return send({}, 409, '授权已变化');
        role.menu_ids = body.menu_ids;
        return send({});
      }
      if (url.origin === origin && path === '/v1/sys/file/43' && request.method() === 'DELETE') {
        deleteAttempts++;
        file.cleanup_pending = true;
        return deleteAttempts === 1 ? send({}, 500, '存储清理失败，请重试删除') : send({});
      }
      if (url.origin === origin && path === '/v1/sys/app/users/24/review' && request.method() === 'POST') {
        const decision = request.postDataJSON();
        reviews.push(decision);
        if (decision.expected_status !== appUser.status || decision.expected_reason !== appUser.review_reason) {
          return send({}, 409, '审核决定已变更');
        }
        appUser.status = decision.status;
        appUser.review_reason = decision.reason;
        return send({});
      }
      unexpected.push(request.method() + ' ' + path);
      return route.abort();
    });
    await checkRoleConflict();
    const tab = await context.newPage();
    await checkAuditFilters(tab);
    await checkFileCleanup(tab);
    await checkAppReviewConflict(tab);
    check(!unexpected.length, '发现意外业务请求：' + unexpected.join(', '));
    return 'PASS：角色授权并发冲突、App 审核冲突、审计动作筛选、文件清理重试';
  } finally { await context.close(); }

  async function checkRoleConflict() {
    const open = async tab => {
      await tab.getByRole('button', { name: '功能授权', exact: true }).click();
      const dialog = tab.getByRole('dialog', { name: /功能授权/ });
      await dialog.getByRole('checkbox', { name: /入口：文件管理/ }).waitFor();
      return dialog;
    };
    const [firstTab, lastTab] = await Promise.all([context.newPage(), context.newPage()]);
    await Promise.all([firstTab, lastTab].map(tab => tab.goto(origin + '/sys/role')));
    const [first, last] = await Promise.all([open(firstTab), open(lastTab)]);
    const box = (dialog, title) => dialog.getByRole('checkbox', { name: new RegExp('入口：' + title) });
    const save = dialog => dialog.getByRole('button', { name: '保存授权' });
    await box(first, '文件管理').check();
    await box(last, '审计日志').check();
    const firstSave = firstTab.waitForResponse(response => new URL(response.url()).pathname === '/v1/sys/role/19/menus');
    await save(first).click();
    await firstSave;
    check(equal(role.menu_ids, [2]), '从空授权保存后应有文件管理权限');
    await first.waitFor({ state: 'hidden' });
    await save(last).click();
    const reload = last.getByRole('button', { name: '加载最新授权' });
    await last.getByText('角色授权已被其他管理员修改').waitFor();
    check(equal(writes[1].menu_ids, [3]) && equal(writes[1].expected_menu_ids, []), '后保存者必须提交编辑前看到的授权');
    check(equal(role.menu_ids, [2]) && await save(last).isDisabled(), '冲突后保留先前授权并禁止重提旧草稿');
    detailReadFails = true;
    await reload.click();
    await last.getByText('授权数据暂时不可用', { exact: false }).waitFor();
    check(await save(last).isDisabled(), '重新加载失败时继续阻止保存');
    detailReadFails = false;
    await reload.click();
    await reload.waitFor({ state: 'hidden' });
    check(await box(last, '文件管理').isChecked() && !(await box(last, '审计日志').isChecked()), '显式加载应显示服务端最新授权');
    await box(last, '审计日志').check();
    await save(last).click();
    await last.waitFor({ state: 'hidden' });
    check(equal(role.menu_ids, [2, 3]) && equal(writes[2].expected_menu_ids, [2]), '重新编辑后保留两人的授权');
    const clear = await open(lastTab);
    await box(clear, '文件管理').uncheck();
    await box(clear, '审计日志').uncheck();
    await save(clear).click();
    await clear.waitFor({ state: 'hidden' });
    check(writes.length === 4 && equal(role.menu_ids, []) && equal(writes[3].expected_menu_ids, [2, 3]), '允许从最新集合显式清空授权');
  }

  async function checkAuditFilters(tab) {
    await tab.goto(origin + '/sys/audit');
    const action = tab.getByRole('combobox', { name: '动作', exact: true });
    const submit = () => tab.getByRole('button', { name: /^查\s*询$/ }).click();
    const visible = user => tab.getByRole('cell', { name: user, exact: true }).waitFor();
    await visible('历史操作人');
    for (const [label, value, user] of [['POST', 'post', '创建账号的管理员'], ['登录', 'login', '登录的管理员']]) {
      await action.fill('');
      await action.click();
      await tab.locator('.ant-select-item-option').filter({ hasText: label }).click();
      await submit();
      await visible(user);
      check(queries.at(-1).get('action') === value, '选择动作后应按实际存储值查询：' + value);
    }
    await action.fill('legacy-action');
    await submit();
    await visible('历史操作人');
    check(queries.at(-1).get('action') === 'legacy-action', '应保留手输的历史动作');
    await action.fill('');
    await tab.getByRole('textbox', { name: '模块', exact: true }).fill('sys/user');
    await submit();
    await visible('创建账号的管理员');
    check(!queries.at(-1).has('action') && queries.at(-1).get('module') === 'sys/user', '清空动作应保留其他筛选');
    await tab.getByRole('button', { name: /^重\s*置$/ }).click();
    await visible('登录的管理员');
    check(!(await action.inputValue()) && !(await tab.getByRole('textbox', { name: '模块', exact: true }).inputValue()), '重置应清空筛选');
  }

  async function checkFileCleanup(tab) {
    await tab.goto(origin + '/sys/file');
    const row = tab.getByRole('row').filter({ hasText: file.original_name });
    await row.getByRole('button', { name: '访问链接' }).waitFor();
    const confirm = tab.getByRole('button', { name: /^确\s*定$/ });
    await row.getByRole('button', { name: /^删\s*除$/ }).click();
    await confirm.click();
    await tab.getByText('存储清理失败，请重试删除', { exact: true }).waitFor();
    await row.getByText('待确认清理').waitFor();
    await tab.reload();
    await row.getByText('待确认清理').waitFor();
    check(fileQueries.length >= 3 && fileQueries.every(Boolean), '文件中心首次、删除失败后及整页刷新都应显式查询待清理文件');
    check(deleteAttempts === 1 && !(await row.getByRole('button', { name: '访问链接' }).count()), '刷新后保留待清理记录，禁止获取链接');
    await row.getByRole('button', { name: '重试清理' }).click();
    await confirm.click();
    await row.waitFor({ state: 'hidden' });
    check(deleteAttempts === 2, '重试必须清理同一文件且只提交一次');
  }

  async function checkAppReviewConflict(tab) {
    await tab.goto(origin + '/app/users');
    const filter = tab.locator('.filter-card');
    await filter.getByRole('combobox', { name: '状态' }).click();
    await tab.locator('.ant-select-item-option').filter({ hasText: '待审核' }).click();
    await filter.getByRole('button', { name: /查\s*询/ }).click();
    await tab.getByRole('row').filter({ hasText: appUser.account }).getByRole('button', { name: /审\s*核$/ }).click();
    const dialog = tab.getByRole('dialog', { name: '审核APP用户' });
    const reason = dialog.getByRole('textbox', { name: '审核说明' });
    await reason.fill('后提交的说明');
    // 另一个审核人已提交，此记录将在刷新后离开「待审核」列表。
    appUser.status = 'active';
    appUser.review_reason = '首位审核人的说明';
    const save = dialog.getByRole('button', { name: '保存审核结果' });
    await save.click();
    await tab.getByText('暂无匹配用户').waitFor();
    await dialog.getByText('审核冲突，此次修改未保存').waitFor();
    await dialog.getByText(/最新状态：已启用。审核说明：首位审核人的说明/).waitFor();
    check(reviews.length === 1 && reviews[0].expected_status === 'pending' && reviews[0].expected_reason === '', '必须提交打开弹窗时的审核快照');
    check(await save.isDisabled() && (await reason.inputValue()) === '后提交的说明', '冲突后应保留草稿并禁止再次提交');
    check(appUser.review_reason === '首位审核人的说明', '先保存的审核决定应保留');
  }
}
