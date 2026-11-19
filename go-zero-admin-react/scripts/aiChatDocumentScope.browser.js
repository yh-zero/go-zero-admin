// 在独立浏览器上下文执行。所有业务接口使用本地响应桩。
// oxlint-disable-next-line no-unused-expressions -- Playwright CLI 执行函数表达式。
async page => {
  const origin = new URL(page.url()).origin;
  const rights = ['sys:ai:kb:list', 'sys:ai:conversation:list', 'sys:ai:message:send'];
  const kb = { id: 7, name: '制度资料', description: '', ready_count: 2, pending_count: 0, failed_count: 0, doc_count: 2, chunk_count: 2 };
  const document = (id, title, amount) => ({ doc: { id, kb_id: kb.id, title, status: 'ready', stage: 'ready', version: 3, chunk_count: 1, embedding_model: 'fixture', error: '', created_at: 1, updated_at: 1 }, content: title + '\n住宿每晚限额 ' + amount + ' 元。' });
  const docs = [document(81, '国内差旅制度', 400), document(82, '海外差旅制度', 900)];
  const sessions = new Map(), assigned = new Map();
  const sends = [], errors = [];
  let unavailable = false, interruptNext = false, loseNextDone = false, heldHead;
  const receipts = new Map();
  const check = (value, reason) => { if (!value) throw new Error(reason); };
  page.on('pageerror', error => errors.push(error.message));
  const context = page.context();
  await context.addInitScript(() => {
    if (!localStorage.getItem('go-zero-auth')) localStorage.setItem('go-zero-auth', JSON.stringify({ state: { sessionId: 'document-scope', token: 'local-test', refreshToken: 'local-refresh', expiresAt: Date.now() / 1000 + 3600, refreshExpiresAt: Date.now() / 1000 + 7200 }, version: 0 }));
  });
  await context.route('**/v1/**', async route => {
    const req = route.request(), url = new URL(req.url()), path = url.pathname;
    const json = result => route.fulfill({ json: { code: 200, result } });
    if (path === '/v1/sys/ai/messages/stream' && req.method() === 'POST') {
      const body = req.postDataJSON(); sends.push(body);
      const id = body.conversation_id || assigned.get(body.request_id) || 19 + sessions.size;
      assigned.set(body.request_id, id);
      if (!sessions.has(id)) sessions.set(id, {
        conversation: { id, user_id: 1, title: '差旅额度 ' + id, created_at: 1, updated_at: 1, message_count: 0, pinned: false }, messages: [],
      });
      const {conversation: conv, messages} = sessions.get(id);
      const frame = (event, data) => 'event: ' + event + '\ndata: ' + JSON.stringify(data) + '\n\n';
      const replay = receipts.get(body.request_id);
      if (replay) return route.fulfill({ contentType: 'text/event-stream', body: replay.meta + frame('done', { tokens: 0, usage_complete: true, result: replay.result }) });
      const next = messages.length + 1;
      const meta = frame('meta', { conversation_id: conv.id, user_message_id: next, ai_message_id: next+1 });
      if (sends.length === 1 || interruptNext) {
        interruptNext = false;
        return route.fulfill({ contentType: 'text/event-stream', body: meta });
      }
      const found = docs.find(item => item.doc.id === body.doc_id);
      const base = { status: 'succeeded', error: '', tokens: 0, tool_calls: '[]', created_at: next, kb_id: body.kb_id, doc_id: body.doc_id, grounded: body.grounded };
      const user = { ...base, id: next, role: 'user', content: body.content, sources: [] };
      const snippet = found?.content.split('\n')[1];
      const answer = { ...base, id: next+1, role: 'assistant', content: found ? snippet+' [1]' : '已在整个知识库查询。', sources: found ? [{ kb_id: kb.id, doc_id: found.doc.id, doc_title: found.doc.title, doc_version: 3, chunk_id: 88, snippet, score: 0.9 }] : [] };
      messages.push(user, answer); conv.message_count = messages.length;
      const result = { conversation_id: conv.id, conversation: conv, user_message: user, ai_message: answer };
      receipts.set(body.request_id, {meta, result});
      if (loseNextDone) { loseNextDone = false; return route.fulfill({contentType:'text/event-stream', body:meta}); }
      return route.fulfill({ contentType: 'text/event-stream', body: meta + frame('done', { tokens: 0, usage_complete: true, result }) });
    }
    if (req.method() !== 'GET') return route.abort('blockedbyclient');
    if (path.endsWith('/perm-codes')) return json({ codes: rights });
    if (path === '/v1/sys/menu/my') return json({ list: [['ai_chat','AI 对话','/ai/chat'], ['ai_kb','知识库','/ai/knowledge']].map(([name,title,path],i) => ({ id:i+1,parent_id:0,name,title,path,component:name,type:2,status:1,visible:true,perm_code:'',icon:'',sort:i,created_at:1 })) });
    if (path === '/v1/sys/profile/') return json({ user: { id:1,username:'fixture',nickname:'回归测试',dept_id:0,role_ids:[] } });
    if (path === '/v1/sys/dept/my') return json({ list:[] });
    if (path.endsWith('/kbs/7')) return json({ kb, capabilities:{ retrieval_configured:true,analysis_configured:true } });
    if (path.endsWith('/kbs/7/docs')) return json({ list:docs.map(item=>item.doc), total:2,ready_count:2,pending_count:0,failed_count:0 });
    const selected = docs.find(item => path.endsWith('/kbs/7/docs/'+item.doc.id));
    if (selected) return unavailable ? route.fulfill({ json:{ code:404,msg:'资料不可用' } }) : json({ ...selected, content: url.searchParams.get('meta_only')==='true' ? '' : selected.content });
    if (path.endsWith('/ai/kbs')) return json({ choices:[{id:7,name:kb.name}],total:1 });
    if (path.endsWith('/ai/conversations')) {
      const list = Array.from(sessions.values(), item => item.conversation);
      return json({ list, total:list.length });
    }
    if (path.includes('/ai/conversations/')) {
      if (heldHead && req.frame().page() === page && path.endsWith('/21')) {
        heldHead.reached.resolve(); await heldHead.gate.promise;
      }
      const session = sessions.get(Number(path.split('/').pop()));
      if (session) return json({ ...session, has_more:false, next_before_id:0 });
    }
    return route.abort('blockedbyclient');
  });
  const input = page.getByRole('textbox', { name:'消息内容' }), send = page.getByRole('button',{ name:/发\s*送$/ });
  const scope = page.getByRole('group',{ name:'本次提问资料' });
  const openConversation = id => page.getByRole('button', {name:new RegExp('^差旅额度 '+id+' ')}).click();
  await page.goto(origin+'/ai/knowledge/7?doc_id=82');
  const preview = page.getByRole('dialog',{name:'文档原文'});
  await preview.getByRole('button',{name:'问这篇资料'}).click();
  await scope.getByText('海外差旅制度').waitFor();
  check(sends.length===0,'阅读或打开提问时不自动发送');
  await input.fill('每晚可以报销多少？'); await send.click();
  const retry = page.getByRole('button',{name:'重查本次结果'});
  await retry.waitFor();
  await scope.getByRole('button',{name:'改问整个知识库'}).click();
  await retry.click();
  const citation = page.getByRole('button', { name:'查看引用 [1]', exact:true });
  await citation.waitFor();
  check(JSON.stringify(sends[0])===JSON.stringify(sends[1]) && sends[0].doc_id===82 && sends[0].kb_id===7 && sends[0].grounded===true,'重查须复用原单篇请求');
  const strict = page.getByRole('switch',{name:'仅依据所选资料回答'});
  check(await strict.isVisible() && await strict.isChecked(),'首发重查成功不能覆盖用户随后明确选择的整库范围');
  await citation.click();
  const drawer = page.getByRole('dialog',{name:'当前原文'});
  await drawer.locator('mark').filter({hasText:'住宿每晚限额 900 元。'}).waitFor();
  check(!(await drawer.innerText()).includes('400 元'),'引用打开的是被选制度');
  await drawer.getByRole('button',{name:/关闭|close/i}).click();
  await input.fill('在整库中找相关制度。'); await send.click();
  await page.getByText('已在整个知识库查询。',{exact:true}).waitFor();
  check(sends[2].conversation_id===19 && sends[2].kb_id===7 && !sends[2].doc_id && sends[2].grounded && sends[2].request_id!==sends[1].request_id,'下一问须使用用户最新选择且有新的请求 ID');
  await page.goto(origin+'/ai/knowledge/7?doc_id=82');
  await preview.getByRole('button',{name:'问这篇资料'}).click();
  await scope.getByText('海外差旅制度').waitFor();
  await input.fill('住宿限额是多少？'); await send.click();
  await citation.waitFor();
  await input.fill('那住两晚呢？'); await send.click();
  await page.waitForFunction(() => document.querySelectorAll('[aria-label="查看引用 [1]"]').length===2);
  check(sends[4].doc_id===82 && sends[4].conversation_id===20,'新开的单篇会话追问须保留所选文档');
  await page.evaluate(() => sessionStorage.clear()); await page.reload(); await openConversation(20);
  await scope.getByText('海外差旅制度').waitFor();
  check(sends.length===5,'没有本地选库记录时从服务端消息恢复，不自动提问');
  unavailable=true; await page.reload(); await openConversation(20);
  await scope.getByText('无法读取这篇资料，请核对权限或重试。').waitFor();
  await input.fill('被删除后'); await input.press('Enter');
  check(await send.isDisabled() && sends.length===5,'文档失效后保留范围并阻止发送');
  await scope.getByRole('button',{name:'改问整个知识库'}).click();
  await page.getByRole('switch',{name:'仅依据所选资料回答'}).waitFor();
  await send.click();
  await page.getByText('已在整个知识库查询。',{exact:true}).waitFor();
  check(sends.length===6 && !sends[5].doc_id && sends[5].kb_id===7 && sends[5].grounded===true,'扩大范围只能显式切换到该库');
  await page.evaluate(() => sessionStorage.clear()); await page.reload(); await openConversation(20);
  await page.getByText('制度资料', {exact:true}).waitFor();
  check(await strict.isChecked() && sends.length===6,'扩大后清除本地选择并重开，仍恢复该知识库和严格模式');
  unavailable = false;
  await page.goto(origin+'/ai/knowledge/7');
  await page.getByRole('button',{ name:'国内差旅制度',exact:true }).click();
  await preview.getByRole('button',{ name:'问这篇资料' }).click();
  await scope.getByText('国内差旅制度').waitFor();
  await input.fill('每晚可以报销多少？'); await send.click();
  await page.getByText('住宿每晚限额 400 元。',{ exact:false }).first().waitFor();
  check(sends.length===7 && sends[6].doc_id===81 && sends[6].conversation_id===0,'切换到同库另一份制度应发起对应文档的全新请求');
  const other = await context.newPage();
  try {
    other.on('pageerror', error => errors.push(error.message));
    await other.goto(origin+'/ai/chat');
    await other.getByRole('button', {name:/^差旅额度 21 /}).click();
    await other.getByRole('button',{name:'改问整个知识库'}).click();
    await other.getByRole('textbox',{name:'消息内容'}).fill('另一标签改查整库');
    await other.getByRole('button',{name:/发\s*送$/}).click();
    await other.getByText('已在整个知识库查询。',{exact:true}).waitFor();
    check(sends.length===8 && sends[7].conversation_id===21 && sends[7].kb_id===7 && !sends[7].doc_id && sends[7].grounded,'另一标签确实已发送新的整库范围');
    heldHead = { reached: Promise.withResolvers(), gate: Promise.withResolvers() };
  } finally { await other.close(); }
  // 原标签保留 sessionStorage 和仍然新鲜的查询缓存；延迟最新页，验证此时不能按旧范围发送。
  try {
    await page.bringToFront();
    // 无头浏览器常把两个页都报告为可见，显式交付浏览器的可见性通知。
    await page.evaluate(() => document.dispatchEvent(new Event('visibilitychange',{bubbles:true})));
    await Promise.race([heldHead.reached.promise,page.waitForTimeout(5000).then(()=>{throw new Error('旧标签没有请求最新页');})]);
    const hot = await page.evaluate(async () => {
      const { queryClient } = await import('/src/utils/queryClient.ts');
      const detail = queryClient.getQueryCache().findAll().find(query=>query.queryKey.at(-2)==='messages' && query.queryKey.at(-1)===21);
      return detail && Date.now()-detail.state.dataUpdatedAt < 30_000;
    });
    await input.fill('原标签接着查'); await input.press('Enter');
    check(hot && await send.isDisabled() && sends.length===8,'30 秒内返回可见的旧标签等待最新页时须阻止按缓存范围发送');
  } finally { heldHead.gate.resolve(); heldHead = undefined; }
  await strict.waitFor();
  check(await strict.isChecked() && await scope.count()===0 && sends.length===8,'旧标签带着已确认的单篇记录重开应恢复更新的整库范围');
  await send.click();
  await page.getByText('原标签接着查',{exact:true}).waitFor();
  check(sends.length===9 && sends[8].conversation_id===21 && sends[8].kb_id===7 && !sends[8].doc_id && sends[8].grounded,'旧标签的实际请求须沿用最新整库范围');
  await page.goto(origin+'/ai/knowledge/7?doc_id=82');
  await preview.getByRole('button',{name:'问这篇资料'}).click();
  await scope.getByText('海外差旅制度').waitFor();
  interruptNext=true; await input.fill('撤权后继续提问'); await send.click(); await retry.waitFor();
  await page.evaluate(async codes => (await import('/src/stores/auth.ts')).useAuthStore.getState().setPermCodes(codes.filter(code=>code!=='sys:ai:kb:list')),rights);
  await page.getByText('当前没有查看这篇资料的权限，已暂停发送并保留原范围。').waitFor();
  const resend = page.getByRole('button',{name:'重新发送（新请求）'});
  await input.press('Enter');
  check(await resend.isDisabled() && await retry.isDisabled() && sends.length===10,'单篇撤权后阻止下一问及原范围重查');
  await page.getByRole('button',{name:'改用自动检索'}).click();
  check(await resend.isEnabled() && await retry.isDisabled() && sends.length===10,'显式清空仅恢复新问题发送，旧请求仍需原权限');
  await resend.click();
  await page.getByText('已在整个知识库查询。',{exact:true}).waitFor();
  check(sends.length===11 && !sends[10].doc_id && !sends[10].kb_id && !sends[10].grounded && sends[10].request_id!==sends[9].request_id,'显式切换使用自动检索的新请求');
  await page.reload(); await openConversation(21); await strict.waitFor();
  const injectLegacy = () => page.evaluate(() => {
    const key = 'ai-chat-knowledge:document-scope', saved = JSON.parse(sessionStorage.getItem(key) || '{}');
    saved['[null,21]'] = {kb_id:7,doc_id:81,grounded:true}; sessionStorage.setItem(key,JSON.stringify(saved));
  });
  const conflict = page.getByText('此标签保存的资料范围与最新消息不一致');
  await injectLegacy(); await openConversation(21); await conflict.waitFor();
  await input.fill('旧选择待确认'); await input.press('Enter');
  check(await send.isDisabled() && sends.length===11,'旧格式与服务端冲突时须等待用户取舍');
  await page.getByRole('button',{name:'沿用本标签范围'}).click();
  await page.reload(); await openConversation(21); await scope.getByText('国内差旅制度').waitFor();
  check(await conflict.count()===0,'沿用旧范围后作为新草稿恢复');
  await injectLegacy(); await openConversation(21); await conflict.waitFor();
  await page.getByRole('button',{name:'采用最新消息范围'}).click(); await strict.waitFor();
  check(await strict.isChecked() && await scope.count()===0 && sends.length===11,'可显式采用服务端的整库范围且不自动发送');
  loseNextDone=true; await input.fill('等待重查的严格提问'); await send.click(); await retry.waitFor();
  await strict.click(); await strict.click();
  const later = await context.newPage();
  try {
    await later.goto(origin+'/ai/chat');
    await later.getByRole('button',{name:/^差旅额度 21 /}).click();
    await later.getByRole('combobox',{name:'对话知识库'}).click();
    await later.locator('.ant-select-dropdown:visible').getByText('自动检索我的知识库',{exact:true}).click();
    await later.getByRole('textbox',{name:'消息内容'}).fill('另一标签改为自动检索');
    await later.getByRole('button',{name:/发\s*送$/}).click();
    await later.getByText('另一标签改为自动检索',{exact:true}).waitFor();
  } finally { await later.close(); }
  await openConversation(21); await page.getByText('另一标签改为自动检索',{exact:true}).waitFor();
  await retry.click(); await retry.waitFor({state:'hidden'});
  check(sends.length===14 && JSON.stringify(sends[11])===JSON.stringify(sends[13]) && await strict.isChecked(),'重查旧消息应保留再次明确选择的相同严格范围');
  await input.fill('使用已确认的严格草稿'); await send.click();
  await page.getByText('使用已确认的严格草稿',{exact:true}).waitFor();
  check(sends.length===15 && sends[14].kb_id===7 && sends[14].grounded && !sends[14].doc_id,'下一次实际发送沿用同值的新草稿');
  await strict.click(); loseNextDone=true;
  await input.fill('未改选择的重查'); await send.click(); await retry.waitFor();
  await openConversation(21); await retry.click(); await retry.waitFor({state:'hidden'});
  const confirmed = await page.evaluate(async () => (await import('/src/pages/ai/chatKnowledge.ts')).readChatKnowledge('document-scope',null,21));
  check(sends.length===17 && JSON.stringify(sends[15])===JSON.stringify(sends[16]) && confirmed.confirmedMessageId>0 && !confirmed.pending,'点击当前会话不产生新选择；重查确认后原草稿可跟随更新的服务端范围');
  check(!errors.length, '页面运行错误：'+errors.join('、'));
  return { checks:['原文进入','单篇提问/追问','首发重查与下一问范围独立','引用高亮','丢失本地选库后的恢复','失效文档','显式扩大后重新打开','跨标签热缓存同步前禁发和同步后发送','单篇撤权后显式切换及原请求重查保护','旧格式冲突两种恢复选择','重查保留回到相同值的新草稿'], realBusinessWrites:0 };
}
