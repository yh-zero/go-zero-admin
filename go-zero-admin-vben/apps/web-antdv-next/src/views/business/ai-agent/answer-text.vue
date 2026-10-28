<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{ content: string }>();

// Only text formatting is supported. Model-supplied HTML, links and images
// remain text, so rendering never executes HTML or opens external resources.
const escapedCharacters = new Set('\\`*_{}[]()#+.!<>|~-');

function unescapeText(text: string) {
  let result = '';
  for (let index = 0; index < text.length; index++) {
    const next = text[index + 1] ?? '';
    if (text[index] === '\\' && escapedCharacters.has(next)) {
      result += next;
      index++;
    } else {
      result += text[index];
    }
  }
  return result;
}

function closingMarker(text: string, start: number, marker: string) {
  for (let index = start; index < text.length; index++) {
    if (text[index] === '\n') return -1;
    if (
      text[index] === '\\' &&
      escapedCharacters.has(text[index + 1] ?? '')
    ) {
      index++;
    } else if (text.startsWith(marker, index)) {
      return index;
    }
  }
  return -1;
}

const parts = computed(() => {
  const text = props.content;
  const result: { kind: 'code' | 'strong' | 'text'; text: string }[] = [];
  let literal = '';
  for (let index = 0; index < text.length; index++) {
    const next = text[index + 1] ?? '';
    if (text[index] === '\\' && escapedCharacters.has(next)) {
      literal += next;
      index++;
      continue;
    }
    const marker = text.startsWith('**', index)
      ? '**'
      : text[index] === '`'
        ? '`'
        : '';
    const end = marker
      ? closingMarker(text, index + marker.length, marker)
      : -1;
    if (end > index + marker.length) {
      if (literal) result.push({ kind: 'text', text: literal });
      literal = '';
      const content = text.slice(index + marker.length, end);
      result.push({
        kind: marker === '`' ? 'code' : 'strong',
        text: marker === '`' ? content : unescapeText(content),
      });
      index = end + marker.length - 1;
    } else {
      literal += text[index];
    }
  }
  if (literal) result.push({ kind: 'text', text: literal });
  return result;
});
</script>

<template>
  <template v-for="(part, index) in parts" :key="index">
    <code
      v-if="part.kind === 'code'"
      class="bg-accent rounded px-1 font-mono"
      >{{ part.text }}</code>
    <strong v-else-if="part.kind === 'strong'">{{ part.text }}</strong>
    <template v-else>{{ part.text }}</template>
  </template>
</template>
