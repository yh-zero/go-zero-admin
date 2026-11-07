import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const htmlText = value => value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#x27;');

test('SafeMarkdown 保留分析标签原文并维持现有聊天渲染', async t => {
  const server = await createServer({ configFile: false, envFile: false, logLevel: 'error', optimizeDeps: { noDiscovery: true }, server: { middlewareMode: true, watch: null, ws: false } });
  try {
    const { SafeMarkdown } = await server.ssrLoadModule('/src/pages/ai/SafeMarkdown.tsx');
    const render = content => renderToStaticMarkup(createElement(SafeMarkdown, { content }));

    await t.test('后端转义的标点按文字展示且不会激活内联语法', () => {
      const cases = [
        ['版本1\\.0的a\\-b：五万元 [1]', '版本1.0的a-b：五万元 [1]'],
        ['代码\\`a\\`：值 [1]', '代码`a`：值 [1]'],
        [String.raw`\*\*重点\*\* \[资料\]\(https://example\.test\)`, '**重点** [资料](https://example.test)'],
        [String.raw`\!\[图片\]\(https://example\.test/a\.png\)`, '![图片](https://example.test/a.png)'],
        [String.raw`\<img src=x onerror=evil\(\)\> \&amp;`, '<img src=x onerror=evil()> &amp;'],
        [String.raw`\{\}\[\]\(\)\#\+\-\.\!\|\~\&\_\\`, '{}[]()#+-.!|~&_\\'],
      ];
      for (const [content, expected] of cases) {
        const markup = render(content);
        assert.ok(markup.includes(`>${htmlText(expected)}</div>`), `${content} 应原样展示为 ${expected}，实际 ${markup}`);
        assert.doesNotMatch(markup, /<(?:code|strong|a|img|script)\b/, '转义的标签不能被重新识别为 Markdown 或 HTML');
      }
    });

    await t.test('转义的块标记、代码围栏和表格分隔符保持文字', () => {
      for (const [content, expected] of [
        [String.raw`\# 标题`, '# 标题'],
        [String.raw`\> 引文`, '> 引文'],
        [String.raw`\- 条目`, '- 条目'],
        [String.raw`\* 条目`, '* 条目'],
        ['\\`\\`\\`js\nalert\\(1\\)\n\\`\\`\\`', '```js'],
        ['\\| A \\| B \\|\n--- | ---\n\\| x \\| y \\|', '| A | B |'],
      ]) {
        const markup = render(content);
        assert.ok(markup.includes(`>${htmlText(expected)}</div>`), content);
        assert.doesNotMatch(markup, /<(?:pre|code|blockquote|table)\b|role="heading"|aria-hidden="true">•/, content);
      }
    });

    await t.test('真实表格的转义竖线留在原单元格中', () => {
      const markup = render('| A | B |\n| --- | --- |\n| x\\|y | z |');
      assert.match(markup, /<table>/);
      assert.match(markup, /<td>x\|y<\/td><td>z<\/td>/);
    });

    await t.test('普通 Windows 路径保留反斜杠，代码中的文字也不反转义', () => {
      const markup = render('路径 C:\\Users\\yanghao\\notes.txt 和 `C:\\data\\a\\.txt`');
      assert.ok(markup.includes('路径 C:\\Users\\yanghao\\notes.txt 和 '));
      assert.match(markup, /<code>C:\\data\\a\\\.txt<\/code>/);
    });

    await t.test('现有聊天的强调、链接、代码、围栏、表格和安全边界保持', () => {
      const inline = render('**重点** `代码` [官网](https://example.test)');
      assert.match(inline, /<strong>重点<\/strong>/);
      assert.match(inline, /<code>代码<\/code>/);
      assert.match(inline, /<a href="https:\/\/example\.test" target="_blank" rel="noopener noreferrer">官网<\/a>/);
      const blocks = render('# 标题\n- 条目\n> 引文\n```js\nconst x = 1;\n```\n| A | B |\n| --- | --- |\n| x | y |');
      assert.match(blocks, /role="heading"/);
      assert.match(blocks, /<blockquote>引文<\/blockquote>/);
      assert.match(blocks, /<pre><code>const x = 1;<\/code><\/pre>/);
      assert.match(blocks, /<td>x<\/td><td>y<\/td>/);
      const unsafe = render('<img src=x onerror=evil()> ![图片](https://example.test/a.png) [脚本](javascript:evil()) [本地](file:///tmp)');
      assert.ok(unsafe.includes('&lt;img src=x onerror=evil()&gt;'));
      assert.doesNotMatch(unsafe, /<(?:img|a|script)\b/);
    });
  } finally {
    await server.close();
  }
});
