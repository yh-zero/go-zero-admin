<script setup lang="ts">
import { computed } from 'vue';

import AnswerText from './answer-text.vue';

type AnswerBlock =
  | { content: string; level: string; type: 'heading' }
  | { content: string; type: 'code' | 'paragraph' }
  | { headers: string[]; rows: string[][]; type: 'table' }
  | { items: string[]; ordered: boolean; start: number; type: 'list' };

const props = defineProps<{ content: string }>();

function tableCells(line: string): string[] | undefined {
  const text = line.trim();
  const cells: string[] = [];
  let cell = '';
  let hasPipe = false;
  for (let index = 0; index < text.length; index++) {
    const character = text[index];
    const next = text[index + 1];
    if (character === '\\' && (next === '|' || next === '\\')) {
      cell += character + next;
      index++;
    } else if (character === '|') {
      hasPipe = true;
      cells.push(cell.trim());
      cell = '';
    } else {
      cell += character;
    }
  }
  cells.push(cell.trim());
  if (text.startsWith('|')) cells.shift();
  if (cell === '' && text.endsWith('|')) cells.pop();
  return hasPipe ? cells : undefined;
}

function tableHeader(lines: string[], index: number) {
  const headers = tableCells(lines[index] ?? '');
  const divider = tableCells(lines[index + 1] ?? '');
  return headers?.length &&
    divider?.length === headers.length &&
    divider.every((cell) => /^:?-{3,}:?$/.test(cell))
    ? headers
    : undefined;
}

const fencePattern = /^ {0,3}(`{3,}|~{3,})/;
const headingPattern = /^ {0,3}(#{1,6})\s+(.+)$/;
const listPattern = /^ {0,3}(?:([-+*])|(\d+)[.)])\s+(.+)$/;

const blocks = computed(() => {
  const lines = props.content.replace(/\r\n?/g, '\n').split('\n');
  const result: AnswerBlock[] = [];
  let index = 0;
  while (index < lines.length) {
    const line = lines[index] ?? '';
    if (!line.trim()) {
      index++;
      continue;
    }
    const fence = fencePattern.exec(line)?.[1];
    if (fence) {
      const code: string[] = [];
      index++;
      while (index < lines.length) {
        const closing = lines[index]?.trim() ?? '';
        if (
          closing.length >= fence.length &&
          [...closing].every((character) => character === fence[0])
        ) {
          index++;
          break;
        }
        code.push(lines[index] ?? '');
        index++;
      }
      result.push({ content: code.join('\n'), type: 'code' });
      continue;
    }
    const headers = tableHeader(lines, index);
    if (headers) {
      const rows: string[][] = [];
      index += 2;
      while (index < lines.length) {
        const row = tableCells(lines[index] ?? '');
        if (!row || row.length !== headers.length) break;
        rows.push(row);
        index++;
      }
      result.push({ headers, rows, type: 'table' });
      continue;
    }
    const heading = headingPattern.exec(line);
    if (heading) {
      result.push({
        content: heading[2] ?? '',
        level: `h${heading[1]?.length}`,
        type: 'heading',
      });
      index++;
      continue;
    }
    const list = listPattern.exec(line);
    if (list) {
      const ordered = !!list[2];
      const items: string[] = [];
      while (index < lines.length) {
        const item = listPattern.exec(lines[index] ?? '');
        if (!item || !!item[2] !== ordered) break;
        items.push(item[3] ?? '');
        index++;
      }
      result.push({
        items,
        ordered,
        start: Math.max(1, Number(list[2]) || 1),
        type: 'list',
      });
      continue;
    }
    const paragraph = [line];
    index++;
    while (index < lines.length) {
      const next = lines[index] ?? '';
      if (
        !next.trim() ||
        fencePattern.test(next) ||
        headingPattern.test(next) ||
        listPattern.test(next) ||
        tableHeader(lines, index)
      )
        break;
      paragraph.push(next);
      index++;
    }
    result.push({ content: paragraph.join('\n'), type: 'paragraph' });
  }
  return result;
});
</script>

<template>
  <div class="min-w-0 space-y-3 break-words" data-agent-answer>
    <p v-if="!blocks.length" class="text-muted-foreground text-sm">
      任务已完成，但没有返回可展示的内容。请尝试重新提问。
    </p>
    <template v-for="(block, index) in blocks" :key="index">
      <div
        v-if="block.type === 'table'"
        class="max-w-full overflow-x-auto rounded border"
      >
        <table class="w-full border-collapse text-left text-sm">
          <thead class="bg-accent">
            <tr>
              <th
                v-for="(header, column) in block.headers"
                :key="column"
                class="border-b p-2 align-top font-semibold"
                scope="col"
              >
                <AnswerText :content="header" />
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, rowIndex) in block.rows" :key="rowIndex">
              <td
                v-for="(cell, column) in row"
                :key="column"
                class="border-b p-2 align-top"
              >
                <AnswerText :content="cell" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <pre
        v-else-if="block.type === 'code'"
        class="bg-accent max-w-full overflow-x-auto rounded p-3 text-sm"
      ><code>{{ block.content }}</code></pre>
      <component
        :is="block.level"
        v-else-if="block.type === 'heading'"
        class="font-semibold"
      >
        <AnswerText :content="block.content" />
      </component>
      <component
        :is="block.ordered ? 'ol' : 'ul'"
        v-else-if="block.type === 'list'"
        class="space-y-1 pl-5"
        :class="block.ordered ? 'list-decimal' : 'list-disc'"
        :start="block.ordered ? block.start : undefined"
      >
        <li v-for="(item, itemIndex) in block.items" :key="itemIndex">
          <AnswerText :content="item" />
        </li>
      </component>
      <p v-else class="whitespace-pre-wrap">
        <AnswerText :content="block.content" />
      </p>
    </template>
  </div>
</template>
