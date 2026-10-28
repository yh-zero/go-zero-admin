import { beforeEach, describe, expect, it, vi } from 'vitest';

import { resolveBusinessPage } from '../../adapter/business/pages';
import {
  cancelAgentRun,
  createAgentRun,
  getAgentConversations,
  getAgentInfo,
  getAgentMessages,
  getAgentRun,
} from './ai-agent';

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock('#/api/request', () => ({ requestClient: mocks }));

describe('AI Agent HTTP contract', () => {
  beforeEach(() => vi.resetAllMocks());
  it('uses real info and run routes with the exact idempotent submission body', async () => {
    await getAgentInfo();
    const data = {
      conversationId: 'conversation',
      requestId: 'request',
      message: '问题',
    };
    await createAgentRun(data);
    await getAgentRun('run/id');
    await cancelAgentRun('run/id');
    expect(mocks.get.mock.calls).toEqual([
      ['/v1/ai/info'],
      ['/v1/ai/runs/run%2Fid'],
    ]);
    expect(mocks.post.mock.calls).toEqual([
      ['/v1/ai/runs', data],
      ['/v1/ai/runs/run%2Fid/cancel'],
    ]);
  });
  it('defaults history pagination to 1/20 and normalizes nullable repeated values', async () => {
    mocks.get.mockResolvedValue({ items: null, total: 0 });
    expect(await getAgentConversations()).toEqual({ items: [], total: 0 });
    expect(await getAgentMessages('conversation/id')).toEqual({
      items: [],
      total: 0,
    });
    expect(mocks.get.mock.calls).toEqual([
      ['/v1/ai/conversations', { params: { pageNo: 1, pageSize: 20 } }],
      [
        '/v1/ai/conversations/conversation%2Fid/messages',
        { params: { pageNo: 1, pageSize: 20 } },
      ],
    ]);
  });
  it('preserves explicit history pages and maps only the registered business page', async () => {
    mocks.get.mockResolvedValue({ items: [], total: 41 });
    await getAgentConversations({ pageNo: 2, pageSize: 20 });
    await getAgentMessages('conversation', { pageNo: 3, pageSize: 20 });
    expect(mocks.get).toHaveBeenLastCalledWith(
      '/v1/ai/conversations/conversation/messages',
      {
        params: { pageNo: 3, pageSize: 20 },
      },
    );
    expect(resolveBusinessPage('views/business/ai-agent/index.vue')).toBe(
      '/business/ai-agent/index.vue',
    );
    expect(resolveBusinessPage('views/business/ai-agent/unknown.vue')).toBe(
      '/business/access/pending.vue',
    );
  });
});
