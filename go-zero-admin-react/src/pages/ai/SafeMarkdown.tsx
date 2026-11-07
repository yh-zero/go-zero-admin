import type { ReactNode } from 'react';
import { App, Button } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import styles from './safeMarkdown.module.css';

export function CopyTextButton({ text, label = '复制' }: { text: string; label?: string }) {
  const { message } = App.useApp();
  return <Button type="text" size="small" icon={<CopyOutlined />} onClick={async () => {
    try { await navigator.clipboard.writeText(text); void message.success('已复制'); }
    catch { void message.error('复制失败，请选择文字后复制'); }
  }}>{label}</Button>;
}

// 只创建 React 元素；原始 HTML、图片和非 HTTP 链接始终作为文字保留。
function inline(text: string): ReactNode[] {
  const parts: ReactNode[] = [];
  // 先消费转义标点为文字；先整体反转义会把分析问项重新激活成链接、代码等语法。
  const tokens = /\\[!-/:-@[-`{-~]|`[^`\n]+`|\*\*[^*\n]+\*\*|\[[^\]\n]{1,300}\]\([^\s)\n]{1,2000}\)/g;
  let previous = 0;
  for (const match of text.matchAll(tokens)) {
    const start = match.index;
    if (start > previous) parts.push(text.slice(previous, start));
    const value = match[0];
    if (value.startsWith('\\')) parts.push(value[1]);
    else if (value.startsWith('`')) parts.push(<code key={start}>{value.slice(1, -1)}</code>);
    else if (value.startsWith('**')) parts.push(<strong key={start}>{value.slice(2, -2)}</strong>);
    else {
      const split = value.indexOf('](');
      const url = value.slice(split + 2, -1);
      // 不发起图片或其他后台请求；链接仅在用户点击后导航。
      if (/^https?:\/\//i.test(url) && Array.from(url).every(character => character.charCodeAt(0) > 32 && character.charCodeAt(0) !== 127) && text[start - 1] !== '!') {
        parts.push(<a key={start} href={url} target="_blank" rel="noopener noreferrer">{value.slice(1, split)}</a>);
      } else parts.push(value);
    }
    previous = start + value.length;
  }
  if (previous < text.length) parts.push(text.slice(previous));
  return parts;
}

function cells(line: string): string[] | null {
  const values: string[] = [];
  let start = 0;
  for (const match of line.matchAll(/\\[!-/:-@[-`{-~]|\|/g)) {
    if (match[0] !== '|') continue;
    values.push(line.slice(start, match.index).trim());
    start = match.index + 1;
  }
  if (values.length === 0) return null;
  values.push(line.slice(start).trim());
  if (values[0] === '') values.shift();
  if (values.at(-1) === '') values.pop();
  return values;
}

export function SafeMarkdown({ content }: { content: string }) {
  const lines = content.split(/\r?\n/);
  const blocks: ReactNode[] = [];
  for (let index = 0; index < lines.length;) {
    const line = lines[index]!;
    const key = index;
    if (/^\s*```/.test(line)) {
      const language = line.trim().slice(3).trim();
      const body: string[] = [];
      index++;
      while (index < lines.length && !/^\s*```/.test(lines[index]!)) body.push(lines[index++]!);
      if (index < lines.length) index++;
      const text = body.join('\n');
      blocks.push(<div className={styles.codeBlock} key={key}><div className={styles.codeHeading}><span>{language || '代码'}</span><CopyTextButton text={text} label="复制代码" /></div><pre><code>{text}</code></pre></div>);
      continue;
    }
    const next = lines[index + 1];
    const headers = cells(line);
    if (headers && next && /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$/.test(next)) {
      const rows: { key: number; values: string[] }[] = [];
      index += 2;
      while (index < lines.length && lines[index]!.trim()) {
        const values = cells(lines[index]!);
        if (!values) break;
        rows.push({ key: index++, values });
      }
      blocks.push(<div className={styles.tableWrap} key={key}><table><thead><tr>{headers.map((cell, column) => <th key={`${key}:${column}`}>{inline(cell)}</th>)}</tr></thead><tbody>{rows.map(row => <tr key={row.key}>{headers.map((_, column) => <td key={`${row.key}:${column}`}>{inline(row.values[column] ?? '')}</td>)}</tr>)}</tbody></table></div>);
      continue;
    }
    const heading = /^(#{1,4})\s+(.+)$/.exec(line);
    if (heading) {
      blocks.push(<div role="heading" aria-level={heading[1]!.length + 1} className={styles.heading} key={key}>{inline(heading[2]!)}</div>);
    } else if (/^\s*[-*]\s+/.test(line)) {
      blocks.push(<div className={styles.listLine} key={key}><span aria-hidden="true">•</span><span>{inline(line.replace(/^\s*[-*]\s+/, ''))}</span></div>);
    } else if (/^>\s?/.test(line)) {
      blocks.push(<blockquote key={key}>{inline(line.replace(/^>\s?/, ''))}</blockquote>);
    } else {
      blocks.push(<div className={styles.line} key={key}>{line ? inline(line) : <br />}</div>);
    }
    index++;
  }
  return <div className={styles.markdown}>{blocks}</div>;
}
