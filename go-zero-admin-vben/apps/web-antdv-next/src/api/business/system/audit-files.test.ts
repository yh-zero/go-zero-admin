import { beforeEach, describe, expect, it, vi } from 'vitest';

import { uploadImage } from '../base';
import { getAuditLogs } from './audit';
import {
  addFileReference,
  deleteFile,
  getFiles,
  getFileURL,
  removeFileReference,
} from './files';

const client = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
}));
vi.mock('#/api/request', () => ({ requestClient: client }));

describe('audit and file HTTP contracts', () => {
  beforeEach(() => vi.clearAllMocks());
  it('sends pagination and time filters as query parameters', async () => {
    const params = {
      pageNo: 2,
      pageSize: 20,
      actorId: 10,
      actorName: 'admin',
      eventType: 'login' as const,
      result: 'failure' as const,
      startTime: '2026-10-02T00:00:00.000Z',
      endTime: '2026-10-02T01:00:00.000Z',
    };
    await getAuditLogs(params);
    expect(client.get).toHaveBeenCalledExactlyOnceWith(
      '/v1/sys/audit/getAuditLogList',
      { params },
    );
    await getFiles({
      pageNo: 1,
      pageSize: 20,
      status: 'deleting',
      name: 'photo',
    });
    expect(client.get).toHaveBeenLastCalledWith('/v1/sys/files/list', {
      params: { pageNo: 1, pageSize: 20, status: 'deleting', name: 'photo' },
    });
  });
  it('uses id for resources and fileId for business references in DELETE bodies', async () => {
    await getFileURL(7);
    expect(client.get).toHaveBeenLastCalledWith('/v1/sys/files/url', {
      params: { id: 7 },
    });
    await deleteFile(7);
    expect(client.delete).toHaveBeenLastCalledWith('/v1/sys/files/resource', {
      data: { id: 7 },
    });
    const reference = { fileId: 7, objectType: 'article', objectId: '42' };
    await addFileReference(reference);
    expect(client.post).toHaveBeenLastCalledWith(
      '/v1/sys/files/reference',
      reference,
    );
    await removeFileReference(reference);
    expect(client.delete).toHaveBeenLastCalledWith('/v1/sys/files/reference', {
      data: reference,
    });
  });
  it('keeps the original upload field and supports explicit private visibility', async () => {
    const file = new File(['image'], 'example.png', { type: 'image/png' });
    await uploadImage(file);
    const [path, publicData, options] = client.post.mock.calls[0]!;
    expect(path).toBe('/v1/sys/base/uploadFileImg');
    expect(publicData.get('file_img')).toBe(file);
    expect(publicData.get('visibility')).toBe('public');
    expect(options.headers['Content-Type']).toBeUndefined();
    await uploadImage(file, 'private');
    expect(client.post.mock.calls[1]![1].get('visibility')).toBe('private');
  });
});
