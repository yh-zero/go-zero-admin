import { beforeEach, describe, expect, it, vi } from 'vitest';

import { previewMenuMove } from './menu';

const mocks = vi.hoisted(() => ({ post: vi.fn() }));
vi.mock('#/api/request', () => ({ requestClient: mocks }));

describe('menu movement preview wire contract', () => {
  beforeEach(() => vi.resetAllMocks());

  it('lets an empty-role or root movement preview render after protobuf returns null collections', async () => {
    mocks.post.mockResolvedValue({
      version: 'bound-preview',
      authorityIds: null,
      ancestorIds: null,
      addedLinks: null,
    });

    const preview = await previewMenuMove(3, 0, 'token');

    expect(preview.authorityIds.join('、')).toBe('');
    expect(preview.ancestorIds.join('、')).toBe('');
    expect(preview.addedLinks).toEqual([]);
    expect(preview.version).toBe('bound-preview');
  });

  it('preserves nonempty role, ancestor and grant changes in a movement preview', async () => {
    mocks.post.mockResolvedValue({
      version: 'bound-preview',
      authorityIds: [11],
      ancestorIds: [42],
      addedLinks: [{ authorityId: 11, menuId: 42 }],
    });

    expect(await previewMenuMove(3, 42, 'token')).toEqual({
      version: 'bound-preview',
      authorityIds: [11],
      ancestorIds: [42],
      addedLinks: [{ authorityId: 11, menuId: 42 }],
    });
  });
});
