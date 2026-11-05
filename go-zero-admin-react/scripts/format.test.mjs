// utils/format.ts 边界值回归。
import test from 'node:test';
import assert from 'node:assert/strict';
import { formatBytes } from '../src/utils/format.ts';

test('formatBytes: 0 字节', () => {
  assert.equal(formatBytes(0), '0 B');
});

test('formatBytes: 负数与非有限值', () => {
  assert.equal(formatBytes(-1), '—');
  assert.equal(formatBytes(NaN), '—');
  assert.equal(formatBytes(Infinity), '—');
});

test('formatBytes: 边界 1023B / 1024B / 1MB', () => {
  assert.equal(formatBytes(1023), '1023 B');
  assert.equal(formatBytes(1024), '1.0 KB');
  assert.equal(formatBytes(1024 * 1024), '1.0 MB');
});

test('formatBytes: 极大值（GB）', () => {
  assert.equal(formatBytes(5 * 1024 * 1024 * 1024), '5.0 GB');
});
