import { describe, expect, it } from 'vitest';

import { adaptMenus } from './menu';
import { businessPages, resolveBusinessPage } from './pages';

describe('business page registry', () => {
  it('resolves every editable component to an existing application view', () => {
    const views = import.meta.glob('../../views/**/*.vue');
    expect(new Set(businessPages.map((page) => page.value)).size).toBe(
      businessPages.length,
    );
    for (const page of businessPages) {
      expect(resolveBusinessPage(page.value)).toBe(page.component);
      expect(views).toHaveProperty(`../../views${page.component}`);
    }
  });
  it('keeps unregistered or external components on the pending page', () => {
    expect(resolveBusinessPage('https://example.com/component.vue')).toBe(
      '/business/access/pending.vue',
    );
    expect(resolveBusinessPage('toString')).toBe(
      '/business/access/pending.vue',
    );
  });
  it('prevents backend menus from replacing the authenticated account route', () => {
    expect(() =>
      adaptMenus(
        [{ name: 'BusinessAccount', path: '/custom-account' }],
        '',
        '888',
      ),
    ).toThrow('冲突');
    expect(() =>
      adaptMenus([{ name: 'accountMenu', path: '/account' }], '', '888'),
    ).toThrow('冲突');
    expect(() =>
      adaptMenus(
        [{ name: 'accountMenu', path: '/account/password' }],
        '',
        '888',
      ),
    ).toThrow('冲突');
  });
});
