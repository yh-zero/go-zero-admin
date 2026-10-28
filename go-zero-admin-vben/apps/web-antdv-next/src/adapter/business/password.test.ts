import { describe, expect, it } from 'vitest';

import { isValidPassword, passwordByteLength } from './password';
describe('UTF-8 password policy', () => {
  it('matches the server byte boundaries for ASCII, Chinese and emoji', () => {
    expect(isValidPassword('1234567')).toBe(false);
    expect(isValidPassword('12345678')).toBe(true);
    expect(isValidPassword('a'.repeat(72))).toBe(true);
    expect(isValidPassword('a'.repeat(73))).toBe(false);
    expect(passwordByteLength('密码测试')).toBe(12);
    expect(isValidPassword('中'.repeat(24))).toBe(true);
    expect(isValidPassword('中'.repeat(25))).toBe(false);
    expect(isValidPassword('😀'.repeat(18))).toBe(true);
    expect(isValidPassword('😀'.repeat(19))).toBe(false);
  });
});
