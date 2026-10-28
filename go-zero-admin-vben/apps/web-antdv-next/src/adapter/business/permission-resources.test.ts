import { readdirSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import {
  missingPermissionApis,
  permissionApiDependencies,
} from './permission-resources';

function required<T>(value: T | undefined): T {
  if (value === undefined) throw new Error('API contract match is incomplete');
  return value;
}

describe('stable business API dependencies', () => {
  it('declares only routes present in backend API contract sources', () => {
    const directory = resolve(
      process.cwd(),
      '../go-zero-admin/application/applet/api/desc/sys_base',
    );
    const registered = new Set<string>();
    for (const name of readdirSync(directory).filter((name) =>
      name.endsWith('.api'),
    )) {
      const source = readFileSync(resolve(directory, name), 'utf8');
      for (const block of source.matchAll(
        /@server\(([\s\S]*?)\)\s*service\s+[\w-]+\s*\{([\s\S]*?)\n\}/g,
      )) {
        const prefix = /prefix:\s*(\S+)/.exec(required(block[1]))?.[1] ?? '';
        for (const route of required(block[2]).matchAll(
          /\b(get|post|put|delete|patch)\s+(\/[^\s(]+)/g,
        ))
          registered.add(
            `${required(route[1]).toUpperCase()} ${prefix}${route[2]}`,
          );
      }
    }
    expect(
      [...Object.values(permissionApiDependencies).flat()].filter(
        (api) => !registered.has(`${api.method} ${api.path}`),
      ),
    ).toEqual([]);
  });
  it('keeps dependency hints after display and route renames without granting APIs', () => {
    const policies: Array<{ method: string; path: string }> = [];
    const missing = missingPermissionApis(
      [
        {
          ID: 1,
          name: 'renamed',
          parentId: 0,
          path: 'x',
          component: '',
          sort: 0,
          hidden: false,
          meta: { title: 'New label', icon: '' },
          menuBtn: [
            {
              ID: 2,
              name: 'renamed',
              desc: '新增',
              permissionKey: 'user:create',
            },
          ],
        },
      ],
      [2],
      policies,
    );
    expect(missing).toEqual([{ button: '新增', api: 'POST /v1/sys/register' }]);
    expect(policies).toEqual([]);
  });
  it('warns about the App avatar save and image upload API permissions separately', () => {
    const menu = {
      ID: 1,
      name: 'app-members-users',
      parentId: 0,
      path: 'users',
      component: '',
      sort: 0,
      hidden: false,
      meta: { title: 'App用户', icon: '' },
      menuBtn: [
        {
          ID: 2,
          name: 'avatar',
          desc: '设置头像',
          permissionKey: 'app-members-users:avatar',
        },
      ],
    };
    expect(missingPermissionApis([menu], [2], [])).toEqual([
      { button: '设置头像', api: 'PUT /v1/sys/app-members/avatar' },
    ]);
    expect(permissionApiDependencies['businessTools:upload']).toEqual([
      { method: 'POST', path: '/v1/sys/base/uploadFileImg' },
    ]);
  });
});
