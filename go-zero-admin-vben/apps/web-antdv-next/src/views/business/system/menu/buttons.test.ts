import { describe, expect, it } from 'vitest';

import { changedMenuButtons, prepareMenuButtons } from './buttons';

describe('menu button definitions', () => {
  it('preserves existing IDs and parent relations while normalizing input', () => {
    const existing = {
      ID: 12,
      sysBaseMenuID: 4,
      name: 'update ',
      desc: ' 修改 ',
    };
    expect(
      prepareMenuButtons([existing, { name: 'create', desc: '新增' }]),
    ).toEqual([
      { ...existing, name: 'update', desc: '修改' },
      { name: 'create', desc: '新增' },
    ]);
    expect(existing.name).toBe('update ');
  });
  it('rejects duplicate permission codes and invalid or blank labels before submission', () => {
    expect(() =>
      prepareMenuButtons([
        { name: 'create', desc: '新增' },
        { name: 'create ', desc: '添加' },
      ]),
    ).toThrow('重复');
    expect(() =>
      prepareMenuButtons([{ name: 'menu:create', desc: '新增' }]),
    ).toThrow('标识');
    expect(() => prepareMenuButtons([{ name: 'create', desc: ' ' }])).toThrow(
      '名称',
    );
  });
  it('warns for deleted or renamed permission codes but not display-only edits', () => {
    const original = [
      { ID: 1, name: 'create', desc: '新增' },
      { ID: 2, name: 'delete', desc: '删除' },
    ];
    expect(
      changedMenuButtons(original, [{ ID: 1, name: 'create', desc: '添加' }]),
    ).toEqual([original[1]]);
    expect(
      changedMenuButtons(original, [
        { ID: 1, name: 'add', desc: '新增' },
        original[1]!,
      ]),
    ).toEqual([original[0]]);
    expect(
      changedMenuButtons(original, [
        ...original,
        { name: 'update', desc: '修改' },
      ]),
    ).toEqual([]);
  });
});
