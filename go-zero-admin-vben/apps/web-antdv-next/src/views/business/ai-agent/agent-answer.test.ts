import type { App } from 'vue';

import { createApp, h, nextTick, ref } from 'vue';

import { afterEach, describe, expect, it } from 'vitest';

import AgentAnswer from './agent-answer.vue';

let app: App | undefined;

function mountAnswer(content: string) {
  const answer = ref(content);
  const container = document.createElement('div');
  document.body.append(container);
  app = createApp(() => h(AgentAnswer, { content: answer.value }));
  app.mount(container);
  return { answer, container };
}

afterEach(() => {
  app?.unmount();
  app = undefined;
  document.body.innerHTML = '';
});

describe('agent answer display', () => {
  it('renders query details as a table, including escaped data', () => {
    const { container } = mountAnswer(
      [
        '查询到 2 条记录。',
        '',
        '### 查询明细',
        '| 名称 | 内容 |',
        '| --- | --- |',
        '| 登录 | 成功 \\| 本人 |',
        '| 路径 | C:\\\\files |',
      ].join('\n'),
    );

    expect(container.querySelector('h3')?.textContent).toBe('查询明细');
    expect(container.querySelectorAll('thead th')).toHaveLength(2);
    expect(container.querySelectorAll('tbody tr')).toHaveLength(2);
    expect(container.querySelectorAll('tbody td')[1]?.textContent).toContain(
      '成功 | 本人',
    );
    expect(container.querySelectorAll('tbody td')[3]?.textContent).toContain(
      'C:\\files',
    );
    expect(container.querySelector('p')?.textContent).toContain(
      '查询到 2 条记录',
    );
  });

  it('keeps query and model HTML, links and images inert', () => {
    const content = [
      '<script>alert(1)</script>',
      '<img src="https://example.invalid/pixel" onerror="alert(1)">',
      '[点击](javascript:alert(1)) ![图片](https://example.invalid/pixel)',
      '',
      '| 数据 |',
      '| --- |',
      '| <svg onload="alert(1)"> |',
    ].join('\n');
    const { container } = mountAnswer(content);

    expect(container.querySelector('script, img, svg, a, iframe')).toBeNull();
    expect(container.textContent).toContain('<script>alert(1)</script>');
    expect(container.textContent).toContain('javascript:alert(1)');
    expect(container.querySelector('td')?.textContent).toContain(
      '<svg onload=',
    );
  });

  it('decodes escaped server data without turning it into markup', () => {
    const { container } = mountAnswer([
      '| 文件 | 原始数据 |',
      '| --- | --- |',
      '| report\\.pdf | \\*\\*literal\\*\\* \\`code\\` \\| C:\\\\files |',
      '| \\#\\# 标题 | \\[链接\\]\\(javascript:alert\\(1\\)\\) \\<img\\> |',
    ].join('\n'));

    expect(container.querySelectorAll('td')[0]?.textContent).toContain('report.pdf');
    expect(container.querySelectorAll('td')[1]?.textContent).toContain(
      '**literal** `code` | C:\\files',
    );
    expect(container.querySelectorAll('td')[2]?.textContent).toContain('## 标题');
    expect(container.querySelectorAll('td')[3]?.textContent).toContain(
      '[链接](javascript:alert(1)) <img>',
    );
    expect(container.querySelector('strong, code, h2, a, img')).toBeNull();
  });

  it('renders lists and code while keeping code literal', () => {
    const { container } = mountAnswer(
      [
        '- **已查询**审计记录',
        '- 文件 `123`',
        '',
        '3. 检查记录',
        '4. 查看结果',
        '',
        '```json',
        '{"name":"<script>","pipe":"a|b"}',
        '```',
      ].join('\n'),
    );

    expect(container.querySelectorAll('ul li')).toHaveLength(2);
    expect(container.querySelector('strong')?.textContent).toBe('已查询');
    expect(container.querySelector('li code')?.textContent).toBe('123');
    expect(container.querySelector('ol')?.getAttribute('start')).toBe('3');
    expect(container.querySelector('pre code')?.textContent).toBe(
      '{"name":"<script>","pipe":"a|b"}',
    );
    expect(container.querySelector('script')).toBeNull();
  });

  it('preserves malformed tables as text instead of hiding data', () => {
    const { container } = mountAnswer(
      [
        '| 名称 | 状态 |',
        '| --- | --- |',
        '| 正常 | 成功 |',
        '| 未匹配列数 |',
        '| 仍应展示 | 原始数据 | 多余列 |',
      ].join('\n'),
    );

    expect(container.querySelectorAll('tbody tr')).toHaveLength(1);
    expect(container.textContent).toContain('| 未匹配列数 |');
    expect(container.textContent).toContain('多余列');
  });

  it('preserves bare pipe lines instead of treating them as a zero-column table', () => {
    const { container } = mountAnswer('|\n|\n|');

    expect(container.querySelector('table')).toBeNull();
    expect(container.textContent).toContain('|\n|\n|');
  });

  it('preserves an unfinished code block and does not parse its table syntax', () => {
    const { container } = mountAnswer('~~~\n| raw | data |\n<script>');

    expect(container.querySelector('table')).toBeNull();
    expect(container.querySelector('pre')?.textContent).toContain(
      '| raw | data |',
    );
    expect(container.querySelector('script')).toBeNull();
  });

  it('shows an explicit message for empty successful answers', () => {
    const { container } = mountAnswer(' \r\n ');

    expect(container.textContent).toContain('没有返回可展示的内容');
  });

  it('preserves unsupported inline formatting as text', () => {
    const { container } = mountAnswer('**\n`\n``');

    expect(container.textContent).toContain('**\n`\n``');
  });

  it('updates displayed data when the task response changes', async () => {
    const { answer, container } = mountAnswer('最初回复');
    answer.value = '| 新数据 |\n| --- |\n| 最新记录 |';
    await nextTick();

    expect(container.textContent).not.toContain('最初回复');
    expect(container.querySelector('td')?.textContent).toContain('最新记录');
  });
});
