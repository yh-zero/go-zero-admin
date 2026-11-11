import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

test('期间用量与今日额度保持各自口径', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const page = await server.ssrLoadModule('/src/pages/ai/UsagePage.tsx');
    const summary = { confirmed_requests: 12, uncertain_requests: 0, known_prompt_tokens: 8000, known_completion_tokens: 253, known_embedding_tokens: 466 };
    await t.test('三项统计保持已确认及已知用量，不再把零未知请求作为第四项指标', () => {
      const markup = renderToStaticMarkup(createElement(page.UsageSummaryCards, { summary }));
      for (const text of ['已确认请求', '已知对话 token', '已知向量化 token', '8,253', '466']) assert.ok(markup.includes(text), text);
      assert.doesNotMatch(markup, /不确定请求/);
    });
    await t.test('统计未取得时保持占位，不能冒充零用量', () => {
      const markup = renderToStaticMarkup(createElement(page.UsageSummaryCards));
      assert.equal((markup.match(/ant-statistic-content-value/g) ?? []).length, 3);
      assert.ok(markup.includes('—'));
    });
    await t.test('今日额度读取policy占用及剩余，不使用期间12次已确认请求推算', () => {
      assert.equal(typeof page.UsageQuotaBar, 'function');
      const markup = renderToStaticMarkup(createElement(page.UsageQuotaBar, { policy: { quota_day: '2026-11-11', daily_request_limit: 1000, used_requests: 2, remaining_requests: 998, unlimited: false, inherited: true } }));
      for (const text of ['今日额度', '998', '1,000', '已占用', '默认额度', '2026-11-11']) assert.ok(markup.includes(text), text);
      assert.doesNotMatch(markup, /已确认请求/);
    });
    await t.test('不限次数单独展示，不构造剩余数或占用比例', () => {
      const markup = renderToStaticMarkup(createElement(page.UsageQuotaBar, { policy: { quota_day: '2026-11-11', daily_request_limit: 0, used_requests: 4, remaining_requests: 0, unlimited: true, inherited: false } }));
      assert.match(markup, /不限次数/);
      assert.doesNotMatch(markup, /ant-progress|剩余/);
    });
    await t.test('请求结果未知和计量不完整分别提示，正常数据不出现异常告警', () => {
      const complete = { uncertain_requests: 0, partial_requests: 0, unknown_requests: 0, invalid_requests: 0 };
      assert.doesNotMatch(renderToStaticMarkup(createElement(page.UsageSummaryAlerts, { summary: complete })), /role="alert"/);
      const unknown = renderToStaticMarkup(createElement(page.UsageSummaryAlerts, { summary: { ...complete, uncertain_requests: 2 } }));
      assert.match(unknown, /2 次请求结果待核对/);
      assert.doesNotMatch(unknown, /次请求用量不完整/);
      const partial = renderToStaticMarkup(createElement(page.UsageSummaryAlerts, { summary: { ...complete, partial_requests: 1, unknown_requests: 2, invalid_requests: 3 } }));
      assert.match(partial, /6 次请求用量不完整/);
      assert.doesNotMatch(partial, /次请求结果待核对/);
    });
  } finally { await server.close(); }
});
