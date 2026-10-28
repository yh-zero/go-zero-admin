import type { ApiResource, Menu } from '#/api/business/system/types';

import { describe, expect, it } from 'vitest';

import {
  homeMenuOptions,
  includeMenuParents,
  parsePolicies,
  permissionChanges,
  replaceGroupSelection,
  splitPolicies,
} from './authorization';
describe('authorization request conversion', () => {
  it('keeps selections outside a filtered group intact', () => {
    expect(replaceGroupSelection([1, 2, 8], [1, 3], [3, 9])).toEqual([2, 8, 3]);
    expect(replaceGroupSelection([1, 2, 8], [1, 3], [])).toEqual([2, 8]);
    expect(replaceGroupSelection([2, 8], [1, 3], [1, 3, 3])).toEqual([
      2, 8, 1, 3,
    ]);
  });
  it('previews unique additions and revocations without treating reordering as a change', () => {
    expect(permissionChanges([1, 2, 3], [3, 1, 4, 4])).toEqual({
      added: [4],
      removed: [2],
      total: 3,
    });
    expect(permissionChanges(['GET /a', 'PUT /b'], [])).toEqual({
      added: [],
      removed: ['GET /a', 'PUT /b'],
      total: 0,
    });
  });
  it('only offers visible home routes including directories with a visible destination', () => {
    const menus = [
      {
        name: 'hiddenParent',
        hidden: true,
        meta: { title: 'hidden' },
        children: [{ name: 'hiddenChild', meta: { title: 'must not appear' } }],
      },
      {
        name: 'emptyFolder',
        meta: { title: 'no visible destination' },
        children: [
          { name: 'hiddenLeaf', hidden: true, meta: { title: 'hidden' } },
        ],
      },
      {
        name: 'folder',
        meta: { title: 'Folder' },
        children: [{ name: 'leaf', meta: { title: 'Leaf' } }],
      },
    ] as Menu[];
    expect(homeMenuOptions(menus)).toEqual([
      { label: 'Folder', value: 'folder' },
      { label: 'Leaf', value: 'leaf' },
    ]);
  });
  it('includes all necessary ancestors and removes duplicates and unknown menu IDs', () => {
    const menus = [
      { ID: 1, children: [{ ID: 2, children: [{ ID: 3 }] }] },
      { ID: 4 },
    ] as Menu[];
    expect(includeMenuParents([3, 3, 4, 99], menus)).toEqual([3, 2, 1, 4]);
    expect(includeMenuParents([], menus)).toEqual([]);
  });
  it('preserves policy rules absent from the API registry', () => {
    const apis = [
      { ID: 5, method: 'GET', path: '/v1/sys/known' },
    ] as ApiResource[];
    expect(
      splitPolicies(
        [
          { method: 'get', path: '/v1/sys/known' },
          { method: 'PUT', path: '/v1/sys/legacy' },
        ],
        apis,
      ),
    ).toEqual({
      ids: [5],
      unknown: [{ method: 'PUT', path: '/v1/sys/legacy' }],
    });
  });
  it('normalizes and deduplicates advanced rules without changing paths', () => {
    expect(
      parsePolicies('get /v1/sys/a\nGET /v1/sys/a\n\n PUT /v1/sys/b'),
    ).toEqual([
      { method: 'GET', path: '/v1/sys/a' },
      { method: 'PUT', path: '/v1/sys/b' },
    ]);
    expect(parsePolicies('')).toEqual([]);
  });
  it('rejects malformed methods and paths before replacing permissions', () => {
    expect(() => parsePolicies('DELETE')).toThrow();
    expect(() => parsePolicies('GET https://example.com')).toThrow();
    expect(() => parsePolicies('BAD /v1/sys/test')).toThrow();
  });
});
