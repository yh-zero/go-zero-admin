import { describe, expect, it } from 'vitest';

import { safeFileURL, validateUpload } from './format';

describe('file resource input and access boundaries', () => {
  it('rejects unsupported, empty, oversized or overlong-name uploads before a request', () => {
    expect(() =>
      validateUpload(new File(['image'], 'good.png', { type: 'image/png' })),
    ).not.toThrow();
    expect(() =>
      validateUpload(
        new File(['<svg/>'], 'bad.svg', { type: 'image/svg+xml' }),
      ),
    ).toThrow('图片');
    expect(() =>
      validateUpload(new File([], 'empty.png', { type: 'image/png' })),
    ).toThrow('不能为空');
    expect(() =>
      validateUpload(
        new File(['x'], `${'x'.repeat(256)}.png`, { type: 'image/png' }),
      ),
    ).toThrow('文件名');
    const big = new File(['x'], 'big.png', { type: 'image/png' });
    Object.defineProperty(big, 'size', { value: 10 * 1024 * 1024 + 1 });
    expect(() => validateUpload(big)).toThrow('10 MB');
  });
  it('allows only HTTP(S) addresses to be rendered or downloaded', () => {
    expect(
      safeFileURL('https://storage.example.test/a.png?Signature=signed'),
    ).toContain('Signature=signed');
    expect(() => safeFileURL('javascript:alert(1)')).toThrow('访问地址');
    expect(() => safeFileURL('data:image/png;base64,image')).toThrow(
      '访问地址',
    );
  });
});
